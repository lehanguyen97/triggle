package main

/*
#cgo CFLAGS: -I../engine/include
#include <stdint.h>
#include <stddef.h>
#include <e/engine_api.h>
#include <e/game_api.h>
*/
import "C"
import "unsafe"

type MeshData struct {
	vertices *float32
	nv       uint
	indices  *uint16
	ni       uint
}

type RenderArg struct {
	bindId       int32
	shaderParams uintptr
}

type Engine uintptr

func NewEngine() Engine {
	return Engine(C.engine_init())
}

func (e Engine) RegisterMesh(vertices []float32, indices []uint16) int32 {
	data := MeshData{
		vertices: &vertices[:1][0],
		nv:       uint(len(vertices)),
		indices:  &indices[:1][0],
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
