package main

import (
	"testing"
)

func TestDownload(t *testing.T) {
	c := &Client{}
	c.Download("https://www.youtube.com/watch?v=sxnjarLK5l4")
}
