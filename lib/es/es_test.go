package es

import (
	"fmt"
	"testing"
)

func TestEs(t *testing.T) {
	t.Log(GetEsIndex(fmt.Sprintf(ESIndexPrefix, "1000"), "2024-02-01 00:00:00", "2024-03-30 23:00:00"))
}
