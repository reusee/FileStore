package snapshot

import (
	crand "crypto/rand"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

func TestSnapshot(t *testing.T) {
	//path, err := ioutil.TempDir("", "")
	//if err != nil {
	//	t.Fatalf("%v", err)
	//}
	//err = generateRandomFileOrDirs(path, 4)
	//if err != nil {
	//	t.Fatalf("%v", err)
	//}
	path := "./testdata"
	set, err := New(path)
	if err != nil {
		t.Fatalf("%v", err)
	}
	err = set.Snapshot()
	if err != nil {
		t.Fatalf("%v", err)
	}
	err = set.Snapshot()
	if err != nil {
		t.Fatalf("%v", err)
	}
}

func generateRandomFileOrDirs(dir string, depth int) error {
	if depth == 0 {
		return nil
	}
	n := 4 + rand.Intn(4)
	for i := 0; i < n; i++ {
		path := filepath.Join(dir, fmt.Sprintf("%d", i))
		print(".")
		os.Stdout.Sync()
		if rand.Intn(2) == 0 { // dir
			os.Mkdir(path, 0755)
			generateRandomFileOrDirs(path, depth-1)
		} else { // file
			f, err := os.Create(path)
			if err != nil {
				return err
			}
			io.CopyN(f, crand.Reader, int64(rand.Intn(64*1024*1024)))
			f.Close()
		}
	}
	return nil
}
