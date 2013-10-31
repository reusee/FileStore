package indexing

import (
	"time"
)

type Index struct {
	Files map[string][]File
}

func NewIndex() *Index {
	return &Index{
		Files: make(map[string][]File),
	}
}

type File struct {
	Version time.Time
	Hash    []byte
	Name    string
	Size    int64
	ModTime time.Time
	Chunks  []Chunk
}

type Chunk struct {
	Offset int64
	Length int64
	Hash   []byte
}
