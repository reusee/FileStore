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
	"net/url"
	"os"
	"path/filepath"
	"sort"
)

func runUpload() {
	var path string
	for i := 2; i < len(os.Args); i++ {
		arg := os.Args[i]
		if arg[0] == '-' {
			fmt.Printf("unknown option %s\n", arg)
			os.Exit(0)
		} else {
			path = arg
		}
	}
	if path == "" {
		fmt.Printf("usage: %s upload [dir]\n", os.Args[0])
		os.Exit(0)
	}

	path, err := filepath.Abs(path)
	if err != nil {
		log.Fatalf("invalid path: %v", err)
	}

	snapshotSet, err := snapshot.New(path)
	if err != nil {
		log.Fatalf("cannot create snapshot set: %v", err)
	}
	escapedPath := url.QueryEscape(path)
	snapshotFilePath := filepath.Join(DATADIR, escapedPath+".snapshots")
	err = snapshotSet.Load(snapshotFilePath)
	if err != nil {
		log.Fatalf("cannot read snapshots from file: %v", err)
	}
	fmt.Printf("loaded %d snapshots from file\n", len(snapshotSet.Snapshots))

	backends := make([]*hashbin.Bin, 0)

	// baidu
	b, err := getBaiduBackend()
	if err != nil {
		log.Fatal(err)
	}
	backends = append(backends, b)

	if len(snapshotSet.Snapshots) == 0 {
		fmt.Printf("no snapshot\n")
		os.Exit(0)
	}
	lastSnapshot := snapshotSet.Snapshots[len(snapshotSet.Snapshots)-1]

	filePaths := make([]string, 0, len(lastSnapshot.Files))
	for filePath := range lastSnapshot.Files {
		filePaths = append(filePaths, filePath)
	}
	sort.Strings(filePaths)

	semSize := 4
	sem := make(chan bool, semSize)
	for i := 0; i < semSize; i++ {
		sem <- true
	}

	for _, filePath := range filePaths {
		file := lastSnapshot.Files[filePath]
		<-sem
		go func(filePath string, file *snapshot.File) {
			defer func() {
				sem <- true
			}()
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
					f, err := os.Open(filePath)
					if err != nil {
						log.Fatal(err)
					}
					o, err := f.Seek(chunk.Offset, 0)
					if err != nil || o != chunk.Offset {
						log.Fatal(err)
					}
					buf := make([]byte, chunk.Length)
					n, err := io.ReadFull(f, buf)
					if int64(n) != chunk.Length || err != nil {
						log.Fatal(err)
					}
					err = backend.Save(int(chunk.Length), chunk.Hash, bytes.NewReader(buf))
					if err != nil {
						log.Fatal(err)
					}
				}
			}
		}(filePath, file)
	}
}

func getBaiduBackend() (*hashbin.Bin, error) {
	var dir string
	var token oauth.Token
	err := REGISTER.Get("baidu_dir", &dir)
	if err != nil {
		return nil, err
	}
	err = REGISTER.Get("baidu_token", &token)
	if err != nil {
		return nil, err
	}
	keyCacheFilePath := filepath.Join(DATADIR, "baidu.keys")
	b, err := baidu.New(dir, &token, keyCacheFilePath)
	if err != nil {
		return nil, err
	}
	return hashbin.New(b), nil
}
