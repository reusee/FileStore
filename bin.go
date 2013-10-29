package hashbin

import (
	"bytes"
	"crypto/sha512"
	"errors"
	"fmt"
	"io"
)

type Bin struct {
	backend Backend
}

func NewBin(backend Backend) *Bin {
	return &Bin{
		backend: backend,
	}
}

type Backend interface {
	NewWriter(length int, hash []byte) (io.WriteCloser, error)
	NewReader(length int, hash []byte) (io.ReadCloser, error)
	Exists(length int, hash []byte) (bool, error)
}

func (self *Bin) Save(length int, hash []byte, reader io.ReadCloser) error {
	writer, err := self.backend.NewWriter(length, hash)
	if err != nil {
		return errors.New(fmt.Sprintf("backend error %v", err))
	}
	n, h, err := pipe(reader, writer)
	if err != nil {
		return err
	}
	if n != length || !bytes.Equal(h, hash) {
		return errors.New(fmt.Sprintf("data not match %d-%s", length, hash))
	}
	return nil
}

func (self *Bin) Fetch(length int, hash []byte, writer io.WriteCloser) error {
	reader, err := self.backend.NewReader(length, hash)
	if err != nil {
		return errors.New(fmt.Sprintf("backend error %v", err))
	}
	n, h, err := pipe(reader, writer)
	if err != nil {
		return err
	}
	if n != length || !bytes.Equal(h, hash) {
		return errors.New(fmt.Sprintf("data not match %d-%s", length, hash))
	}
	return nil
}

func pipe(reader io.ReadCloser, writer io.WriteCloser) (int, []byte, error) {
	defer reader.Close()
	defer writer.Close()
	readN := 0
	buf := make([]byte, 1*1024*1024)
	hasher := sha512.New()
	var n int
	var err error
	for {
		n, err = reader.Read(buf)
		if n > 0 {
			readN += n
			hasher.Write(buf[:n])
			_, err = writer.Write(buf[:n])
			if err != nil {
				return readN, hasher.Sum(nil), errors.New(fmt.Sprintf("writer write error %v", err))
			}
		}
		if err == io.EOF {
			break
		} else if err != nil {
			return readN, hasher.Sum(nil), errors.New(fmt.Sprintf("reader read error %v", err))
		}
	}
	return readN, hasher.Sum(nil), nil
}

func (self *Bin) Exists(length int, hash []byte) (bool, error) {
	return self.backend.Exists(length, hash)
}

type wrapReader struct {
	io.Reader
}

func (self *wrapReader) Close() error {
	return nil
}

func WrapReader(r io.Reader) *wrapReader {
	return &wrapReader{r}
}

type wrapWriter struct {
	io.Writer
}

func (self *wrapWriter) Close() error {
	return nil
}

func WrapWriter(r io.Writer) *wrapWriter {
	return &wrapWriter{r}
}
