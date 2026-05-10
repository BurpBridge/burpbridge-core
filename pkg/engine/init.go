package engine

import (
	"runtime/debug"
)

func init() {
	debug.SetMemoryLimit(10 * 1024 * 1024)
}
