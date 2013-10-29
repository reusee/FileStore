package hashbin

import (
	"testing"
)

func TestMemBackend(t *testing.T) {
	mem := NewMembin()
	bin := NewBin(mem)
	RunTest(bin, t)
}
