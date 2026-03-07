# Wasm2Wasm Design

## Goal

Go owns all game logic. C++/Sokol is a thin graphics backend. Same Go game code compiles for native (CGO) and browser (WASM dual-module) via build tags. No syscall/js.

## Current State (branch: wasm-go)

### What Works

- Spinning cube + ground plane with Phong shading + shadow mapping
- Handle-based engine API: mesh, shader, pipeline, image, sampler, pass (all int32 IDs)
- Native: CGO `c-archive`, direct C calls, `Ptr = uintptr`
- Browser: two WASM modules (Emscripten engine + Go wasip1 reactor), `go:wasmimport`/`go:wasmexport`, `Ptr = uint32`, JS `bulk_copy` bridge
- Binary descriptor encoding: Go builds blobs in `gfx.go`, C++ parses with `BlobReader` in `engine_api_impl.cpp`
- Two-pass rendering: shadow depth pass → main phong pass with shadow sampling

### Architecture

```
Native:
  Go (c-archive) → libgamego.a
  C++ (Sokol) → links libgamego.a → triggle executable
  Calls: CGO (direct C function calls, shared heap)

Browser:
  Go (wasip1, c-shared) → game.wasm (reactor)
  C++ (Emscripten) → triggle.js + triggle.wasm
  Calls: go:wasmimport/wasmexport across two WASM modules
  Memory: two linear memories, JS bulk_copy bridge
  WASI: minimal polyfills in triggle.html (fd_write, clock_time_get, etc.)
```

### Data Flow

```
Game.update()
  → engine.BulkCopy(uniformPtr, &goData, size)   // copy Go data to engine memory
  → engine.PassBegin(pass, clearDepth)
  → engine.ApplyPipeline(pipeline)
  → engine.BindMesh(mesh)
  → engine.ApplyUniforms(slot, uniformPtr, size)
  → engine.DrawElements(base, count, instances)
  → engine.PassEnd()
  → engine.Commit()
```

Mesh upload: `engine.Malloc` → `engine.BulkCopy` → `engine.MeshCreate` → `engine.Free`

### Known Issues

1. **Binary descriptors fragile** — no version field, Go encoder tightly coupled to C++ BlobReader
2. **No resource validation** — stale/invalid handles → undefined behavior
3. **Hardcoded uniform sizes** — `Malloc(192)` etc, no sizeof derivation
4. **Event API divergent** — CGO uses GEvent struct, WASM uses flattened int32 args
5. **bulk_copy one-way** — game→engine only, no readback
6. **No error messages** — all failures return -1

---

## Next: Host Struct Pattern

Adopt graphics.gd's pattern: replace per-file adapter functions with a single `Host` struct with function fields, initialized per platform at startup.

### Current (adapter functions per build tag)

```go
// engine_api_adapter_cgo.go — one file
func NewEngine() Engine { return Engine{handle: int32(C.engine_init())} }
func (e Engine) Malloc(size int32) Ptr { return Ptr(C.engine_malloc(C.int(size))) }
func (e Engine) MeshCreate(...) int32 { return int32(C.engine_mesh_create(...)) }
// ... 20+ methods duplicated

// engine_api_adapter_wasm.go — another file, same methods, different impl
func NewEngine() Engine { return Engine{handle: _engine_init()} }
func (e Engine) Malloc(size int32) Ptr { return Ptr(_engine_malloc(size)) }
func (e Engine) MeshCreate(...) int32 { return _engine_mesh_create(...) }
```

Problems:
- Every new engine function requires edits in both files
- Method signatures must stay in sync manually
- Can't mock for testing
- Engine struct methods mix handle state with platform dispatch

### Target (Host struct)

```go
// host.go — shared, platform-agnostic
var Host EngineHost

type EngineHost struct {
    Init    func() int32
    Cleanup func(e int32) int32

    Memory struct {
        Malloc   func(size int32) Ptr
        Free     func(p Ptr)
        BulkCopy func(dst Ptr, src unsafe.Pointer, length int32)
    }

    Mesh struct {
        Create  func(e int32, verts Ptr, vertBytes int32, indices Ptr, idxBytes int32) int32
        Destroy func(m int32)
    }

    Shader struct {
        Create func(e int32, desc Ptr, descLen int32) int32
    }

    Pipeline struct {
        Create func(e int32, desc Ptr, descLen int32) int32
    }

    Image struct {
        CreateTarget func(e int32, w, h, pixelFormat int32) int32
    }

    Sampler struct {
        Create func(e int32, minFilter, magFilter, wrap, compare int32) int32
    }

    Pass struct {
        Create       func(e int32, color, depth int32) int32
        Begin        func(e int32, p int32, clearDepth float32)
        BeginDefault func(e, int32, r, g, b, a, depth float32)
        End          func(e int32)
    }

    Draw struct {
        ApplyPipeline func(e int32, p int32)
        BindMesh      func(e int32, m int32)
        BindImage     func(e int32, slot int32, img int32, smp int32)
        ApplyUniforms func(e int32, slot int32, data Ptr, len int32)
        DrawElements  func(e int32, base, count, instances int32)
    }

    Commit func(e int32)
}
```

### Platform init

```go
// host_cgo.go (//go:build !js && !wasip1)
func init() {
    Host.Init = func() int32 { return int32(C.engine_init()) }
    Host.Memory.Malloc = func(size int32) Ptr {
        return Ptr(uintptr(C.engine_malloc(C.int(size))))
    }
    Host.Memory.Free = func(p Ptr) { C.engine_free(unsafe.Pointer(p)) }
    Host.Memory.BulkCopy = func(dst Ptr, src unsafe.Pointer, length int32) {
        C.memcpy(unsafe.Pointer(dst), src, C.size_t(length))
    }
    // ... all other fields
}

// host_wasm.go (//go:build js || wasip1)
//go:wasmimport env engine_init
func _engine_init() int32
//go:wasmimport env engine_malloc
func _engine_malloc(size int32) uint32
// ... all wasmimport declarations

func init() {
    Host.Init = _engine_init
    Host.Memory.Malloc = func(size int32) Ptr { return Ptr(_engine_malloc(size)) }
    Host.Memory.Free = func(p Ptr) { _engine_free(uint32(p)) }
    Host.Memory.BulkCopy = func(dst Ptr, src unsafe.Pointer, length int32) {
        _bulk_copy(uint32(dst), uint32(uintptr(src)), length)
    }
    // ... all other fields
}
```

### Engine struct becomes thin wrapper

```go
// engine.go — shared
type Engine struct {
    handle int32
}

func NewEngine() Engine {
    return Engine{handle: Host.Init()}
}

func (e Engine) Malloc(size int32) Ptr        { return Host.Memory.Malloc(size) }
func (e Engine) Free(p Ptr)                   { Host.Memory.Free(p) }
func (e Engine) BulkCopy(dst Ptr, src unsafe.Pointer, length int32) {
    Host.Memory.BulkCopy(dst, src, length)
}
func (e Engine) MeshCreate(verts Ptr, vb int32, indices Ptr, ib int32) int32 {
    return Host.Mesh.Create(e.handle, verts, vb, indices, ib)
}
// ... all methods delegate to Host
```

### Benefits

- Single source of truth for API surface (`host.go`)
- Platform files only assign function pointers — no signature duplication
- Engine methods are trivial delegates — no build-tag split needed
- Testable: `Host.Memory.Malloc = func(size int32) Ptr { return mockAlloc(size) }`
- Adding new engine function: add field to Host + assign in both init() funcs

### Migration Steps

1. Create `host.go` with `EngineHost` struct and `Engine` wrapper methods
2. Create `host_cgo.go` — move CGO imports + init assignments from `engine_api_adapter_cgo.go`
3. Create `host_wasm.go` — move wasmimport decls + init assignments from `engine_api_adapter_wasm.go`
4. Delete `engine_api_adapter_{cgo,wasm}.go`
5. Verify native build + WASM build still work

### What Stays the Same

- `game.go` — no changes (calls `Engine` methods, which now delegate to Host)
- `game_api_impl_{cgo,wasm}.go` — stays split (go:wasmexport vs cgo //export are fundamentally different)
- `ptr_{native,wasm}.go` — stays (Ptr type alias)
- `gfx.go`, `shader_phong.go` — no changes
- All C/C++ code — no changes
- `triggle.html` — no changes

---

## Future Work (after Host struct)

1. **Triggle game logic** — hex board, pegs, rubber bands, triangle detection
2. **Mouse/touch input** — extend event system (flattened args for both platforms)
3. **Camera control** — orbit/pan/zoom
4. **Drag and drop** — ray picking against board geometry
5. **Dynamic meshes** — rubber bands as dynamic vertex buffers
6. **Text/UI** — score display, turn indicator
7. **Reverse bulk_copy** — engine→game for viewport queries, picking results
