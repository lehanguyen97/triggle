//go:build !js && !wasip1

package backend

// Ptr is a native pointer passed across CGO.
type Ptr uintptr
