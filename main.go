package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
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

var (
	jobs   = make(map[string]string)
	sema   = make(chan struct{}, 1)
	retry  = 5
	dlPath = "Downloads"
)

func init() {
	root, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}
	dlPath = filepath.Join(root, dlPath)
}

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
		fmt.Fprintf(w, "Download videos list: \n")
		for jv, jq := range jobs {
			fmt.Fprintf(w, "video id: %s, video quality: %s\n", jv, jq)
		}
	}
}

func treatJobs() error {
	for {
		for v, q := range jobs {
			sema <- struct{}{}
			if err := download(v, q); err != nil {
				log.Println(err)
			}
			log.Printf("video: %s download done.", v)
			<-sema
		}
	}
}

// download will download youtube video by src and qulity,
// src is the video id,
// quality can be hd720 or hd1080 etc., default is medium
func download(id string, quality string) error {
	defer func() {
		cleanup()
		delete(jobs, id)
	}()
	dl := ytdl.Downloader{}
	// dl.Debug = true
	v, err := dl.Client.GetVideo(id)
	if err != nil {
		log.Printf("err video: %s, %v", id, err)
		retry -= 1
		if retry == 0 {
			return errors.New("retry many times, pass this video: " + id + ".")
		}
		return download(id, quality)
	}

	if quality == "" {
		quality = "hd720"
	}
	vfmt := v.Formats.FindByQuality(quality)
	if vfmt == nil {
		if len(v.Formats) > 0 {
			vfmt = &v.Formats[0]
		} else {
			return errors.New("cannot fetch video format on id" + id)
		}
	}

	dl.OutputDir = dlPath

	if strings.HasPrefix(vfmt.Quality, "hd") {
		if err = checkFFMPEG(); err != nil {
			return err
		}
		return dl.DownloadComposite(context.Background(), "", v, vfmt.Quality, "mp4")
	}
	return dl.Download(context.Background(), v, vfmt, "")
}

func checkFFMPEG() error {
	if err := exec.Command("ffmpeg", "-version").Run(); err != nil {
		return fmt.Errorf("please check ffmpegCheck is installed correctly")
	}
	return nil
}

func cleanup() error {
	files, err := ioutil.ReadDir(dlPath)
	if err != nil {
		return err
	}
	for _, f := range files {
		fExt := filepath.Ext(f.Name())
		if fExt == ".m4a" || fExt == ".m4v" {
			if err = os.Remove(f.Name()); err != nil {
				return err
			}
		}
	}
	return nil
}
