package web

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func designReservation(i int) storedDesign {
	return storedDesign{DesignAsset: DesignAsset{Scope: "project", ScopeKey: "design-test", Title: "Test", Filename: "opaque.zip", UploaderPrincipal: "test", UploaderOwner: "test"}, RequestID: fmt.Sprintf("request-%d", i), MetadataHash: "metadata"}
}

func TestDesignQuotaAtomicReservationsAndDiskFloor(t *testing.T) {
	f := designFixtureNew(t)
	d := f.s.designs
	ctx := context.Background()
	d.freeDisk = func(string) (uint64, error) { return diskFloor + (2 << 30), nil }
	var wg sync.WaitGroup
	results := make(chan error, 24)
	for i := range 24 {
		wg.Add(1)
		go func() { defer wg.Done(); _, _, err := d.reserve(ctx, designReservation(i)); results <- err }()
	}
	wg.Wait()
	close(results)
	accepted, rejected := 0, 0
	for err := range results {
		if err == nil {
			accepted++
		} else {
			designRequireStatus(t, err, 413)
			rejected++
		}
	}
	if accepted != 20 || rejected != 4 {
		t.Fatalf("quota admitted %d rejected %d", accepted, rejected)
	}
	status, err := f.s.DesignStorageStatus(ctx)
	if err != nil || status.Reserved != 20*maxDesignFile || status.Used != 0 {
		t.Fatalf("quota totals: %+v %v", status, err)
	}
	if err = d.recover(ctx); err != nil {
		t.Fatal(err)
	}
	d.freeDisk = func(string) (uint64, error) { return diskFloor + 75<<20, nil }
	if _, _, err = d.reserve(ctx, designReservation(100)); err != nil {
		t.Fatal(err)
	}
	_, _, err = d.reserve(ctx, designReservation(101))
	designRequireStatus(t, err, 507)
	if err = d.recover(ctx); err != nil {
		t.Fatal(err)
	}
	d.freeDisk = func(string) (uint64, error) { return 0, errors.New("statfs unavailable") }
	_, _, err = d.reserve(ctx, designReservation(102))
	designRequireStatus(t, err, 507)
}

func TestDesignRecoveryPendingOrphanAndReadyIntegrity(t *testing.T) {
	f := designFixtureNew(t)
	ctx := context.Background()
	d := f.s.designs
	pending, _, err := d.reserve(ctx, designReservation(1))
	if err != nil {
		t.Fatal(err)
	}
	staged, err := designOpen(d.dir, "staging", pending.Blob, os.O_CREATE|os.O_EXCL|os.O_WRONLY)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = staged.WriteString("partial upload"); err != nil {
		t.Fatal(err)
	}
	staged.Close()
	if err = designPublish(d.dir, pending.Blob); err != nil {
		t.Fatal(err)
	}
	orphan, err := designID()
	if err != nil {
		t.Fatal(err)
	}
	orphanFile, err := designOpen(d.dir, "staging", orphan, os.O_CREATE|os.O_EXCL|os.O_WRONLY)
	if err != nil {
		t.Fatal(err)
	}
	orphanFile.Close()
	if err = d.recover(ctx); err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{"staging", "blobs"} {
		entries, e := os.ReadDir(filepath.Join(d.dir, part))
		if e != nil || len(entries) != 0 {
			t.Fatalf("recovery %s: %v %v", part, entries, e)
		}
	}
	status, err := f.s.DesignStorageStatus(ctx)
	if err != nil || status.Used != 0 || status.Reserved != 0 {
		t.Fatalf("released quota: %+v %v", status, err)
	}
	asset, err := f.upload(t, "", designInput(), []byte("<!DOCTYPE html><title>ready</title>"))
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(d.dir, "blobs", asset.AssetID), []byte("tampered"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = d.recover(ctx); err == nil {
		t.Fatal("corrupt ready file did not fail startup")
	}
}

func TestDesignRejectsPathsAndSymlinks(t *testing.T) {
	f := designFixtureNew(t)
	for _, name := range []string{"../x", "/etc/passwd", "..\\x", strings.Repeat("x", 256)} {
		in := designInput()
		in.Filename = name
		_, err := f.upload(t, "", in, []byte("<!DOCTYPE html>"))
		if err == nil {
			t.Fatalf("accepted filename %q", name)
		}
		designRequireStatus(t, err, 400)
	}
	if designFilename("name\r\nInjected: yes") {
		t.Fatal("accepted control characters in display filename")
	}
	id, err := designID()
	if err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(filepath.Dir(f.dir), "private-outside")
	if err = os.WriteFile(outside, []byte("do not expose"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(f.dir, "blobs", id)
	if err = os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	if file, e := designOpen(f.dir, "blobs", id, os.O_RDONLY); e == nil {
		file.Close()
		t.Fatal("followed regular symlink")
	}
	if err = os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink(outside+"-missing", link); err != nil {
		t.Fatal(err)
	}
	if file, e := designOpen(f.dir, "blobs", id, os.O_CREATE|os.O_EXCL|os.O_WRONLY); e == nil {
		file.Close()
		t.Fatal("followed dangling symlink")
	}
	if _, err = os.Stat(outside + "-missing"); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("created outside root")
	}
	if file, e := designOpen(f.dir, "blobs", "../private-outside", os.O_RDONLY); e == nil {
		file.Close()
		t.Fatal("accepted traversal storage name")
	}
}

type designReadCounter struct {
	io.Reader
	read int64
}

func (c *designReadCounter) Read(p []byte) (int, error) {
	n, e := c.Reader.Read(p)
	c.read += int64(n)
	return n, e
}

func TestDesignMultipartEarlyCSRFAndDeadline(t *testing.T) {
	f := designFixtureNew(t)
	var prefix bytes.Buffer
	writer := multipart.NewWriter(&prefix)
	if err := writer.WriteField("csrf", "wrong"); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.CreateFormFile("file", "source.html"); err != nil {
		t.Fatal(err)
	}
	counter := &designReadCounter{Reader: io.MultiReader(bytes.NewReader(prefix.Bytes()), io.LimitReader(designZeroReader{}, maxDesignFile))}
	r := httptest.NewRequest("POST", "/web/designs", counter)
	r.Header.Set("Content-Type", writer.FormDataContentType())
	_, err := f.s.uploadDesign(&designDeadlineWriter{httptest.NewRecorder()}, r, f.b)
	designRequireStatus(t, err, 403)
	if counter.read > 64<<10 {
		t.Fatalf("read file bytes before CSRF: %d", counter.read)
	}
	if used, reserved, e := f.s.designs.totals(context.Background()); e != nil || used != 0 || reserved != 0 {
		t.Fatalf("CSRF changed quota %d/%d %v", used, reserved, e)
	}
	r = httptest.NewRequest("POST", "/web/designs", nil)
	_, err = f.s.uploadDesign(httptest.NewRecorder(), r, f.b)
	designRequireStatus(t, err, 503)
}

func TestDesignAPICurrentAuthSessionAndDownloadOnly(t *testing.T) {
	f := designFixtureNew(t)
	ctx := context.Background()
	a, err := f.upload(t, "", designInput(), []byte("<!DOCTYPE html><script>no inline</script>"))
	if err != nil {
		t.Fatal(err)
	}
	request := func(path, token, session, origin string) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest("GET", path, nil)
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		if session != "" {
			r.Header.Set("X-Session", session)
		}
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		w := &designDeadlineWriter{httptest.NewRecorder()}
		f.s.ServeHTTP(w, r)
		return w.ResponseRecorder
	}
	for _, path := range []string{"/api/v1/designs", "/api/v1/designs/" + a.AssetID, "/api/v1/designs/" + a.AssetID + "/download"} {
		if w := request(path, "", "", ""); w.Code != 401 {
			t.Fatalf("anonymous %s: %d", path, w.Code)
		}
		if w := request(path, f.token, "", ""); w.Code != 401 {
			t.Fatalf("no session %s: %d", path, w.Code)
		}
		if w := request(path, f.token, f.b.Session.SessionID, "https://other.example"); w.Code != 403 {
			t.Fatalf("origin %s: %d", path, w.Code)
		}
		if w := request(path, f.token, "other-session", ""); w.Code != 401 {
			t.Fatalf("wrong session %s: %d %s", path, w.Code, w.Body.String())
		}
	}
	w := request("/api/v1/designs/"+a.AssetID+"/download", f.token, f.b.Session.SessionID, "")
	if w.Code != 200 || w.Header().Get("Content-Type") != "application/octet-stream" || !strings.HasPrefix(w.Header().Get("Content-Disposition"), "attachment") {
		t.Fatalf("unsafe download: %d %v", w.Code, w.Header())
	}
	if err = f.board.RevokeToken(ctx, f.b.TokenID); err != nil {
		t.Fatal(err)
	}
	if w = request("/api/v1/designs/"+a.AssetID+"/download", f.token, f.b.Session.SessionID, ""); w.Code != 401 {
		t.Fatalf("revoked download: %d", w.Code)
	}
}

func TestDesignCleanupExcludesActivePending(t *testing.T) {
	f := designFixtureNew(t)
	d := f.s.designs
	pending, _, err := d.reserve(context.Background(), designReservation(1))
	if err != nil {
		t.Fatal(err)
	}
	file, err := designOpen(d.dir, "staging", pending.Blob, os.O_CREATE|os.O_EXCL|os.O_WRONLY)
	if err != nil {
		t.Fatal(err)
	}
	file.Close()
	old := time.Now().Add(-48 * time.Hour)
	if err = os.Chtimes(filepath.Join(d.dir, "staging", pending.Blob), old, old); err != nil {
		t.Fatal(err)
	}
	d.lastCleanup = time.Time{}
	if err = d.cleanup(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(filepath.Join(d.dir, "staging", pending.Blob)); err != nil {
		t.Fatalf("removed active pending: %v", err)
	}
}
