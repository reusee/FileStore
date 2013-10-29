package hashbin

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

type Membin struct {
	store map[string][]byte
}

func NewMembin() *Membin {
	return &Membin{
		store: make(map[string][]byte),
	}
}

type membuffer struct {
	m   map[string][]byte
	key string
	buf *bytes.Buffer
}

func (self *membuffer) Write(buf []byte) (int, error) {
	return self.buf.Write(buf)
}

func (self *membuffer) Close() error {
	self.m[self.key] = self.buf.Bytes()
	return nil
}

func (self *Membin) NewWriter(length int, hash []byte) (io.WriteCloser, error) {
	return &membuffer{
		m:   self.store,
		key: fmt.Sprintf("%d-%x", length, hash),
		buf: new(bytes.Buffer),
	}, nil
}

type closingReader struct {
	*bytes.Reader
}

func (self closingReader) Close() error {
	return nil
}

func (self *Membin) NewReader(length int, hash []byte) (io.ReadCloser, error) {
	if v, ok := self.store[fmt.Sprintf("%d-%x", length, hash)]; ok {
		return closingReader{bytes.NewReader(v)}, nil
	}
	return nil, errors.New("not exists")
}

func (self *Membin) Exists(length int, hash []byte) (bool, error) {
	if _, ok := self.store[fmt.Sprintf("%d-%x", length, hash)]; ok {
		return true, nil
	}
	return false, nil
}
