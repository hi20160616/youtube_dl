package main

import (
	"io"
	"log"
	"net/http"
)

func main() {
	// Hello world, the web server

	helloHandler := func(w http.ResponseWriter, req *http.Request) {
		io.WriteString(w, "Hello, world!\n")
	}

	http.HandleFunc("/", helloHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func ytdlHandler(w http.ResponseWriter, r *http.Request) {
	v := r.URL.Query().Get("v")
	c := &Client{}
	if err := c.Download(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
