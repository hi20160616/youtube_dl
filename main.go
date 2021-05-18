package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kkdai/youtube/v2"
	ytdl "github.com/kkdai/youtube/v2/downloader"
	"golang.org/x/sync/errgroup"
)

const address = ":1234"

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	// Web goroutine
	g.Go(func() error {
		defer cancel()
		http.HandleFunc("/", ytdlHandler)
		log.Println("Youtube Downloader running on ", address)
		return http.ListenAndServe(address, nil)
	})

	// Download
	g.Go(func() error {
		defer cancel()
		return treatJobs()
	})

	if err := g.Wait(); err != nil {
		log.Printf("main: %#v", err)
	}
}

func ytdlHandler(w http.ResponseWriter, r *http.Request) {
	v := r.URL.Query().Get("v")
	q := r.URL.Query().Get("q")
	if v == "" {
		http.Error(w, "Hello world!\n", http.StatusBadRequest)
	} else {
		id, err := youtube.ExtractVideoID(v)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		jobs[id] = q
	}
}

var (
	jobs = make(map[string]string)
	sema = make(chan struct{}, 1)
)

func treatJobs() error {
	for {
		for v, q := range jobs {
			sema <- struct{}{}
			if err := download(v, q); err != nil {
				log.Println(err)
			}
			<-sema
		}
	}
}

// download will download youtube video by src and qulity,
// src is the video url or video id,
// quality can be hd720 or hd1080 etc., default is medium
func download(src string, quality string) error {
	dl := ytdl.Downloader{}
	// dl.Debug = true
	v, err := dl.Client.GetVideo(src)
	if err != nil {
		return err
	}

	if quality == "" {
		quality = "hd720"
	}
	vfmt := v.Formats.FindByQuality(quality)

	root, err := os.Getwd()
	if err != nil {
		return err
	}
	dl.OutputDir = filepath.Join(root, "Downloads")

	if strings.HasPrefix(quality, "hd") {
		if err = checkFFMPEG(); err != nil {
			return err
		}
		return dl.DownloadComposite(context.Background(), "", v, quality, "mp4")
	}
	return dl.Download(context.Background(), v, vfmt, "")
}

func checkFFMPEG() error {
	if err := exec.Command("ffmpeg", "-version").Run(); err != nil {
		return fmt.Errorf("please check ffmpegCheck is installed correctly")
	}
	return nil
}
