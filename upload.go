package main

import (
	"./baidu"
	"./hashbin"
	"./snapshot"
	"./utils"
	"bytes"
	"code.google.com/p/goauth2/oauth"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type Job struct {
	backend *hashbin.Bin
	path    string
	chunk   *snapshot.Chunk
}

func (self *App) runUpload() {

	// backends //TODO configurable
	backends := make([]*hashbin.Bin, 0)
	b, err := self.getBaiduBackend() // baidu
	if err != nil {
		log.Fatal(err)
	}
	backends = append(backends, b)

	// snapshot to upload //TODO specifiable
	if len(self.snapshotSet.Snapshots) == 0 {
		fmt.Printf("no snapshot\n")
		os.Exit(0)
	}
	lastSnapshot := self.snapshotSet.Snapshots[len(self.snapshotSet.Snapshots)-1]

	// generate jobs
	paths := make([]string, 0, len(lastSnapshot.Files))
	for path, _ := range lastSnapshot.Files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	jobs := make([]Job, 0)
	var totalSize int64
	for _, path := range paths {
		file := lastSnapshot.Files[path]
		for _, chunk := range file.Chunks {
			for _, backend := range backends {
				exists, err := backend.Exists(int(chunk.Length), chunk.Hash)
				if err != nil {
					log.Fatal(err)
				}
				if exists {
					continue
				}
				totalSize += chunk.Length
				jobs = append(jobs, Job{
					backend: backend,
					chunk:   chunk,
					path:    path,
				})
			}
		}
	}

	// upload
	semSize := 4
	sem := make(chan []byte, semSize)
	for i := 0; i < semSize; i++ {
		sem <- make([]byte, snapshot.MAX_CHUNK_SIZE)
	}

	var uploaded int64
	go func() {
		for _ = range time.NewTicker(time.Second * 10).C {
			fmt.Printf("=> %s / %s / %s\n",
				utils.FormatSize(int(uploaded)),
				utils.FormatSize(int(totalSize)),
				utils.FormatSize(int(totalSize-uploaded)))
		}
	}()

	for i, job := range jobs {
		buf := <-sem
		go func(i int, job Job) {
			defer func() {
				sem <- buf
				uploaded += job.chunk.Length
			}()
			f, err := os.OpenFile(job.path, os.O_RDONLY, 0644)
			if err != nil {
				log.Fatal(err)
			}
			defer f.Close()
			o, err := f.Seek(job.chunk.Offset, 0)
			if err != nil || o != job.chunk.Offset {
				log.Fatal(err)
			}
			buf = buf[:job.chunk.Length]
			n, err := io.ReadFull(f, buf)
			if int64(n) != job.chunk.Length || err != nil {
				log.Fatal(err)
			}
			fmt.Printf("=> job %d / %d: %s %d %d\n\t%d-%s\n",
				i+1, len(jobs),
				job.path, job.chunk.Offset, job.chunk.Length,
				job.chunk.Length, job.chunk.Hash)
			job.backend.Save(int(job.chunk.Length), job.chunk.Hash, bytes.NewReader(buf))
		}(i, job)
	}

}

func (self *App) getBaiduBackend() (*hashbin.Bin, error) {
	var dir string
	var token oauth.Token
	err := self.register.Get("baidu_dir", &dir)
	if err != nil {
		return nil, err
	}
	err = self.register.Get("baidu_token", &token)
	if err != nil {
		return nil, err
	}
	keyCacheFilePath := filepath.Join(self.dataDir, "baidu.keys")
	b, err := baidu.New(dir, &token, keyCacheFilePath)
	if err != nil {
		return nil, err
	}
	return hashbin.New(b), nil
}
