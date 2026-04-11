# Reference: graphics.gd Wasm2Wasm Architecture

Source: `~/ws-local/graphics.gd/`

## How It Works

graphics.gd runs Go game code against Godot engine. Both compile to separate WASM modules for browser. They solved the same dual-platform (cgo + wasm) problem.

### Memory Problem
Two WASM modules = two separate linear memories. Pointers from one are meaningless in the other. Engine functions expect pointers in engine memory.

### Solution: bulk_copy Bridge

1. Engine exports `memory_malloc(size) -> ptr` — allocates in engine heap
2. JS glue provides `bulk_copy(dst_engine, src_game, len)` — copies bytes between the two WASM memories (JS can access both `WebAssembly.Memory` objects)
3. Go allocates in engine memory, bulk_copies data there, then calls engine functions with engine-memory pointers
4. For cgo: same API, pointers are real C pointers, bulk_copy = memcpy

### Key Files

| File | Purpose |
|------|---------|
| `internal/ring/ring.go` | Ring buffer (256 entries) for batching method calls |
| `internal/ring/ring_wasm.go` | WASM flush: bulk_copy ring to shadow ring in engine memory, then flush |
| `internal/ring/ring_cgo.go` | CGO flush: direct C call with pointer to ring entries |
| `startup/startup_wasm_web.go` | `go:wasmimport gd bulk_copy`, `go:wasmimport gd memory_malloc` |
| `startup/startup_wasm_exports.go` | 280+ `go:wasmexport` functions (game callbacks to engine) |
| `startup/startup_cgo_v2.go` | CGO `//export` equivalents |
| `startup/startup_cgo_v2.c` | Unified C backend — same .c for Emscripten and native cgo |
| `internal/gdextension/ptr_js.go` | `type Pointer = uint32` (32-bit wasm) |
| `internal/gdextension/ptr_64.go` | `type Pointer = uintptr` (64-bit native) |
| `internal/gdmemory/memory_wasm.go` | Read from engine memory: Load.Uint32/Uint16/Byte per-word |
| `internal/gdmemory/memory.go` | CopyArguments: writes Go data into engine memory word-by-word |

### Platform Switching

Build tags: `//go:build cgo` vs `//go:build js` vs `//go:build wasip1`

Pointer type alias changes per platform:
```go
// ptr_js.go (//go:build js)
type Pointer = uint32

// ptr_64.go (//go:build amd64 || arm64)
type Pointer = uintptr
```

Function implementations swapped at init() via function pointers on a Host struct:
```go
// startup_wasm_web.go
func init() {
    gdextension.Host.Memory.Malloc = func(size int) gdextension.Pointer {
        return gdextension.Pointer(wasm_gd_memory_malloc(uint32(size)))
    }
}
```

### WASM Import Examples
```go
//go:wasmimport gd memory_malloc
func wasm_gd_memory_malloc(size uint32) uint32

//go:wasmimport gd bulk_copy
func wasm_gd_bulk_copy(godot_dst uint32, go_src uint32, length uint32)

//go:wasmimport gd object_unsafe_call
func wasm_gd_object_unsafe_call(obj uint32, method uint32, result uint32, shape_hi uint32, shape_lo uint32, args uint32)
```

### WASM Export Examples
```go
//go:wasmexport go_on_callable_call
func go_on_callable_call(p0 uint32, p1 uint32, p2 int32, p3 uint32, p4 uint32) { ... }
```

### Ring Buffer Optimization
Not needed for triggle (low call volume), but worth noting: graphics.gd batches method calls in a ring buffer, bulk_copies the whole buffer to engine memory in one shot, then tells engine to process the batch. Void calls have near-zero overhead.
