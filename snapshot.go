package main

import (
	"./snapshot"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func (self *App) runSnapshot() {
	var readCache bool
	strategy := snapshot.FULL_HASH
	for _, flag := range self.flags {
		if flag == "-c" || flag == "--continue" {
			readCache = true
		} else if flag == "-fc" || flag == "--fast-check" {
			strategy = snapshot.FAST_CHECK
		} else if flag == "-fh" || flag == "--fast-hash" {
			strategy = snapshot.FAST_HASH
		} else if flag[0] == '-' {
			fmt.Printf("unknown option %s\n", flag)
			os.Exit(0)
		}
	}

	cacheFilePath := filepath.Join(self.dataDir, self.escapedPath+".cache")
	err := self.snapshotSet.Snapshot(cacheFilePath, readCache, strategy)
	if err != nil {
		log.Fatalf("snapshot error: %v", err)
	}

	fmt.Printf("saving snapshots\n")
	err = self.snapshotSet.Save(self.snapshotFilePath)
	if err != nil {
		log.Fatalf("cannot save snapshot to file: %v", err)
	}
	fmt.Printf("snapshots saved\n")
}
