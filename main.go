package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	ytdl "github.com/kkdai/youtube/v2/downloader"
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
	q := r.URL.Query().Get("q")
	if err := Download(v, q); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Download will download youtube video by src and qulity,
// src is the video url or video id,
// quality can be hd720 or hd1080 etc., default is medium
func Download(src string, quality string) error {
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
	fmt.Println("check ffmpeg is installed....")
	if err := exec.Command("ffmpeg", "-version").Run(); err != nil {
		return fmt.Errorf("please check ffmpegCheck is installed correctly")
	}

	return nil
}
