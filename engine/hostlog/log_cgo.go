//go:build !js && !wasip1

// Package hostlog sends UTF-8 lines from Go to the native host (stderr) via game_api.h.
package hostlog

/*
#cgo CFLAGS: -I../../backend/include
#include <e/game_api.h>
*/
import "C"

import "unsafe"

func LogError(msg string) {
	msgP, msgL := cStrSpan(msg)
	C.backend_log_error(msgP, msgL)
}

func LogWarning(msg string) {
	msgP, msgL := cStrSpan(msg)
	C.backend_log_warning(msgP, msgL)
}

func cStrSpan(s string) (*C.char, C.int32_t) {
	if len(s) == 0 {
		return nil, 0
	}
	return (*C.char)(unsafe.Pointer(unsafe.StringData(s))), C.int32_t(len(s))
}
