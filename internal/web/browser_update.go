package web

import (
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"agent-board/internal/core"
)

// Browser capabilities only predict button availability. Store.Update remains
// authoritative for requester, current lease ownership, fencing, and replay.
func browserUpdateInput(r *http.Request, id, key string) (core.UpdateInput, error) {
	title, description := r.PostForm.Get("title"), r.PostForm.Get("description")
	if strings.TrimSpace(title) == "" || utf8.RuneCountInString(title) > 200 {
		return core.UpdateInput{}, core.Fail("validation", "Введите заголовок до 200 символов")
	}
	if len(description) > 16384 {
		return core.UpdateInput{}, core.Fail("validation", "Введите описание до 16 КиБ")
	}
	if err := formTextError(r.PostForm, []string{"title"}, []string{"description"}); err != nil {
		return core.UpdateInput{}, err
	}
	var generation int64
	if value := r.PostForm.Get("generation"); value != "" {
		var err error
		generation, err = strconv.ParseInt(value, 10, 64)
		if err != nil || generation < 0 {
			return core.UpdateInput{}, core.Fail("validation", "Неверное поколение владения")
		}
	}
	return core.UpdateInput{LeaseInput: core.LeaseInput{RequestID: id, TaskKey: key, Generation: generation}, Title: &title, Description: &description}, nil
}
