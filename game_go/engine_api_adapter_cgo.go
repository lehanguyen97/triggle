//go:build !js && !wasm

package main

/*
#cgo CFLAGS: -I../engine/include
#include <stdint.h>
#include <stddef.h>
#include <e/engine_api.h>
#include <e/game_api.h>
*/
import "C"
import (
	"unsafe"
)

type MeshData struct {
	vertices unsafe.Pointer
	nv       uint
	indices  unsafe.Pointer
	ni       uint
}

type RenderArg struct {
	bindId       int32
	shaderParams uintptr
}

type Engine int32

func NewEngine() Engine {
	return Engine(C.engine_init())
}

func (e Engine) RegisterMesh(vertices []float32, indices []uint16) int32 {
	cVertices := C.CBytes(unsafe.Slice((*byte)(unsafe.Pointer(&vertices[0])), len(vertices)*4))
	cIndices := C.CBytes(unsafe.Slice((*byte)(unsafe.Pointer(&indices[0])), len(indices)*2))
	data := MeshData{
		vertices: cVertices,
		nv:       uint(len(vertices)),
		indices:  cIndices,
		ni:       uint(len(indices)),
	}
	return int32(C.engine_register_mesh(C.engine_t(e), *(*C.MeshData)(unsafe.Pointer(&data))))
}

func (e Engine) Render(bindId int32, shaderParams uintptr) int32 {
	data := RenderArg{
		bindId:       bindId,
		shaderParams: shaderParams,
	}
	return int32(C.engine_render(C.engine_t(e), *(*C.RenderArg)(unsafe.Pointer(&data))))
}

func (e Engine) CleanUp() int32 {
	return int32(C.engine_cleanup(C.engine_t(e)))
}
