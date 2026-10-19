package web

import (
	"net/url"
	"strconv"
	"testing"
)

func TestWhitespaceAnswerIsCorrectableWithoutLosingLease(t *testing.T) {
	f := fixture(t)
	token := f.token(t, "human")
	task := f.question(t, token)
	cookie := f.login(t, token)
	page := f.page(t, cookie, "/web/tasks/"+task.Key)
	if w := f.post(cookie, "/web/tasks/"+task.Key+"/claim", page, "claim", nil); w.Code != 303 {
		t.Fatal(w.Code)
	}
	page = f.page(t, cookie, "/web/tasks/"+task.Key)
	fields := url.Values{"generation": {strconv.FormatInt(page.Task.Generation, 10)}, "answer": {"  "}}
	w := f.post(cookie, "/web/tasks/"+task.Key+"/answer", page, "answer", fields)
	if w.Code != 400 {
		t.Fatalf("empty answer status%d", w.Code)
	}
	corrected := f.render.last()
	if !corrected.CanEdit || corrected.FieldErrors["answer"] == "" || corrected.Draft["answer"] != "  " {
		t.Fatal("validation lost edit state or draft")
	}
	fields.Set("answer", "Valid decision")
	if w = f.post(cookie, "/web/tasks/"+task.Key+"/answer", corrected, "answer", fields); w.Code != 303 {
		t.Fatalf("corrected answer status%d", w.Code)
	}
}
