package web

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"
)

const maxDesignFile int64 = 50 << 20
const designQuota int64 = 1 << 30
const diskFloor uint64 = 512 << 20
const maxPreview int64 = 8 << 20

// designStore uses the web database only; board schema and transactions stay separate.
type designStore struct {
	db           *sql.DB
	dir          string
	mu           sync.Mutex
	uploads      chan struct{}
	previews     chan struct{}
	freeDisk     func(string) (uint64, error)
	lastCleanup  time.Time
	cleanupError string
}

type storedDesign struct {
	DesignAsset
	Sequence                                       int64
	Status, RequestID, MetadataHash, Blob, Preview string
	Reserved, PreviewSize                          int64
}

const designColumns = `seq,asset_id,design_id,version,scope,scope_key,title,filename,media_type,size,sha256,uploader_principal,uploader_owner,created_at,status,request_id,metadata_hash,blob,preview,preview_size,reserved`
const designSchema = `CREATE TABLE IF NOT EXISTS design_assets(
 seq INTEGER PRIMARY KEY AUTOINCREMENT, asset_id TEXT NOT NULL UNIQUE,
 design_id TEXT NOT NULL, version INTEGER NOT NULL DEFAULT 0,
 scope TEXT NOT NULL, scope_key TEXT NOT NULL, title TEXT NOT NULL, filename TEXT NOT NULL,
 media_type TEXT NOT NULL DEFAULT '', size INTEGER NOT NULL DEFAULT 0,
 sha256 TEXT NOT NULL DEFAULT '', uploader_principal TEXT NOT NULL, uploader_owner TEXT NOT NULL,
 created_at TEXT NOT NULL, status TEXT NOT NULL CHECK(status IN ('pending','ready')),
 request_id TEXT NOT NULL, metadata_hash TEXT NOT NULL, blob TEXT NOT NULL UNIQUE,
 preview TEXT NOT NULL DEFAULT '', preview_size INTEGER NOT NULL DEFAULT 0,
 reserved INTEGER NOT NULL DEFAULT 0,
 UNIQUE(uploader_principal,request_id));
 CREATE UNIQUE INDEX IF NOT EXISTS design_versions ON design_assets(design_id,version) WHERE status='ready';
 CREATE INDEX IF NOT EXISTS design_scope ON design_assets(status,scope,scope_key,seq);`

func (s *Server) initDesigns() error {
	d := &designStore{db: s.db, dir: s.options.DataDir, uploads: make(chan struct{}, 2), previews: make(chan struct{}, 1), freeDisk: designFreeDisk}
	for _, name := range []string{"blobs", "staging"} {
		if err := designPrivateDir(filepath.Join(d.dir, name)); err != nil {
			return err
		}
	}
	if _, err := d.db.Exec(designSchema); err != nil {
		return fmt.Errorf("design schema: %w", err)
	}
	if err := d.recover(context.Background()); err != nil {
		return err
	}
	s.designs = d
	return nil
}

func designID() (string, error) {
	var b [24]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func scanDesign(row interface{ Scan(...any) error }) (storedDesign, error) {
	var v storedDesign
	var created string
	err := row.Scan(&v.Sequence, &v.AssetID, &v.DesignID, &v.Version, &v.Scope, &v.ScopeKey, &v.Title, &v.Filename, &v.MediaType, &v.Size, &v.SHA256, &v.UploaderPrincipal, &v.UploaderOwner, &created, &v.Status, &v.RequestID, &v.MetadataHash, &v.Blob, &v.Preview, &v.PreviewSize, &v.Reserved)
	if err != nil {
		return v, err
	}
	v.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	return v, err
}

func (d *designStore) get(ctx context.Context, id string) (storedDesign, error) {
	if !designValidID(id) {
		return storedDesign{}, designFail(404, "NOT_FOUND", "Материал не найден")
	}
	v, err := scanDesign(d.db.QueryRowContext(ctx, `SELECT `+designColumns+` FROM design_assets WHERE asset_id=? AND status='ready'`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return v, designFail(404, "NOT_FOUND", "Материал не найден")
	}
	return v, err
}

func (d *designStore) totals(ctx context.Context) (used, reserved int64, err error) {
	err = d.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(CASE WHEN status='ready' THEN size+preview_size ELSE 0 END),0),COALESCE(SUM(reserved),0) FROM design_assets`).Scan(&used, &reserved)
	return
}

func (d *designStore) disk(ctx context.Context, extra int64) error {
	_, reserved, err := d.totals(ctx)
	if err != nil {
		return err
	}
	free, err := d.freeDisk(d.dir)
	if err != nil {
		return designFail(507, "STORAGE", "Не удалось проверить свободное место")
	}
	if free < diskFloor+uint64(reserved+extra) {
		return designFail(507, "STORAGE", "Недостаточно свободного места")
	}
	return nil
}

func (d *designStore) reserve(ctx context.Context, v storedDesign) (storedDesign, bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	old, err := scanDesign(d.db.QueryRowContext(ctx, `SELECT `+designColumns+` FROM design_assets WHERE uploader_principal=? AND request_id=?`, v.UploaderPrincipal, v.RequestID))
	if err == nil {
		if old.MetadataHash != v.MetadataHash {
			return old, false, designFail(409, "CONFLICT", "Повтор запроса содержит другие поля")
		}
		if old.Status != "ready" {
			return old, false, designFail(409, "CONFLICT", "Эта загрузка уже выполняется")
		}
		return old, true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return v, false, err
	}
	used, reserved, err := d.totals(ctx)
	if err != nil {
		return v, false, err
	}
	if used+reserved+maxDesignFile > designQuota {
		return v, false, designFail(413, "QUOTA", "Исчерпана квота файлов проекта")
	}
	if err = d.disk(ctx, maxDesignFile); err != nil {
		return v, false, err
	}
	if v.DesignID == "" {
		v.DesignID, err = designID()
		if err != nil {
			return v, false, err
		}
	} else {
		old, e := scanDesign(d.db.QueryRowContext(ctx, `SELECT `+designColumns+` FROM design_assets WHERE design_id=? AND status='ready' ORDER BY version DESC LIMIT 1`, v.DesignID))
		if errors.Is(e, sql.ErrNoRows) {
			return v, false, designFail(404, "NOT_FOUND", "Материал для новой версии не найден")
		}
		if e != nil {
			return v, false, e
		}
		if old.Scope != v.Scope || old.ScopeKey != v.ScopeKey {
			return v, false, designFail(409, "CONFLICT", "Связи новой версии должны совпадать с материалом")
		}
	}
	v.AssetID, err = designID()
	if err != nil {
		return v, false, err
	}
	v.Blob = v.AssetID
	v.Status = "pending"
	v.CreatedAt = time.Now().UTC()
	v.Reserved = maxDesignFile
	_, err = d.db.ExecContext(ctx, `INSERT INTO design_assets(asset_id,design_id,scope,scope_key,title,filename,uploader_principal,uploader_owner,created_at,status,request_id,metadata_hash,blob,reserved) VALUES(?,?,?,?,?,?,?,?,?,'pending',?,?,?,?)`, v.AssetID, v.DesignID, v.Scope, v.ScopeKey, v.Title, v.Filename, v.UploaderPrincipal, v.UploaderOwner, v.CreatedAt.Format(time.RFC3339Nano), v.RequestID, v.MetadataHash, v.Blob, v.Reserved)
	return v, false, err
}

func (d *designStore) abort(v storedDesign) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	var status string
	if err := d.db.QueryRow(`SELECT status FROM design_assets WHERE asset_id=?`, v.AssetID).Scan(&status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}
	if status != "pending" {
		return nil
	}
	// Remove both possible publish locations before releasing the reservation.
	for _, p := range []struct{ dir, name string }{{"staging", v.Blob}, {"blobs", v.Blob}, {"staging", v.Blob + ".preview"}, {"blobs", v.Blob + ".preview"}} {
		if err := designRemove(d.dir, p.dir, p.name); err != nil {
			return err
		}
	}
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`DELETE FROM design_assets WHERE asset_id=? AND status='pending'`, v.AssetID); err != nil {
		return err
	}
	if _, err = tx.Exec(`INSERT INTO web_audit(event,principal,at) VALUES(?,?,?)`, "design_upload_aborted", v.UploaderPrincipal, time.Now().UnixNano()); err != nil {
		return err
	}
	return tx.Commit()
}

func (d *designStore) finish(ctx context.Context, v storedDesign) (storedDesign, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return v, err
	}
	defer tx.Rollback()
	if err = tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(version),0)+1 FROM design_assets WHERE design_id=? AND status='ready'`, v.DesignID).Scan(&v.Version); err != nil {
		return v, err
	}
	_, err = tx.ExecContext(ctx, `UPDATE design_assets SET seq=(SELECT COALESCE(MAX(seq),0)+1 FROM design_assets),status='ready',version=?,media_type=?,size=?,sha256=?,preview=?,preview_size=?,reserved=0 WHERE asset_id=? AND status='pending'`, v.Version, v.MediaType, v.Size, v.SHA256, v.Preview, v.PreviewSize, v.AssetID)
	if err != nil {
		return v, err
	}
	// Move the numeric boundary at publication: an older pending upload must
	// never enter an already-started list pass when it becomes ready later.
	if _, err = tx.ExecContext(ctx, `INSERT INTO web_audit(event,principal,at) VALUES(?,?,?)`, "design_uploaded", v.UploaderPrincipal, time.Now().UnixNano()); err != nil {
		return v, err
	}
	if err = tx.Commit(); err != nil {
		return v, err
	}
	v.Status = "ready"
	return v, nil
}
