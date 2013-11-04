package utils

import (
	"fmt"
)

func FormatSize(n int) string {
	units := []string{"b", "k", "m", "g", "t"}
	i := 0
	ret := ""
	for n > 0 && i < len(units) {
		if n%1024 > 0 {
			ret = fmt.Sprintf("%d%s", n%1024, units[i]) + ret
		}
		n = n / 1024
		i += 1
	}
	return ret
}
