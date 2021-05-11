package main

import (
	"io"
	"os"

	"github.com/kkdai/youtube/v2"
)

type Client struct {
	youtube.Client
}

func (c *Client) Download(s string) error {
	v, err := c.GetVideo(s)
	if err != nil {
		return err
	}

	resp, err := c.GetStream(v, &v.Formats[0])
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// title := v.Title

	file, err := os.Create("video.mp4")
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return err
	}
	return nil
}
