package main

import (
	"./snapshot"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
)

func (self *App) runList() {
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
		fmt.Printf("usage: %s list [dir]\n", os.Args[0])
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
	snapshotFilePath := filepath.Join(self.dataDir, escapedPath+".snapshots")
	err = snapshotSet.Load(snapshotFilePath)
	if err != nil {
		log.Fatalf("cannot read snapshots from file: %v", err)
	}
	fmt.Printf("loaded %d snapshots from file\n", len(snapshotSet.Snapshots))
}
