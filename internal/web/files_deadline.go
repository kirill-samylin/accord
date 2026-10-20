package web

import (
	"net/http"
	"time"
)

func designDeadline(w http.ResponseWriter) error {
	controller := http.NewResponseController(w)
	deadline := time.Now().Add(120 * time.Second)
	if err := controller.SetReadDeadline(deadline); err != nil {
		return designFail(503, "DEADLINE", "Загрузка файлов временно недоступна")
	}
	if err := controller.SetWriteDeadline(deadline); err != nil {
		return designFail(503, "DEADLINE", "Передача файлов временно недоступна")
	}
	return nil
}
