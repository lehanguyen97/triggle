//go:build js || wasm

package main

import (
	"syscall/js"
	"unsafe"
)

type MeshData struct {
	vertices uintptr
	nv       uint
	indices  uintptr
	ni       uint
}

type Engine struct {
	handle int
	Module js.Value
}

func NewEngine() Engine {
	return Engine{
		js.Global().Call("_engine_init").Int(),
		js.Global().Get("Module"),
	}
}

func (e Engine) copyToWasm(data []byte) uintptr {
	ptr := uintptr(js.Global().Call("_malloc", len(data)).Int())
	buf := e.Module.Get("HEAPU8").Call("subarray", ptr, ptr+uintptr(len(data)))
	js.CopyBytesToJS(buf, data)
	return ptr
}

func (e Engine) RegisterMesh(vertices []float32, indices []uint16) int32 {
	verticesPtr := e.copyToWasm(unsafe.Slice((*byte)(unsafe.Pointer(&vertices[0])), len(vertices)*4))
	indicesPtr := e.copyToWasm(unsafe.Slice((*byte)(unsafe.Pointer(&indices[0])), len(indices)*4))
	js.Global().Call("_engine_register_mesh", e)
}
