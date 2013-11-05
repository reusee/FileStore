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
)

func (self *App) runUpload() {
	backends := make([]*hashbin.Bin, 0)

	// baidu
	b, err := self.getBaiduBackend()
	if err != nil {
		log.Fatal(err)
	}
	backends = append(backends, b)

	if len(self.snapshotSet.Snapshots) == 0 {
		fmt.Printf("no snapshot\n")
		os.Exit(0)
	}
	lastSnapshot := self.snapshotSet.Snapshots[len(self.snapshotSet.Snapshots)-1]

	paths := make([]string, 0, len(lastSnapshot.Files))
	var remaining, uploaded int64
	for path, file := range lastSnapshot.Files {
		paths = append(paths, path)
		remaining += file.Size
	}
	sort.Strings(paths)

	semSize := 4
	sem := make(chan []byte, semSize)
	for i := 0; i < semSize; i++ {
		sem <- make([]byte, snapshot.MAX_CHUNK_SIZE)
	}

	for _, path := range paths {
		file := lastSnapshot.Files[path]
		buf := <-sem
		go func(path string) {
			defer func() {
				sem <- buf
				remaining -= file.Size
				fmt.Printf("%s uploaded, %s remaining\n",
					utils.FormatSize(int(uploaded)),
					utils.FormatSize(int(remaining)))
			}()
			f, err := os.Open(path)
			if err != nil {
				log.Fatal(err)
			}
			defer f.Close()
			for _, chunk := range file.Chunks {
				for _, backend := range backends {
					exists, err := backend.Exists(int(chunk.Length), chunk.Hash)
					if err != nil {
						log.Fatal(err)
					}
					if !exists {
						fmt.Printf("uploading %s %d %s\n", path, chunk.Offset, chunk.Hash)
						o, err := f.Seek(chunk.Offset, 0)
						if err != nil || o != chunk.Offset {
							log.Fatal(err)
						}
						buf = buf[:chunk.Length]
						n, err := io.ReadFull(f, buf)
						if int64(n) != chunk.Length || err != nil {
							log.Fatal(err)
						}
						backend.Save(int(chunk.Length), chunk.Hash, bytes.NewReader(buf))
					} else {
						fmt.Printf("skip %s %d %s\n", path, chunk.Offset, chunk.Hash)
					}
				}
				remaining -= chunk.Length
				uploaded += chunk.Length
			}
		}(path)
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
