package main

import (
	"./utils"
	"fmt"
	"log"
	"os"
	"sort"
)

func (self *App) runList() {
	if len(self.snapshotSet.Snapshots) == 0 {
		fmt.Printf("no snapshot\n")
		os.Exit(0)
	}
	lastSnapshot := self.snapshotSet.Snapshots[len(self.snapshotSet.Snapshots)-1]

	paths := make([]string, 0, len(lastSnapshot.Files))
	for path := range lastSnapshot.Files {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	b, err := self.getBaiduBackend()
	if err != nil {
		log.Fatal(err)
	}

	var totalSize int64
	for _, path := range paths {
		file := lastSnapshot.Files[path]
		complete := true
		for _, chunk := range file.Chunks {
			e, err := b.Exists(int(chunk.Length), chunk.Hash)
			if err != nil {
				log.Fatal(err)
			}
			if !e {
				complete = false
			}
		}
		//if complete {
		_ = complete
		fmt.Printf("%s\n", path)
		totalSize += file.Size
		//}
	}

	fmt.Printf("%s\n", utils.FormatSize(int(totalSize)))
}
