package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDownload(t *testing.T) {
	// if err := download("https://www.youtube.com/watch?v=sxnjarLK5l4", ""); err != nil {
	if err := download("https://www.youtube.com/watch?v=eK4xFueaUsI", ""); err != nil {
		t.Error(err)
	}
}

func TestYtdlHandler(t *testing.T) {
	// r, err := http.NewRequest("get", "/?v=https://www.youtube.com/watch?v=sxnjarLK5l4", nil)
	r, err := http.NewRequest("get", "/?v=https://www.youtube.com/watch?v=sxnjarLK5l4&q=hd720", nil)
	if err != nil {
		t.Error(err)
	}

	w := httptest.NewRecorder()
	h := http.HandlerFunc(ytdlHandler)
	h.ServeHTTP(w, r)
	if status := w.Code; status != http.StatusOK {
		t.Error(status)
	}
}
