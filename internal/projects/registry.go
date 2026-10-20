package projects

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"time"
)

func Open(options Options) (*Registry, error) {
	if options.Default == nil || !options.Offline && (options.Browser == nil || options.Renderer == nil || options.Origin == "") {
		return nil, errors.New("default board and browser required")
	}
	if err := Preflight(options.DataDir, options.DefaultDB, options.DefaultWeb); err != nil {
		return nil, err
	}
	root, err := privateRoot(options.DataDir)
	if err != nil {
		return nil, err
	}
	options.DataDir = root
	options.DefaultDB, err = filepath.Abs(options.DefaultDB)
	if err != nil {
		return nil, err
	}
	options.DefaultWeb, err = filepath.Abs(options.DefaultWeb)
	if err != nil {
		return nil, err
	}
	fresh, err := activateRegistryRoot(root)
	if err != nil {
		return nil, err
	}
	var db *sql.DB
	if fresh {
		db, err = createRegistry(root)
	} else {
		db, err = openRegistry(root)
	}
	if err != nil {
		return nil, err
	}
	r := &Registry{db: db, options: options, runtimes: make(map[string]*runtime)}
	var id, dbPath, webPath, key string
	err = db.QueryRow(`SELECT default_id,default_db,default_web,signing_key FROM registry_meta WHERE id=1`).Scan(&id, &dbPath, &webPath, &key)
	if errors.Is(err, sql.ErrNoRows) && fresh {
		id, err = random("p_", 16)
		if err != nil {
			db.Close()
			return nil, err
		}
		key, err = random("", 32)
		if err != nil {
			db.Close()
			return nil, err
		}
		tx, e := db.Begin()
		if e != nil {
			db.Close()
			return nil, e
		}
		_, err = tx.Exec(`INSERT INTO registry_meta(id,default_id,default_db,default_web,signing_key) VALUES(1,?,?,?,?)`, id, options.DefaultDB, options.DefaultWeb, key)
		if err == nil {
			_, err = tx.Exec(`INSERT INTO projects(id,name,state,version,creator,created) VALUES(?,?,'active',1,'',?)`, id, options.Config.Project, time.Now().UnixNano())
		}
		if err == nil {
			err = tx.Commit()
		} else {
			tx.Rollback()
		}
	} else if err == nil && (dbPath != options.DefaultDB || webPath != options.DefaultWeb) {
		err = errors.New("registry default board paths changed")
	}
	if err != nil {
		db.Close()
		return nil, err
	}
	if err = syncRegistryDirectory(root); err != nil {
		db.Close()
		return nil, err
	}
	r.defaultID = id
	r.signingKey, err = hex.DecodeString(key)
	if err != nil || len(r.signingKey) != 32 {
		db.Close()
		return nil, errors.New("invalid registry signing key")
	}
	if options.Offline {
		r.runtimes[id] = &runtime{store: options.Default}
	} else {
		if err = r.installDefault(); err != nil {
			db.Close()
			return nil, err
		}
	}
	return r, nil
}
func (r *Registry) DefaultID() string { return r.defaultID }
func (r *Registry) projectPaths(id string) (string, string, error) {
	if !validID(id, "p_") {
		return "", "", notFound()
	}
	if id == r.defaultID {
		return r.options.DefaultDB, r.options.DefaultWeb, nil
	}
	base := filepath.Join(r.options.DataDir, "projects", id)
	return filepath.Join(base, "board.db"), filepath.Join(base, "web"), nil
}
func (r *Registry) identity(ctx context.Context, tx *sql.Tx, id string) (Identity, error) {
	var who Identity
	err := tx.QueryRowContext(ctx, `SELECT id,source,token_id,owner,machine FROM identities WHERE id=?`, id).Scan(&who.ID, &who.Source, &who.TokenID, &who.Owner, &who.Machine)
	if errors.Is(err, sql.ErrNoRows) {
		return who, notFound()
	}
	if err != nil {
		return who, err
	}
	if who.Source == "legacy" {
		p, until, e := r.options.Default.AuthenticateID(ctx, who.TokenID)
		if e != nil {
			return who, e
		}
		if p.Owner != who.Owner || p.Machine != who.Machine {
			return who, denied()
		}
		who.Expires = until
	} else {
		var expires int64
		var revoked bool
		err = tx.QueryRowContext(ctx, `SELECT expires,revoked FROM credentials WHERE identity_id=?`, id).Scan(&expires, &revoked)
		who.Expires = time.Unix(0, expires)
		if err == nil && (revoked || !who.Expires.After(time.Now())) {
			err = denied()
		}
	}
	return who, err
}

// A durable marker is a one-way activation barrier. Interrupted initialization
// requires operator inspection; it never authorizes guessing a fresh identity DB.
func activateRegistryRoot(root string) (bool, error) {
	marker := filepath.Join(root, "MULTIPROJECT_REQUIRES_CAPABILITY")
	info, err := os.Lstat(marker)
	if err == nil {
		if !info.Mode().IsRegular() || info.Mode().Perm() != 0600 {
			return false, errors.New("unsafe projects capability marker")
		}
		return false, nil
	}
	if !os.IsNotExist(err) {
		return false, err
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return false, err
	}
	if len(entries) != 0 {
		return false, errors.New("unmarked registry state requires inspection")
	}
	file, err := os.OpenFile(marker, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return false, err
	}
	_, err = file.WriteString("registry=1 acl=1\n")
	if err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return false, err
	}
	if err = syncRegistryDirectory(root); err != nil {
		return false, err
	}
	return true, nil
}
func syncRegistryDirectory(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	err = file.Sync()
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	return err
}
