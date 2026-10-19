package web

import (
	"agent-board/internal/core"
	"context"
	"io"
	"mime"
	"mime/multipart"
	"strings"
	"unicode"
	"unicode/utf8"
)

type designUpload struct{ Title, Scope, ScopeKey, EpicKey, DesignID, Filename string }

func designField(reader *multipart.Reader, name string) (string, error) {
	part, err := reader.NextPart()
	if err != nil {
		return "", designFail(400, "VALIDATION", "Не заполнены поля загрузки")
	}
	if part.FormName() != name || part.FileName() != "" {
		return "", designFail(403, "CSRF", "Сначала передайте защитные поля формы")
	}
	b, err := io.ReadAll(io.LimitReader(part, 4097))
	if err != nil || len(b) > 4096 {
		return "", designFail(400, "VALIDATION", "Слишком большое поле формы")
	}
	return string(b), nil
}

func designParseUpload(reader *multipart.Reader) (designUpload, *multipart.Part, error) {
	var in designUpload
	seen := make(map[string]bool)
	for range 6 {
		p, err := reader.NextPart()
		if err != nil {
			return in, nil, designFail(400, "VALIDATION", "Выберите файл")
		}
		if len(p.Header) > 2 || len(p.Header.Values("Content-Disposition")) != 1 {
			return in, nil, designFail(400, "VALIDATION", "Недопустимые заголовки файла")
		}
		name := p.FormName()
		if seen[name] {
			return in, nil, designFail(400, "VALIDATION", "Поле формы повторяется")
		}
		seen[name] = true
		if name == "file" {
			_, params, e := mime.ParseMediaType(p.Header.Get("Content-Disposition"))
			if e != nil {
				return in, nil, designFail(400, "VALIDATION", "Неверное имя файла")
			}
			in.Filename = params["filename"]
			if !designFilename(in.Filename) {
				return in, nil, designFail(400, "VALIDATION", "Недопустимое имя файла")
			}
			if strings.TrimSpace(in.Title) == "" || utf8.RuneCountInString(in.Title) > 200 || !utf8.ValidString(in.Title) {
				return in, nil, designFail(400, "VALIDATION", "Название должно содержать от 1 до 200 символов")
			}
			if in.DesignID != "" && !designValidID(in.DesignID) {
				return in, nil, designFail(400, "VALIDATION", "Неверный материал")
			}
			return in, p, nil
		}
		b, e := io.ReadAll(io.LimitReader(p, 4097))
		if e != nil || len(b) > 4096 {
			return in, nil, designFail(400, "VALIDATION", "Слишком большое поле формы")
		}
		switch name {
		case "title":
			in.Title = string(b)
		case "scope":
			in.Scope = string(b)
		case "scope_key":
			in.ScopeKey = string(b)
		case "design_id":
			in.DesignID = string(b)
		case "epic_key":
			in.EpicKey = string(b)
		default:
			return in, nil, designFail(400, "VALIDATION", "Неизвестное поле формы")
		}
	}
	return in, nil, designFail(400, "VALIDATION", "Слишком много полей формы")
}

func designFilename(name string) bool {
	if name == "" || name == "." || name == ".." || len(name) > 255 || !utf8.ValidString(name) || strings.ContainsAny(name, "/\\") {
		return false
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func (s *Server) designEpic(ctx context.Context, a core.Actor, in designUpload) error {
	if in.EpicKey == "" {
		return nil
	}
	if in.Scope != "task" {
		return designFail(400, "VALIDATION", "Эпик задачи можно указать только для связи с задачей")
	}
	task, err := s.board.Get(ctx, a, core.GetInput{TaskKey: in.ScopeKey})
	if err != nil {
		return err
	}
	epic, err := s.board.Get(ctx, a, core.GetInput{TaskKey: in.EpicKey})
	if err != nil {
		return err
	}
	if epic.Type != "epic" || task.EpicKey != epic.Key {
		return designFail(400, "VALIDATION", "Задача не принадлежит выбранному эпику")
	}
	return nil
}
