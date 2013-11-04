package baidu

import (
	"../hashbin"
	"testing"
)

func TestBaiduBackend(t *testing.T) {
	baidu, err := NewBaidu(
		BAIDU_KEY,
		BAIDU_SECRET,
		BAIDU_DIR,
		BAIDU_TOKEN,
	)
	if err != nil {
		t.Fatal(err)
	}
	bin := hashbin.NewBin(baidu)
	hashbin.RunTest(bin, t)
}
