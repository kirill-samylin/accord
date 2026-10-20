package web

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
)

func (s *Server) uploadDesign(w http.ResponseWriter, r *http.Request, b *browserSession) (DesignAsset, error) {
	if err := designDeadline(w); err != nil {
		return DesignAsset{}, err
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxDesignFile+(64<<10))
	budget := &designBodyBudget{ReadCloser: r.Body, remaining: 64 << 10}
	r.Body = budget
	reader, err := r.MultipartReader()
	if err != nil {
		return DesignAsset{}, designFail(400, "VALIDATION", "Ожидается форма загрузки файла")
	}
	csrf, err := designField(reader, "csrf")
	if err != nil {
		return DesignAsset{}, err
	}
	if !designEqual(csrf, b.CSRF) {
		return DesignAsset{}, designFail(403, "CSRF", "Обновите форму загрузки")
	}
	request, err := designField(reader, "action_id")
	if err != nil {
		return DesignAsset{}, err
	}
	if err = s.checkAction(r.Context(), b, request, "upload"); err != nil {
		return DesignAsset{}, err
	}
	if err = s.designActor(r.Context(), b.actor()); err != nil {
		return DesignAsset{}, err
	}
	in, part, err := designParseUpload(reader)
	r.PostForm = url.Values{"action_id": {request}, "title": {in.Title}, "scope": {in.Scope}, "scope_key": {in.ScopeKey}, "design_id": {in.DesignID}, "epic_key": {in.EpicKey}}
	if err != nil {
		return DesignAsset{}, err
	}
	if in.Scope == "project" {
		if in.ScopeKey != "" && in.ScopeKey != b.Session.Project {
			return DesignAsset{}, designFail(400, "VALIDATION", "Неверный проект")
		}
		in.ScopeKey = b.Session.Project
	}
	if err = s.designScope(r.Context(), b.actor(), in.Scope, in.ScopeKey, true); err != nil {
		return DesignAsset{}, err
	}
	if err = s.designEpic(r.Context(), b.actor(), in); err != nil {
		return DesignAsset{}, err
	}
	d := s.designs
	select {
	case d.uploads <- struct{}{}:
		defer func() { <-d.uploads }()
	default:
		return DesignAsset{}, designFail(429, "BUSY", "Одновременно загружаются два файла. Повторите позже")
	}
	_ = d.cleanup(r.Context())
	metadata, err := json.Marshal(in)
	if err != nil {
		return DesignAsset{}, err
	}
	sum := sha256.Sum256(metadata)
	v := storedDesign{DesignAsset: DesignAsset{Title: in.Title, Scope: in.Scope, ScopeKey: in.ScopeKey, DesignID: in.DesignID, Filename: in.Filename, UploaderPrincipal: b.Principal.ID, UploaderOwner: b.Principal.Owner}, RequestID: request, MetadataHash: hex.EncodeToString(sum[:])}
	var replay bool
	err = s.guarded(r.Context(), true, b.Session.SessionID, func() error { var e error; v, replay, e = d.reserve(r.Context(), v); return e })
	if err != nil {
		return DesignAsset{}, err
	}
	accepted := replay
	defer func() {
		if !accepted {
			_ = d.abort(v)
		}
	}()
	var target io.Writer = io.Discard
	var f *os.File
	if !replay {
		f, err = designOpen(d.dir, "staging", v.Blob, os.O_WRONLY|os.O_CREATE|os.O_EXCL)
		if err != nil {
			return DesignAsset{}, err
		}
		defer f.Close()
		target = f
	}
	budget.remaining = maxDesignFile + (64 << 10) - budget.read
	digest, size, err := d.stream(r.Context(), target, part)
	if err != nil {
		return DesignAsset{}, err
	}
	budget.remaining = 64 << 10
	if p, e := reader.NextPart(); !errors.Is(e, io.EOF) || p != nil {
		return DesignAsset{}, designFail(400, "VALIDATION", "Файл должен быть последним и единственным")
	}
	if err = s.designActor(r.Context(), b.actor()); err != nil {
		return DesignAsset{}, err
	}
	canonical := append(append([]byte(nil), metadata...), []byte("\x00"+digest)...)
	if _, err = s.bindAction(r.Context(), b, request, "upload", canonical); err != nil {
		return DesignAsset{}, err
	}
	if replay {
		if err = s.guarded(r.Context(), true, b.Session.SessionID, func() error { return nil }); err != nil {
			return DesignAsset{}, err
		}
		if v.SHA256 != digest || v.Size != size {
			return DesignAsset{}, designFail(409, "CONFLICT", "Повтор запроса содержит другой файл")
		}
		if err = s.completeAction(r.Context(), request, v.AssetID); err != nil {
			return DesignAsset{}, err
		}
		return designMetadata(v, true), nil
	}
	v.Size = size
	v.SHA256 = digest
	if err = f.Sync(); err != nil {
		return DesignAsset{}, err
	}
	if err = f.Close(); err != nil {
		return DesignAsset{}, err
	}
	v.MediaType, err = d.inspect(r.Context(), &v)
	if err != nil {
		return DesignAsset{}, err
	}
	if err = designPublish(d.dir, v.Blob); err != nil {
		return DesignAsset{}, err
	}
	if v.Preview != "" {
		if err = designPublish(d.dir, v.Preview); err != nil {
			return DesignAsset{}, err
		}
	}
	if err = s.designActor(r.Context(), b.actor()); err != nil {
		return DesignAsset{}, err
	}
	err = s.guarded(r.Context(), true, b.Session.SessionID, func() error { var e error; v, e = d.finish(r.Context(), v); return e })
	if err != nil {
		return DesignAsset{}, err
	}
	accepted = true
	if err = s.completeAction(r.Context(), request, v.AssetID); err != nil {
		return DesignAsset{}, err
	}
	return designMetadata(v, true), nil
}

func (d *designStore) stream(ctx context.Context, to io.Writer, from io.Reader) (string, int64, error) {
	h := sha256.New()
	out := io.MultiWriter(to, h)
	buf := make([]byte, 64<<10)
	var size, checked int64
	for {
		if err := ctx.Err(); err != nil {
			return "", size, err
		}
		n, err := from.Read(buf)
		if n > 0 {
			size += int64(n)
			if size > maxDesignFile {
				return "", size, designFail(413, "FILE_TOO_LARGE", "Файл превышает 50 МиБ")
			}
			if _, e := out.Write(buf[:n]); e != nil {
				return "", size, designFail(507, "STORAGE", "Не удалось сохранить файл")
			}
			if size-checked >= 1<<20 {
				if e := d.disk(ctx, 0); e != nil {
					return "", size, e
				}
				checked = size
			}
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			var tooBig *http.MaxBytesError
			if errors.As(err, &tooBig) {
				return "", size, designFail(413, "FILE_TOO_LARGE", "Слишком большая форма загрузки")
			}
			return "", size, err
		}
	}
	if size == 0 {
		return "", 0, designFail(400, "FORMAT", "Файл пуст")
	}
	return hex.EncodeToString(h.Sum(nil)), size, nil
}
