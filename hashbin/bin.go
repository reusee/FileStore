package hashbin

import (
	"crypto/sha512"
	"errors"
	"fmt"
	"io"
)

type Bin struct {
	backend Backend
}

func New(backend Backend) *Bin {
	return &Bin{
		backend: backend,
	}
}

type Callback func(error) error

type Backend interface {
	NewWriter(length int, hash string) (io.Writer, Callback, error)
	NewReader(length int, hash string) (io.Reader, Callback, error)
	Exists(length int, hash string) (bool, error)
}

func (self *Bin) Save(length int, hash string, reader io.Reader) (err error) {
	writer, cb, err := self.backend.NewWriter(length, hash)
	if err != nil {
		return errors.New(fmt.Sprintf("backend error %v", err))
	}
	if cb != nil {
		defer func() {
			err = cb(err)
		}()
	}
	n, h, err := pipe(reader, writer)
	if err != nil {
		return err
	}
	if n != length || !(h == hash) {
		return errors.New(fmt.Sprintf("data not match %d-%s", length, hash))
	}
	return nil
}

func (self *Bin) Fetch(length int, hash string, writer io.Writer) (err error) {
	reader, cb, err := self.backend.NewReader(length, hash)
	if err != nil {
		return errors.New(fmt.Sprintf("backend error %v", err))
	}
	if cb != nil {
		defer func() {
			err = cb(err)
		}()
	}
	n, h, err := pipe(reader, writer)
	if err != nil {
		return err
	}
	if n != length {
		return errors.New(fmt.Sprintf("fetched data length not match, expected %d, get %d", length, n))
	}
	if !(h == hash) {
		return errors.New(fmt.Sprintf("fetched data hash not match, expected %x, get %x", hash, h))
	}
	return nil
}

func pipe(reader io.Reader, writer io.Writer) (int, string, error) {
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
				return readN, fmt.Sprintf("%x", hasher.Sum(nil)), errors.New(fmt.Sprintf("writer write error %v", err))
			}
		}
		if err == io.EOF {
			break
		} else if err != nil {
			return readN, fmt.Sprintf("%x", hasher.Sum(nil)), errors.New(fmt.Sprintf("reader read error %v", err))
		}
	}
	return readN, fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

func (self *Bin) Exists(length int, hash string) (bool, error) {
	return self.backend.Exists(length, hash)
}
