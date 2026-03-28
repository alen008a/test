package utils

import (
	"fmt"
	"strings"
	"testing"
)

func TestUtils(t *testing.T) {
	startAt := GetBjNowTime().Format(TimeTBjFormat)
	t.Log(startAt)
	endAtTime, err := BjTBarFmtTimeFormat(startAt, TimeTBjFormat)
	if err != nil {
		t.Log(err)
	} else {
		t.Log(endAtTime)
		t.Log(endAtTime.Format(TimeTBjFormat))
	}
	t.Log(fmt.Sprintf("%s+08:00", strings.Replace("2024-03-07 19:46:27", " ", "T", -1)))
}
