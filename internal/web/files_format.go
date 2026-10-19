package web

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/xml"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"strings"
)

func (d *designStore) inspect(ctx context.Context, v *storedDesign) (string, error) {
	f, err := designOpen(d.dir, "staging", v.Blob, os.O_RDONLY)
	if err != nil {
		return "", err
	}
	defer f.Close()
	var head [512]byte
	n, e := f.Read(head[:])
	if e != nil && !errors.Is(e, io.EOF) {
		return "", e
	}
	data := head[:n]
	kind := http.DetectContentType(data)
	switch {
	case kind == "image/png" || kind == "image/jpeg":
		if err = d.imagePreview(ctx, v, f); err != nil {
			return "", err
		}
		return kind, nil
	case len(data) >= 16 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP":
		if int64(binary.LittleEndian.Uint32(data[4:8]))+8 != v.Size {
			return "", designFail(400, "FORMAT", "Повреждённый WebP")
		}
		if len(data) < 20 || (string(data[12:16]) != "VP8 " && string(data[12:16]) != "VP8L" && string(data[12:16]) != "VP8X") {
			return "", designFail(400, "FORMAT", "Повреждённый WebP")
		}
		return "image/webp", nil
	case bytes.HasPrefix(data, []byte("%PDF-")):
		return "application/pdf", nil
	case bytes.HasPrefix(data, []byte("PK\x03\x04")) || bytes.HasPrefix(data, []byte("PK\x05\x06")):
		return "application/zip", nil
	case strings.HasPrefix(kind, "text/html"):
		return "text/html", nil
	default:
		if _, err = f.Seek(0, io.SeekStart); err != nil {
			return "", err
		}
		// SVG stays opaque. Read only a bounded prefix to identify its root;
		// no active document is rendered, expanded, or fully buffered.
		decoder := xml.NewDecoder(io.LimitReader(f, 64<<10))
		for {
			if err = ctx.Err(); err != nil {
				return "", err
			}
			token, e := decoder.Token()
			if e != nil {
				return "", designFail(400, "FORMAT", "Неподдерживаемый или повреждённый формат")
			}
			switch el := token.(type) {
			case xml.StartElement:
				if el.Name.Local == "svg" {
					return "image/svg+xml", nil
				}
				return "", designFail(400, "FORMAT", "Неподдерживаемый формат")
			case xml.CharData:
				if strings.TrimSpace(string(el)) != "" {
					return "", designFail(400, "FORMAT", "Неподдерживаемый формат")
				}
			}
		}
	}
}

// Raster decoding is serialized and bounded before Decode. Original bytes are
// attachment-only; the preview always comes from a fresh JPEG encoding.
func (d *designStore) imagePreview(ctx context.Context, v *storedDesign, f *os.File) error {
	select {
	case d.previews <- struct{}{}:
		defer func() { <-d.previews }()
	case <-ctx.Done():
		return ctx.Err()
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	config, format, err := image.DecodeConfig(f)
	if err != nil || config.Width < 1 || config.Height < 1 || config.Width > 8192 || config.Height > 8192 || int64(config.Width)*int64(config.Height) > 16*1024*1024 {
		return designFail(400, "FORMAT", "Изображение повреждено или превышает 16 мегапикселей / 8192 пикселя на сторону")
	}
	if format != "png" && format != "jpeg" {
		return designFail(400, "FORMAT", "Неподдерживаемое изображение")
	}
	if _, err = f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	decoded, decodedFormat, err := image.Decode(f)
	if err != nil || decodedFormat != format {
		return designFail(400, "FORMAT", "Изображение повреждено")
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if !d.reservePreview(ctx, v) {
		return nil
	}
	name := v.Blob + ".preview"
	preview, err := designOpen(d.dir, "staging", name, os.O_WRONLY|os.O_CREATE|os.O_EXCL)
	if err != nil {
		return err
	}
	limited := &designPreviewWriter{to: preview, left: maxPreview}
	err = jpeg.Encode(limited, decoded, &jpeg.Options{Quality: 80})
	if err != nil {
		preview.Close()
		if e := designRemove(d.dir, "staging", name); e != nil {
			return e
		}
		if errors.Is(err, errPreviewLimit) {
			return nil
		}
		return err
	}
	if err = preview.Sync(); err != nil {
		preview.Close()
		return err
	}
	if err = preview.Close(); err != nil {
		return err
	}
	v.Preview = name
	v.PreviewSize = maxPreview - limited.left
	return nil
}

func (d *designStore) reservePreview(ctx context.Context, v *storedDesign) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	used, reserved, err := d.totals(ctx)
	if err != nil || used+reserved-maxDesignFile+v.Size+maxPreview > designQuota {
		return false
	}
	if err = d.disk(ctx, maxPreview); err != nil {
		return false
	}
	_, err = d.db.ExecContext(ctx, `UPDATE design_assets SET reserved=? WHERE asset_id=? AND status='pending'`, v.Size+maxPreview, v.AssetID)
	return err == nil
}

var errPreviewLimit = fmt.Errorf("preview exceeds bounded size")

type designPreviewWriter struct {
	to   io.Writer
	left int64
}

func (w *designPreviewWriter) Write(p []byte) (int, error) {
	if int64(len(p)) > w.left {
		return 0, errPreviewLimit
	}
	n, err := w.to.Write(p)
	w.left -= int64(n)
	return n, err
}
