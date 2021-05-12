package main

import (
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/hi20160616/gears"
	"github.com/kkdai/youtube/v2"
	ytdl "github.com/kkdai/youtube/v2/downloader"
	"github.com/vbauerster/mpb/v5"
	"github.com/vbauerster/mpb/v5/decor"
)

type Client struct {
	youtube.Client
	dr *ytdl.Downloader
}

func (c *Client) logf(format string, v ...interface{}) {
	if c.Debug {
		log.Printf(format, v...)
	}
}

func (c *Client) Download(s string) error {
	// c.Debug = true
	v, err := c.GetVideo(s)
	if err != nil {
		return err
	}
	vfmt := v.Formats.FindByQuality("hd720")
	c.logf("Video '%s' - Quality '%s' - Codec '%s'", v.Title, vfmt.QualityLabel, vfmt.MimeType)

	root, err := os.Getwd()
	if err != nil {
		return err
	}
	dlPath := filepath.Join(root, "Downloads")
	outName := filepath.Join(dlPath, Sanitize(v.Title)+pickIdealFileExtension(vfmt.MimeType))
	gears.MakeDirAll(dlPath)
	outFile, err := os.Create(outName)
	if err != nil {
		return err
	}
	defer outFile.Close()

	c.logf("Download to file=%s", outName)
	return c.walk(outFile, v, vfmt)
}

func (c *Client) walk(out *os.File, video *youtube.Video, format *youtube.Format) error {
	resp, err := c.GetStream(video, format)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	prog := &progress{
		contentLength: float64(resp.ContentLength),
	}

	// create progress bar
	progress := mpb.New(mpb.WithWidth(64))
	bar := progress.AddBar(
		int64(prog.contentLength),

		mpb.PrependDecorators(
			decor.CountersKibiByte("% .2f / % .2f"),
			decor.Percentage(decor.WCSyncSpace),
		),
		mpb.AppendDecorators(
			decor.EwmaETA(decor.ET_STYLE_GO, 90),
			decor.Name(" ] "),
			decor.EwmaSpeed(decor.UnitKiB, "% .2f", 60),
		),
	)

	reader := bar.ProxyReader(resp.Body)
	mw := io.MultiWriter(out, prog)
	_, err = io.Copy(mw, reader)
	if err != nil {
		return err
	}

	progress.Wait()
	return nil
}
