package hashbin

import (
	"testing"
)

func TestMemBackend(t *testing.T) {
	mem := NewMembin()
	bin := New(mem)
	RunTest(bin, t)
}
