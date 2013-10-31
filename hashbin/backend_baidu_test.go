package hashbin

import (
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
	bin := NewBin(baidu)
	RunTest(bin, t)
}
