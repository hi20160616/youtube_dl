package main

import (
	"context"
	"io"
	"log"
	"mime"
	"os"
	"path/filepath"
	"strings"

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

	// filename := filepath.Join("./", Sanitize(v.Title))
	vfmt := v.Formats.FindByQuality("hd720")
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	dlPath := filepath.Join(root, "Downloads")
	outName := filepath.Join(dlPath, Sanitize(v.Title)+pickIdealFileExtension(vfmt.MimeType))
	gears.MakeDirAll(dlPath)
	file, err := os.Create(outName)
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

func Sanitize(title string) string {
	// Characters not allowed on mac
	//	:/
	// Characters not allowed on linux
	//	/
	// Characters not allowed on windows
	//	<>:"/\|?*
	r := strings.NewReplacer("<", "(", ">", ")", "\\", "-", "/", "-", ":", "：", "*", "x", "\"", "“", "|", "｜")
	return r.Replace(title)
}

func (c *Client) logf(format string, v ...interface{}) {
	if c.Debug {
		log.Printf(format, v...)
	}
}

func (c *Client) Download2(ctx context.Context, s string) error {
	v, err := c.GetVideo(s)
	// c.Debug = true
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
	return c.videoDLWorker(outFile, v, vfmt)
}

func (c *Client) videoDLWorker(out *os.File, video *youtube.Video, format *youtube.Format) error {
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

type progress struct {
	contentLength     float64
	totalWrittenBytes float64
	downloadLevel     float64
}

func (dl *progress) Write(p []byte) (n int, err error) {
	n = len(p)
	dl.totalWrittenBytes = dl.totalWrittenBytes + float64(n)
	currentPercent := (dl.totalWrittenBytes / dl.contentLength) * 100
	if (dl.downloadLevel <= currentPercent) && (dl.downloadLevel < 100) {
		dl.downloadLevel++
	}
	return
}

const defaultExtension = ".mov"

// Rely on hardcoded canonical mime types, as the ones provided by Go aren't exhaustive [1].
// This seems to be a recurring problem for youtube downloaders, see [2].
// The implementation is based on mozilla's list [3], IANA [4] and Youtube's support [5].
// [1] https://github.com/golang/go/blob/ed7888aea6021e25b0ea58bcad3f26da2b139432/src/mime/type.go#L60
// [2] https://github.com/ZiTAL/youtube-dl/blob/master/mime.types
// [3] https://developer.mozilla.org/en-US/docs/Web/HTTP/Basics_of_HTTP/MIME_types/Common_types
// [4] https://www.iana.org/assignments/media-types/media-types.xhtml#video
// [5] https://support.google.com/youtube/troubleshooter/2888402?hl=en
var canonicals = map[string]string{
	"video/quicktime":  ".mov",
	"video/x-msvideo":  ".avi",
	"video/x-matroska": ".mkv",
	"video/mpeg":       ".mpeg",
	"video/webm":       ".webm",
	"video/3gpp2":      ".3g2",
	"video/x-flv":      ".flv",
	"video/3gpp":       ".3gp",
	"video/mp4":        ".mp4",
	"video/ogg":        ".ogv",
	"video/mp2t":       ".ts",
}

func pickIdealFileExtension(mediaType string) string {
	mediaType, _, err := mime.ParseMediaType(mediaType)
	if err != nil {
		return defaultExtension
	}

	if extension, ok := canonicals[mediaType]; ok {
		return extension
	}

	// Our last resort is to ask the operating system, but these give multiple results and are rarely canonical.
	extensions, err := mime.ExtensionsByType(mediaType)
	if err != nil || extensions == nil {
		return defaultExtension
	}

	return extensions[0]
}
