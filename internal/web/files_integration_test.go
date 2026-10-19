package web

import (
	"agent-board/internal/core"
	"bytes"
	"context"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// This builds the actual UI multipart order, including its optional epic_key,
// while streaming the file from a reader instead of allocating the file in RAM.
func (f *designFixture) uploadStream(t *testing.T, id string, in designUpload, data io.Reader) (DesignAsset, error) {
	t.Helper()
	if id == "" {
		var err error
		id, err = f.s.issueAction(context.Background(), f.b, "upload")
		if err != nil {
			t.Fatal(err)
		}
	}
	var prefix bytes.Buffer
	writer := multipart.NewWriter(&prefix)
	for _, field := range []struct{ k, v string }{{"csrf", f.b.CSRF}, {"action_id", id}, {"title", in.Title}, {"scope", in.Scope}, {"scope_key", in.ScopeKey}, {"epic_key", in.EpicKey}, {"design_id", in.DesignID}} {
		if err := writer.WriteField(field.k, field.v); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := writer.CreateFormFile("file", in.Filename); err != nil {
		t.Fatal(err)
	}
	start := append([]byte(nil), prefix.Bytes()...)
	prefix.Reset()
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("POST", "/web/designs", io.MultiReader(bytes.NewReader(start), data, bytes.NewReader(prefix.Bytes())))
	r.Header.Set("Content-Type", writer.FormDataContentType())
	return f.s.uploadDesign(&designDeadlineWriter{httptest.NewRecorder()}, r, f.b)
}

func TestDesignActualMultipartExactSizeAndCleanup(t *testing.T) {
	f := designFixtureNew(t)
	html := []byte("<!DOCTYPE html><title>large</title>")
	for _, extra := range []int64{0, 1} {
		source := io.MultiReader(bytes.NewReader(html), io.LimitReader(designZeroReader{}, maxDesignFile-int64(len(html))+extra))
		asset, err := f.uploadStream(t, "", designInput(), source)
		if extra == 0 {
			if err != nil || asset.Size != maxDesignFile {
				t.Fatalf("exact multipart limit: %+v %v", asset, err)
			}
		} else {
			designRequireStatus(t, err, 413)
		}
	}
	status, err := f.s.DesignStorageStatus(context.Background())
	if err != nil || status.Used != maxDesignFile || status.Reserved != 0 {
		t.Fatalf("failed upload leaked quota: %+v %v", status, err)
	}
}

func TestDesignActualMultipartScopeAndOptionalEpic(t *testing.T) {
	f := designFixtureNew(t)
	ctx := context.Background()
	epic, err := f.board.Create(ctx, f.b.actor(), core.CreateInput{RequestID: "scope-epic", Type: "epic", Title: "Epic"})
	if err != nil {
		t.Fatal(err)
	}
	other, err := f.board.Create(ctx, f.b.actor(), core.CreateInput{RequestID: "scope-other", Type: "epic", Title: "Other"})
	if err != nil {
		t.Fatal(err)
	}
	task, err := f.board.Create(ctx, f.b.actor(), core.CreateInput{RequestID: "scope-task", Type: "task", EpicKey: epic.Key, Role: "developer", Title: "Task", Description: "Scope test"})
	if err != nil {
		t.Fatal(err)
	}
	in := designInput()
	in.Scope = "task"
	in.ScopeKey = task.Key
	in.EpicKey = epic.Key
	asset, err := f.uploadStream(t, "", in, strings.NewReader("<!DOCTYPE html>"))
	if err != nil || asset.ScopeKey != task.Key {
		t.Fatalf("valid task+epic: %+v %v", asset, err)
	}
	in.EpicKey = other.Key
	_, err = f.uploadStream(t, "", in, strings.NewReader("<!DOCTYPE html>"))
	designRequireStatus(t, err, 400)
	in.Scope = "epic"
	in.ScopeKey = task.Key
	in.EpicKey = ""
	_, err = f.uploadStream(t, "", in, strings.NewReader("<!DOCTYPE html>"))
	designRequireStatus(t, err, 400)
	in.Scope = "task"
	in.ScopeKey = "T-unknown"
	_, err = f.uploadStream(t, "", in, strings.NewReader("<!DOCTYPE html>"))
	designRequireStatus(t, err, 404)
	in = designInput()
	asset, err = f.uploadStream(t, "", in, strings.NewReader("<!DOCTYPE html>"))
	if err != nil || asset.ScopeKey != f.b.Session.Project {
		t.Fatalf("empty optional epic/project key: %+v %v", asset, err)
	}
}

func TestDesignFormatFailureRotatesIDAndUncertaintyPreservesID(t *testing.T) {
	f := designFixtureNew(t)
	ctx := context.Background()
	var raster bytes.Buffer
	if err := png.Encode(&raster, image.NewNRGBA(image.Rect(0, 0, 4, 4))); err != nil {
		t.Fatal(err)
	}
	id, err := f.s.issueAction(ctx, f.b, "upload")
	if err != nil {
		t.Fatal(err)
	}
	in := designInput()
	in.Filename = "design.png"
	_, err = f.upload(t, id, in, raster.Bytes()[:len(raster.Bytes())/2])
	designRequireStatus(t, err, 400)
	r := httptest.NewRequest("POST", "/web/designs", nil)
	r.PostForm = url.Values{"action_id": {id}, "title": {in.Title}, "scope": {in.Scope}, "epic_key": {""}}
	f.s.designPageError(httptest.NewRecorder(), r, f.b, err)
	nextID := f.render.Page.Actions["upload"]
	if nextID == id || nextID == "" || f.render.Page.Draft["title"] != in.Title {
		t.Fatal("format failure did not preserve draft and rotate action")
	}
	if _, err = f.upload(t, nextID, in, raster.Bytes()); err != nil {
		t.Fatalf("corrected upload rejected: %v", err)
	}
	uncertain := designFail(503, "STORAGE", "Unknown result")
	f.s.designPageError(httptest.NewRecorder(), r, f.b, uncertain)
	if f.render.Page.Actions["upload"] != id {
		t.Fatal("uncertain outcome discarded replay ID")
	}
}

func TestDesignImageBoundsBeforeFullDecodeAndDownloadOnlySVG(t *testing.T) {
	f := designFixtureNew(t)
	var raster bytes.Buffer
	if err := png.Encode(&raster, image.NewNRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	oversized := append([]byte(nil), raster.Bytes()...)
	binary.BigEndian.PutUint32(oversized[16:20], 9000)
	binary.BigEndian.PutUint32(oversized[29:33], crc32.ChecksumIEEE(oversized[12:29]))
	in := designInput()
	in.Filename = "large.png"
	_, err := f.upload(t, "", in, oversized)
	designRequireStatus(t, err, 400)
	for _, kind := range []string{"svg", "pdf", "zip"} {
		var data []byte
		switch kind {
		case "svg":
			data = []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)
		case "pdf":
			data = []byte("%PDF-1.7\nopaque")
		case "zip":
			data = []byte("PK\x03\x04opaque")
		}
		in.Filename = "design." + kind
		asset, e := f.upload(t, "", in, data)
		if e != nil || asset.PreviewURL != "" {
			t.Fatalf("download-only %s: %+v %v", kind, asset, e)
		}
	}
}

func TestDesignUploadConcurrencyAndOversizedHeaders(t *testing.T) {
	f := designFixtureNew(t)
	d := f.s.designs
	d.uploads <- struct{}{}
	d.uploads <- struct{}{}
	_, err := f.upload(t, "", designInput(), []byte("<!DOCTYPE html>"))
	designRequireStatus(t, err, 429)
	<-d.uploads
	<-d.uploads
	id, err := f.s.issueAction(context.Background(), f.b, "upload")
	if err != nil {
		t.Fatal(err)
	}
	prefix := "--boundary\r\nContent-Disposition: form-data; name=\"csrf\"\r\n\r\n" + f.b.CSRF + "\r\n--boundary\r\nContent-Disposition: form-data; name=\"action_id\"\r\n\r\n" + id + "\r\n--boundary\r\nContent-Disposition: form-data; name=\"" + strings.Repeat("x", 100<<10) + "\"\r\n\r\nvalue\r\n--boundary--\r\n"
	counter := &designReadCounter{Reader: strings.NewReader(prefix)}
	r := httptest.NewRequest(http.MethodPost, "/web/designs", counter)
	r.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
	_, err = f.s.uploadDesign(&designDeadlineWriter{httptest.NewRecorder()}, r, f.b)
	if err == nil || counter.read > 64<<10 {
		t.Fatalf("unbounded multipart header read %d %v", counter.read, err)
	}
}

func TestDesignRestartKeepsReadyVersions(t *testing.T) {
	f := designFixtureNew(t)
	asset, err := f.upload(t, "", designInput(), []byte("<!DOCTYPE html><title>persist</title>"))
	if err != nil {
		t.Fatal(err)
	}
	if err = f.s.Close(); err != nil {
		t.Fatal(err)
	}
	next, err := New(f.board, Options{Origin: "https://accord.example", DataDir: f.dir, Renderer: f.render})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { next.Close() })
	got, err := next.GetDesign(context.Background(), f.b.actor(), asset.AssetID)
	if err != nil || got.SHA256 != asset.SHA256 || got.Version != 1 || got.DesignID != asset.DesignID {
		t.Fatalf("restart lost immutable metadata: %+v %v", got, err)
	}
}
