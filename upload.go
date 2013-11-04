package main

import (
	"./baidu"
	"./hashbin"
	"./snapshot"
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

	filePaths := make([]string, 0, len(lastSnapshot.Files))
	for filePath := range lastSnapshot.Files {
		filePaths = append(filePaths, filePath)
	}
	sort.Strings(filePaths)

	semSize := 4
	sem := make(chan []byte, semSize)
	for i := 0; i < semSize; i++ {
		sem <- make([]byte, snapshot.MAX_CHUNK_SIZE)
	}

	for _, filePath := range filePaths {
		file := lastSnapshot.Files[filePath]
		buf := <-sem
		go func(filePath string) {
			defer func() {
				sem <- buf
			}()
			f, err := os.Open(filePath)
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
					if exists {
						fmt.Printf("skip %s %d %s\n", filePath, chunk.Offset, chunk.Hash)
						return
					}
					fmt.Printf("uploading %s %d %s\n", filePath, chunk.Offset, chunk.Hash)
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
				}
			}
		}(filePath)
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
