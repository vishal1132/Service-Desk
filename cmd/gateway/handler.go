package main

import (
	"io"
	"net/http"

	"github.com/rs/zerolog"
)

type company struct {
	nAgents   int
	slots     []int
	companyID string
}

var companyMap map[string]*company

var servermapping map[string]string

type handler struct {
	l *zerolog.Logger
}

func (h *handler) handleNotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
}

func (h *handler) handleRUOK(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "imok")
}

func agents() {

}
