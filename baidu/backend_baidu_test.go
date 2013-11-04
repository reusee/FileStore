package baidu

import (
	"../hashbin"
	"testing"
)

func TestBaiduBackend(t *testing.T) {
	baidu, err := NewBaiduWithStringToken(
		BAIDU_DIR,
		BAIDU_TOKEN,
	)
	if err != nil {
		t.Fatal(err)
	}
	bin := hashbin.New(baidu)
	hashbin.RunTest(bin, t)
}
