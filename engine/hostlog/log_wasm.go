//go:build js || wasip1

// Package hostlog sends UTF-8 lines from Go to the JS env (console) via wasmimport.
package hostlog

import "unsafe"

//go:wasmimport env backend_log_error
func rawBackendLogError(msgPtr uint32, msgLen int32)

//go:wasmimport env backend_log_warning
func rawBackendLogWarning(msgPtr uint32, msgLen int32)

func LogError(msg string) {
	mp, ml := wasmStrSpan(msg)
	rawBackendLogError(mp, ml)
}

func LogWarning(msg string) {
	mp, ml := wasmStrSpan(msg)
	rawBackendLogWarning(mp, ml)
}

func wasmStrSpan(s string) (uint32, int32) {
	if len(s) == 0 {
		return 0, 0
	}
	return uint32(uintptr(unsafe.Pointer(unsafe.StringData(s)))), int32(len(s))
}
