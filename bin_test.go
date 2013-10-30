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
	data := genRandBytes(1024 * 1024 * 4)
	hash := hashBytes(data)
	err := bin.Save(len(data), hash, bytes.NewReader(data))
	if err != nil {
		t.Fatalf("save error: %v", err)
	}
	buf := new(bytes.Buffer)
	err = bin.Fetch(len(data), hash, buf)
	if err != nil {
		t.Fatalf("fetch error: %v", err)
	}
	if !bytes.Equal(buf.Bytes(), data) {
		t.Fatal("fetch data incorrect")
	}

	// false length
	data = genRandBytes(1024 * 1024 * 4)
	hash = hashBytes(data)
	err = bin.Save(len(data)+1, hash, bytes.NewReader(data))
	if err == nil {
		t.Fatalf("saved wrong length data")
	}

	// false hash
	data = genRandBytes(1024 * 1024 * 4)
	hash = []byte{1, 2, 3}
	err = bin.Save(len(data), hash, bytes.NewReader(data))
	if err == nil {
		t.Fatalf("saved wrong hash data")
	}

	// false data
	data = genRandBytes(1024 * 1024 * 4)
	hash = hashBytes(data)
	err = bin.Save(len(data), hash, bytes.NewReader([]byte("foobar")))
	if err == nil {
		t.Fatalf("saved wrong data")
	}

	// fetch non exists
	err = bin.Fetch(1, genRandBytes(128), new(bytes.Buffer))
	if err == nil {
		t.Fatalf("fetched non exists data")
	}
}

func genRandBytes(max int) []byte {
	l := rand.Intn(max)
	data := make([]byte, l)
	for i := 0; i < l; i++ {
		data[i] = byte(rand.Intn(256))
	}
	return data
}

func hashBytes(bs []byte) []byte {
	hasher := sha512.New()
	hasher.Write(bs)
	return hasher.Sum(nil)
}
