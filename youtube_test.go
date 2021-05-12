package main

import (
	"testing"
)

func TestDownload(t *testing.T) {
	c := &Client{}
	if err := c.Download("https://www.youtube.com/watch?v=sxnjarLK5l4"); err != nil {
		t.Error(err)
	}
	// c.Download("https://www.youtube.com/watch?v=sxnjarLK5l4")
}
