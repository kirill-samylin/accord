package web

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"agent-board/internal/core"
)

type designCursor struct {
	High, Last int64
	Filter     string
}

func (s *Server) designActor(ctx context.Context, a core.Actor) error {
	p, _, err := s.board.AuthenticateID(ctx, a.Principal.ID)
	if err != nil {
		return err
	}
	if p != a.Principal {
		return core.Fail("AUTH_REQUIRED", "Недействительный участник")
	}
	_, err = s.board.ListTasks(ctx, a, core.ListInput{Limit: 1})
	return err
}

// ListDesigns returns bounded metadata after revalidating the live token and session.
// Its opaque cursor freezes a numeric creation boundary for a stable paginated pass.
func (s *Server) ListDesigns(ctx context.Context, a core.Actor, f DesignFilter) (DesignPage, error) {
	if err := s.designActor(ctx, a); err != nil {
		return DesignPage{}, err
	}
	if err := s.designScope(ctx, a, f.Scope, f.ScopeKey, false); err != nil {
		return DesignPage{}, err
	}
	if f.DesignID != "" && !designValidID(f.DesignID) {
		return DesignPage{}, designFail(400, "VALIDATION", "Неверный материал")
	}
	if f.Limit == 0 {
		f.Limit = 30
	}
	if f.Limit < 1 || f.Limit > 100 {
		return DesignPage{}, designFail(400, "VALIDATION", "Неверный размер страницы")
	}
	d := s.designs
	hash := sha256.Sum256([]byte(strings.Join([]string{f.Scope, f.ScopeKey, f.DesignID, strconv.Itoa(f.Limit)}, "\x00")))
	filter := hex.EncodeToString(hash[:])
	var c designCursor
	if f.Cursor != "" {
		if len(f.Cursor) > 512 {
			return DesignPage{}, designFail(400, "VALIDATION", "Неверная страница")
		}
		data, err := base64.RawURLEncoding.DecodeString(f.Cursor)
		if err != nil || json.Unmarshal(data, &c) != nil || c.High < 0 || c.Last < 1 || c.Last > c.High || c.Filter != filter {
			return DesignPage{}, designFail(400, "VALIDATION", "Неверная страница")
		}
	} else {
		if err := d.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(seq),0) FROM design_assets WHERE status='ready'`).Scan(&c.High); err != nil {
			return DesignPage{}, err
		}
		c.Last = c.High + 1
		c.Filter = filter
	}
	rows, err := d.db.QueryContext(ctx, `SELECT `+designColumns+` FROM design_assets WHERE status='ready' AND seq<=? AND seq<? AND (?='' OR scope=?) AND (?='' OR scope_key=?) AND (?='' OR design_id=?) ORDER BY seq DESC LIMIT ?`, c.High, c.Last, f.Scope, f.Scope, f.ScopeKey, f.ScopeKey, f.DesignID, f.DesignID, f.Limit+1)
	if err != nil {
		return DesignPage{}, err
	}
	defer rows.Close()
	page := DesignPage{Assets: []DesignAsset{}}
	var last int64
	for rows.Next() {
		v, e := scanDesign(rows)
		if e != nil {
			return DesignPage{}, e
		}
		if len(page.Assets) == f.Limit {
			c.Last = last
			data, e := json.Marshal(c)
			if e != nil {
				return DesignPage{}, e
			}
			page.NextCursor = base64.RawURLEncoding.EncodeToString(data)
			break
		}
		last = v.Sequence
		page.Assets = append(page.Assets, designMetadata(v, false))
	}
	return page, rows.Err()
}

// GetDesign never exposes storage paths or credentials. Download uses current auth.
func (s *Server) GetDesign(ctx context.Context, a core.Actor, id string) (DesignAsset, error) {
	if err := s.designActor(ctx, a); err != nil {
		return DesignAsset{}, err
	}
	v, err := s.designs.get(ctx, id)
	if err != nil {
		return DesignAsset{}, err
	}
	return designMetadata(v, false), nil
}

func designMetadata(v storedDesign, browser bool) DesignAsset {
	a := v.DesignAsset
	prefix := "/api/v1/designs/"
	if browser {
		prefix = "/web/designs/"
	}
	a.DownloadURL = prefix + a.AssetID + "/download"
	if browser && v.Preview != "" {
		a.PreviewURL = prefix + a.AssetID + "/preview"
	}
	return a
}

func (s *Server) designScope(ctx context.Context, a core.Actor, scope, key string, required bool) error {
	if scope == "" && !required && key == "" {
		return nil
	}
	switch scope {
	case "project":
		if required && key == "" {
			return designFail(400, "VALIDATION", "Не указан проект")
		}
	case "epic", "task":
		if key == "" {
			return designFail(400, "VALIDATION", "Выберите связь с материалом")
		}
		t, err := s.board.Get(ctx, a, core.GetInput{TaskKey: key})
		if err != nil {
			return err
		}
		if t.Type != scope {
			return designFail(400, "VALIDATION", "Тип связи не совпадает")
		}
	default:
		return designFail(400, "VALIDATION", "Неверный вид связи")
	}
	return nil
}

func (d *designStore) findRequest(ctx context.Context, principal, request string) (storedDesign, error) {
	v, err := scanDesign(d.db.QueryRowContext(ctx, `SELECT `+designColumns+` FROM design_assets WHERE uploader_principal=? AND request_id=?`, principal, request))
	if errors.Is(err, sql.ErrNoRows) {
		return v, fmt.Errorf("upload request unavailable")
	}
	return v, err
}
