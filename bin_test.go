package hashbin

import (
	"bytes"
	"crypto/sha512"
	"math/rand"
	"testing"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func RunTest(bin *Bin, t *testing.T) {
	// basic save and fetch
	l := rand.Intn(1024)*1024 + 8
	data := make([]byte, l)
	for i := 0; i < l; i++ {
		data[i] = byte(rand.Intn(256))
	}
	hash := hashBytes(data)
	err := bin.Save(l, hash, WrapReader(bytes.NewReader(data)))
	if err != nil {
		t.Fatalf("save error: %v", err)
	}
	buf := new(bytes.Buffer)
	err = bin.Fetch(l, hash, WrapWriter(buf))
	if err != nil {
		t.Fatalf("fetch error: %v", err)
	}
	if !bytes.Equal(buf.Bytes(), data) {
		t.Fatal("fetch data incorrect")
	}
}

func hashBytes(bs []byte) []byte {
	hasher := sha512.New()
	hasher.Write(bs)
	return hasher.Sum(nil)
}
