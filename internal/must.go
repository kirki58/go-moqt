package internal

import (
	"fmt"
)

// Must is a generic helper that panics if err is not nil.
// It returns the value so you can use it in one line.

func Must[T any](val T, err error) T {
    if err != nil {
        panic(fmt.Sprintf("Must failed: %v", err))
    }
    return val
}