package snapshot

import (
	"compress/gzip"
	"crypto/sha512"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
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

func (self *SnapshotSet) Snapshot(cacheFile string, readCache bool, strategy int) error {
	var lastSnapshotFiles map[string]*File
	if len(self.Snapshots) > 0 {
		lastSnapshotFiles = self.Snapshots[len(self.Snapshots)-1].Files
	}
	snapshot := &Snapshot{
		Time:  time.Now(),
		Files: make(map[string]*File),
	}
	if readCache {
		f, err := os.Open(cacheFile)
		if err == nil {
			defer f.Close()
			err = gob.NewDecoder(f).Decode(&snapshot.Files)
			if err != nil {
				return errors.New(fmt.Sprintf("read cache error: %v", err))
			}
			fmt.Printf("read %d files from cache\n", len(snapshot.Files))
		}
	}

	infos := make([]os.FileInfo, 0)
	paths := make([]string, 0)
	err := collectFiles(self.Path, &infos, &paths)
	if err != nil {
		return err
	}
	cacheTimer := time.NewTimer(time.Second * 10)

	ignorePatterns := make([]*regexp.Regexp, 0)
	ignoreFilePath := filepath.Join(self.Path, ".fsignore")
	content, err := ioutil.ReadFile(ignoreFilePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	for _, pattern := range strings.Split(string(content), "\n") {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		pattern = "^" + pattern
		fmt.Printf("%s\n", pattern)
		ignorePatterns = append(ignorePatterns, regexp.MustCompilePOSIX(pattern))
	}

	for i, info := range infos {
		select {
		case <-cacheTimer.C: // write to cache
			fmt.Printf("saving cache\n")
			t := time.Now()
			f, err := os.Create(cacheFile + ".new")
			if err != nil {
				return errors.New(fmt.Sprintf("cannot create cache file: %v", err))
			}
			err = gob.NewEncoder(f).Encode(snapshot.Files)
			if err != nil {
				return errors.New(fmt.Sprintf("cannot write to cache file: %v", err))
			}
			f.Close()
			err = os.Rename(cacheFile+".new", cacheFile)
			if err != nil {
				return errors.New(fmt.Sprintf("cannot write to cache file: %v", err))
			}
			fmt.Printf("cache saved\n")
			cacheTimer.Reset(time.Now().Sub(t) * 10)
		default:
		}

		path := paths[i]
		relativePath := strings.TrimPrefix(path, self.Path)
		relativePath = strings.TrimPrefix(relativePath, "/")
		ignore := false
		for _, pattern := range ignorePatterns {
			if pattern.MatchString(relativePath) {
				fmt.Printf("ignore %s by %v\n", path, pattern)
				ignore = true
				break
			}
		}
		if ignore {
			continue
		}

		if _, ok := snapshot.Files[path]; readCache && ok {
			fmt.Printf("skip %s\n", path)
			continue
		}
		fmt.Printf("checking %s\n", path)

		file := &File{
			Path:    path,
			Size:    info.Size(),
			ModTime: info.ModTime(),
		}
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

func (self *SnapshotSet) Load(path string) error {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	z, err := gzip.NewReader(f)
	if err != nil {
		return nil
	}
	defer func() {
		z.Close()
		f.Close()
	}()
	err = gob.NewDecoder(z).Decode(&self.Snapshots)
	return err
}

func (self *SnapshotSet) Save(path string) error {
	f, err := os.OpenFile(path+".new", os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	z := gzip.NewWriter(f)
	err = gob.NewEncoder(z).Encode(self.Snapshots)
	if err != nil {
		z.Close()
		f.Close()
		return err
	}
	z.Close()
	f.Close()
	return os.Rename(path+".new", path)
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
