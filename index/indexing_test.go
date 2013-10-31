package indexing

import (
	"encoding/gob"
	"fmt"
	"os"
	"testing"
)

func TestIndex(t *testing.T) {
	index := NewIndex()

	f, err := os.Open("index")
	if err == nil {
		err = gob.NewDecoder(f).Decode(&index)
		if err != nil {
			t.Fatal(err)
		}
		f.Close()
	} else {
		index.Index("/media/store/inoue_marina", FULL_HASH)
	}

	index.Index("/media/store/inoue_marina", NO_HASH)

	for name, files := range index.Files {
		fmt.Printf("%s\n", name)
		for _, f := range files {
			fmt.Printf("  %v\n", f.Version)
		}
	}

	f, err = os.OpenFile("index", os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		t.Fatal(err)
	}
	err = gob.NewEncoder(f).Encode(index)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
}
