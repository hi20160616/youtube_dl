package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/hi20160616/gears"
	"github.com/kkdai/youtube/v2"
	ytdl "github.com/kkdai/youtube/v2/downloader"
	"golang.org/x/sync/errgroup"
)

const address = ":1234"

var (
	jobs    = make(map[string]string)
	sema    = make(chan struct{}, 1)
	retry   = 10
	dlPath  = "Downloads"
	ytdlExe = "ytdl"
)

func init() {
	root, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}
	dlPath = filepath.Join(root, dlPath)
	if err = downloadYtdlExe(); err != nil {
		log.Printf("%v", err)
	}
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
			if err := download2(v, q); err != nil {
				log.Println(err)
			}
			<-sema
		}
	}
}

func download2(id string, quality string) error {
	defer func() {
		// cleanup()
		delete(jobs, id)
	}()

	if err := checkFFMPEG(); err != nil {
		return err
	}

	id, err := youtube.ExtractVideoID(id)
	if err != nil {
		return err
	}

	opt := "bestvideo[height <=? 720][ext=mp4]+bestaudio[ext=m4a]/best[height <=? 720]/best"
	if strings.Contains(quality, "1080") {
		opt = "bestvideo[height <=? 1080][ext=mp4]+bestaudio[ext=m4a]/best[height <=? 1080]/best"
	}

	exe := func() string {
		if runtime.GOOS == "windows" {
			return ytdlExe + ".exe"
		}
		return "./" + ytdlExe
	}()

	// https://github.com/ytdl-org/youtube-dl
	cmd := &exec.Cmd{}
	cmd = exec.Command(exe, "-o", filepath.Join(dlPath, "%(title)s.%(ext)s"),
		"-f", opt, id)
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		log.Printf("%v", err)
	}
	log.Printf("video: %s download done.", id)
	return nil
}

// download will download youtube video by src and quality,
// src is the video id,
// quality can be hd720 or hd1080 etc., default is medium
func download(id string, quality string) error {
	defer func() {
		// cleanup()
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
	if err = dl.Download(context.Background(), v, vfmt, ""); err != nil {
		return err
	}
	log.Printf("video: %s download done.", id)
	return nil
}

func checkFFMPEG() error {
	if err := exec.Command("ffmpeg", "-version").Run(); err != nil {
		return fmt.Errorf("please check ffmpegCheck is installed correctly")
	}
	return nil
}

func downloadFile(url, savePath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(savePath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func downloadYtdlExe() error {
	savePath, url := "", ""
	if runtime.GOOS == "windows" {
		savePath = ytdlExe + ".exe"
		url = "https://yt-dl.org/latest/youtube-dl.exe"
	} else {
		savePath = ytdlExe
		url = "https://yt-dl.org/downloads/latest/youtube-dl"
	}
	if !gears.Exists(savePath) {
		return downloadFile(url, savePath)
	}
	return nil
}

// cannot work right on windows cause files on use
// func cleanup() error {
//         files, err := ioutil.ReadDir(dlPath)
//         if err != nil {
//                 return err
//         }
//         for _, f := range files {
//                 fExt := filepath.Ext(f.Name())
//                 if fExt == ".m4a" || fExt == ".m4v" {
//                         if err = os.Remove(f.Name()); err != nil {
//                                 return err
//                         }
//                 }
//         }
//         return nil
// }
