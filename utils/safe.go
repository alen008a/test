package utils

import (
	"fmt"
	"runtime/debug"
)

func SafeGo(f func()) {
	go func() {
		defer func() {
			fmt.Println("stacktrace from panic: \n" + string(debug.Stack()))
		}()
		f()
	}()
}
