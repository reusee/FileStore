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

func (self *Membin) NewWriter(length int, hash []byte) (io.Writer, Callback, error) {
	buf := new(bytes.Buffer)
	return buf, func(err error) error {
		if err != nil {
			return err
		}
		self.store[fmt.Sprintf("%d-%x", length, hash)] = buf.Bytes()
		return nil
	}, nil
}

func (self *Membin) NewReader(length int, hash []byte) (io.Reader, Callback, error) {
	if v, ok := self.store[fmt.Sprintf("%d-%x", length, hash)]; ok {
		return bytes.NewReader(v), nil, nil
	}
	return nil, nil, errors.New("not exists")
}

func (self *Membin) Exists(length int, hash []byte) (bool, error) {
	if _, ok := self.store[fmt.Sprintf("%d-%x", length, hash)]; ok {
		return true, nil
	}
	return false, nil
}
