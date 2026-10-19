package web

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
	"testing"
)

func TestDesignCursorExcludesEarlierPendingPublication(t *testing.T) {
	f := designFixtureNew(t)
	ctx := context.Background()
	d := f.s.designs
	pending, _, err := d.reserve(ctx, designReservation(500))
	if err != nil {
		t.Fatal(err)
	}
	var firstID string
	for range 2 {
		asset, e := f.upload(t, "", designInput(), []byte("<!DOCTYPE html><title>ready</title>"))
		if e != nil {
			t.Fatal(e)
		}
		if firstID == "" {
			firstID = asset.AssetID
		}
	}
	page, err := f.s.ListDesigns(ctx, f.b.actor(), DesignFilter{Limit: 1})
	if err != nil || page.NextCursor == "" {
		t.Fatalf("first page: %+v %v", page, err)
	}
	data := []byte("<!DOCTYPE html><title>late pending</title>")
	sum := sha256.Sum256(data)
	file, err := designOpen(d.dir, "staging", pending.Blob, os.O_CREATE|os.O_EXCL|os.O_WRONLY)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = file.Write(data); err != nil {
		t.Fatal(err)
	}
	if err = file.Sync(); err != nil {
		t.Fatal(err)
	}
	file.Close()
	if err = designPublish(d.dir, pending.Blob); err != nil {
		t.Fatal(err)
	}
	pending.Size = int64(len(data))
	pending.SHA256 = hex.EncodeToString(sum[:])
	pending.MediaType = "text/html"
	if _, err = d.finish(ctx, pending); err != nil {
		t.Fatal(err)
	}
	next, err := f.s.ListDesigns(ctx, f.b.actor(), DesignFilter{Limit: 1, Cursor: page.NextCursor})
	if err != nil || len(next.Assets) != 1 || next.Assets[0].AssetID != firstID || next.NextCursor != "" {
		t.Fatalf("late publication changed old cursor pass: %+v %v", next, err)
	}
}

type designInterruptedReader struct{ sent bool }

func (r *designInterruptedReader) Read(p []byte) (int, error) {
	if r.sent {
		return 0, context.Canceled
	}
	r.sent = true
	return copy(p, []byte("<!DOCTYPE html><title>partial</title>")), nil
}
func TestDesignInterruptedUploadReleasesPendingQuota(t *testing.T) {
	f := designFixtureNew(t)
	_, err := f.uploadStream(t, "", designInput(), &designInterruptedReader{})
	if err == nil || !strings.Contains(err.Error(), "canceled") {
		t.Fatalf("interruption lost: %v", err)
	}
	status, err := f.s.DesignStorageStatus(context.Background())
	if err != nil || status.Used != 0 || status.Reserved != 0 {
		t.Fatalf("interruption leaked quota: %+v %v", status, err)
	}
	for _, dir := range []string{"blobs", "staging"} {
		entries, e := os.ReadDir(f.dir + "/" + dir)
		if e != nil || len(entries) != 0 {
			t.Fatalf("interruption left %s: %v %v", dir, entries, e)
		}
	}
}
