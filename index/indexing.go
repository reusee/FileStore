package indexing

import (
	"bytes"
	"crypto/sha512"
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"
)

const MAX_CHUNK_SIZE = 8 * 1024 * 1024

const (
	NO_HASH = iota
	FAST_HASH
	FULL_HASH
)

func (self *Index) Index(path string, flag int) error {
	filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if versions, ok := self.Files[path]; ok {
			switch flag {
			case NO_HASH:
				for _, v := range versions {
					if v.ModTime == info.ModTime() && v.Size == info.Size() {
						return nil
					}
				}
			case FAST_HASH:
				for _, v := range versions {
					chunks, _, _, err := hashChunks(path, 1)
					if err != nil {
						return nil
					}
					firstChunk, err := v.getChunk(0)
					if err == nil && firstChunk.Length == chunks[0].Length && bytes.Equal(firstChunk.Hash, chunks[0].Hash) {
						return nil
					}
				}
			}
		}
		file := File{
			Version: time.Now(),
			Name:    path,
			Size:    info.Size(),
			ModTime: info.ModTime(),
		}
		chunks, n, h, err := hashChunks(path, -1)
		if err != nil {
			return nil
		}
		if n != file.Size {
			return nil
		}
		file.Chunks = chunks
		for _, v := range self.Files[file.Name] {
			if bytes.Equal(v.Hash, h) {
				return nil
			}
		}
		file.Hash = h
		self.Files[file.Name] = append(self.Files[file.Name], file)
		return nil
	})
	return nil
}

func (self *File) getChunk(offset int64) (Chunk, error) {
	for _, t := range self.Chunks {
		if t.Offset == offset {
			return t, nil
		}
	}
	return Chunk{}, errors.New("not exists")
}

func hashChunks(path string, max int) ([]Chunk, int64, []byte, error) {
	offset := int64(0)
	buf := make([]byte, MAX_CHUNK_SIZE)
	var n int
	var err error
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, nil, err
	}
	chunks := make([]Chunk, 0, 1)
	hasher := sha512.New()
	h := sha512.New()
	c := 0
	for {
		n, err = f.Read(buf)
		if n > 0 {
			hasher.Reset()
			hasher.Write(buf[:n])
			h.Write(buf[:n])
			chunk := Chunk{
				Offset: offset,
				Length: int64(n),
				Hash:   hasher.Sum(nil),
			}
			offset += int64(n)
			chunks = append(chunks, chunk)
		}
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, 0, nil, err
		}
		c += 1
		if c == max {
			return chunks, offset, h.Sum(nil), nil
		}
	}
	return chunks, offset, h.Sum(nil), nil
}
