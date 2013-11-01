package snapshot

import (
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const MAX_CHUNK_SIZE = 16 * 1024 * 1024

type SnapshotSet struct {
	Snapshots []*Snapshot
	Path      string
}

type Snapshot struct {
	Time  time.Time
	Files map[string]*File
}

type File struct {
	Path    string
	Size    int64
	ModTime time.Time
	Chunks  []*Chunk
}

type Chunk struct {
	Offset int64
	Length int64
	Hash   string
}

const (
	FAST_CHECK = iota
	FAST_HASH
	FULL_HASH
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func New(path string) (*SnapshotSet, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, errors.New(fmt.Sprintf("not a directory: %s", path))
	}
	return &SnapshotSet{
		Path: path,
	}, nil
}

func (self *SnapshotSet) Snapshot() error {
	var lastSnapshotFiles map[string]*File
	if len(self.Snapshots) > 0 {
		lastSnapshotFiles = self.Snapshots[len(self.Snapshots)-1].Files
	}
	snapshot := &Snapshot{
		Time:  time.Now(),
		Files: make(map[string]*File),
	}

	infos := make([]os.FileInfo, 0)
	paths := make([]string, 0)
	err := collectFiles(self.Path, &infos, &paths)
	if err != nil {
		return err
	}

	for i, info := range infos {
		fmt.Printf("%s\n", paths[i])
		file := &File{
			Path:    paths[i],
			Size:    info.Size(),
			ModTime: info.ModTime(),
		}
		strategy := FAST_HASH //TODO read from file
		err = file.getChunks(lastSnapshotFiles, strategy)
		if err != nil {
			return err
		}
		snapshot.Files[file.Path] = file
	}

	self.Snapshots = append(self.Snapshots, snapshot)
	return nil
}

func makeSemaphore(n int) chan int {
	ret := make(chan int, n)
	for i := 0; i < n; i++ {
		ret <- 1
	}
	return ret
}

func collectFiles(top string, infos *[]os.FileInfo, paths *[]string) error {
	f, err := os.Open(top)
	if err != nil {
		if err.(*os.PathError).Err.Error() == "permission denied" {
			return nil
		}
		return err
	}
	subs, err := f.Readdir(-1)
	if err != nil {
		return err
	}
	for _, info := range subs {
		if info.IsDir() {
			err = collectFiles(filepath.Join(top, info.Name()), infos, paths)
			if err != nil {
				return err
			}
		} else {
			*infos = append(*infos, info)
			*paths = append(*paths, filepath.Join(top, info.Name()))
		}
	}
	return nil
}

func (self *File) getChunks(lastSnapshotFiles map[string]*File, strategy int) error {
	if old, ok := lastSnapshotFiles[self.Path]; ok {
		if strategy == FAST_CHECK {
			if old.Size == self.Size && old.ModTime == self.ModTime {
				self.Chunks = old.Chunks
				return nil
			}
		} else if strategy == FAST_HASH {
			oldChunk := old.GetChunk(0)
			err := self.HashChunks(1)
			newChunk := self.GetChunk(0)
			if err != nil {
				return err
			}
			if oldChunk != nil && oldChunk.Length == newChunk.Length && oldChunk.Hash == newChunk.Hash {
				self.Chunks = old.Chunks
				return nil
			}
		}
	}
	return self.HashChunks(-1)
}

func (self *File) GetChunk(offset int64) *Chunk {
	for _, chunk := range self.Chunks {
		if chunk.Offset == offset {
			return chunk
		}
	}
	return nil
}

func (self *File) HashChunks(maxChunks int) error {
	offset := int64(0)
	buf := make([]byte, MAX_CHUNK_SIZE)
	var n int
	var err error
	f, err := os.Open(self.Path)
	if err != nil {
		return err
	}
	self.Chunks = make([]*Chunk, 0, 1)
	hasher := sha512.New()
	c := 0
	for {
		n, err = f.Read(buf)
		if n > 0 {
			hasher.Reset()
			hasher.Write(buf[:n])
			chunk := &Chunk{
				Offset: offset,
				Length: int64(n),
				Hash:   hex.EncodeToString(hasher.Sum(nil)),
			}
			offset += int64(n)
			self.Chunks = append(self.Chunks, chunk)
		}
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		c += 1
		if c == maxChunks {
			return nil
		}
	}
	return nil
}
