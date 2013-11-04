package main

import (
	"./snapshot"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
)

func runSnapshot() {
	var readCache bool
	var path string
	strategy := snapshot.FULL_HASH
	for i := 2; i < len(os.Args); i++ {
		arg := os.Args[i]
		if arg == "-c" || arg == "--continue" {
			readCache = true
		} else if arg == "-fc" || arg == "--fast-check" {
			strategy = snapshot.FAST_CHECK
		} else if arg == "-fh" || arg == "--fast-hash" {
			strategy = snapshot.FAST_HASH
		} else if arg[0] == '-' {
			fmt.Printf("unknown option %s\n", arg)
			os.Exit(0)
		} else {
			path = arg
		}
	}
	if path == "" {
		fmt.Printf("usage: %s snapshot [dir]\n", os.Args[0])
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
	snapshotFilePath := filepath.Join(dataDir, escapedPath+".snapshots")
	err = snapshotSet.Load(snapshotFilePath)
	if err != nil {
		log.Fatalf("cannot read snapshots from file: %v", err)
	}
	fmt.Printf("loaded %d snapshots from file\n", len(snapshotSet.Snapshots))

	cacheFilePath := filepath.Join(dataDir, escapedPath+".cache")
	err = snapshotSet.Snapshot(cacheFilePath, readCache, strategy)
	if err != nil {
		log.Fatalf("snapshot error: %v", err)
	}

	fmt.Printf("saving snapshots\n")
	err = snapshotSet.Save(snapshotFilePath)
	if err != nil {
		log.Fatalf("cannot save snapshot to file: %v", err)
	}
	fmt.Printf("snapshots saved\n")
}
