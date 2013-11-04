package register

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"os"
)

type Register struct {
	r    map[string][]byte
	path string
}

func NewRegister(path string) (*Register, error) {
	r := &Register{
		r:    make(map[string][]byte),
		path: path,
	}
	f, err := os.Open(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, errors.New(fmt.Sprintf("error when open register file: %v", err))
	}
	if err == nil {
		err = gob.NewDecoder(f).Decode(&r.r)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("error when decode register file: %v", err))
		}
	}
	return r, nil
}

func (self *Register) Set(key string, value interface{}) error {
	buf := new(bytes.Buffer)
	err := gob.NewEncoder(buf).Encode(value)
	if err != nil {
		return err
	}
	self.r[key] = buf.Bytes()
	err = self.save()
	if err != nil {
		return err
	}
	return nil
}

func (self *Register) save() error {
	f, err := os.Create(self.path + ".new")
	if err != nil {
		return err
	}
	err = gob.NewEncoder(f).Encode(self.r)
	if err != nil {
		return err
	}
	err = os.Rename(self.path+".new", self.path)
	if err != nil {
		return err
	}
	return nil
}

func (self *Register) Get(key string, target interface{}) error {
	value, ok := self.r[key]
	if !ok {
		return errors.New(fmt.Sprintf("key not found: %s", key))
	}
	err := gob.NewDecoder(bytes.NewReader(value)).Decode(target)
	if err != nil {
		return err
	}
	return nil
}
