# Wasm2Wasm Design

## Goal
Replace `syscall/js` with `go:wasmimport`/`go:wasmexport`. Keep identical Go game code for native (cgo) and browser (wasm) builds. Engine is a thin C/C++ wrapper around external libs (sokol, cglm, cgltf). Go owns all game logic.

## Current State (branch: wasm-go)

### Architecture
```
Engine (C++/Sokol) ──Emscripten──> engine WASM (.js + .wasm)
Game   (Go)        ──GOOS=js────> game.wasm + wasm_exec.js
Communication: syscall/js → js.Global().Call("_engine_init") → Emscripten exports
```

### Current Files

**Engine C API** (`engine/include/e/engine_api.h`):
```c
engine_t engine_init();
int32_t engine_register_mesh(engine_t e, MeshData data);  // MeshData has float*, uint16_t*
int32_t engine_render(engine_t e, RenderArg arg);          // RenderArg has uintptr_t shader_params
int32_t engine_cleanup(engine_t e);
```
Problem: passes structs with pointers — can't cross WASM module boundary.

**Engine impl** (`engine/src/`):
- `engine.hpp` — Engine struct: sokol pipeline, vector<sg_bindings*>, pass_action
- `engine.cpp` — init (sokol setup, shader, pipeline), register_mesh (create GPU buffers), render (begin_pass → apply_pipeline → apply_bindings → apply_uniforms → draw → end_pass → commit), cleanup
- `engine_api_impl.cpp` — static Engine* singleton, C wrappers with EMSCRIPTEN_KEEPALIVE
- `main.cpp` — sokol_app lifecycle, calls game_init/frame/event/cleanup, maps sapp events to GEvent
- `shader.glsl` — simple MVP * position vertex shader, passthrough color fragment shader

**Game API** (`engine/include/e/game_api.h`):
```c
game_t game_init();
int32_t game_frame(game_t game, double dt);
int32_t game_event(game_t game, GEvent event);
int32_t game_cleanup(game_t game);
```

**Go game** (`game_go/`):
- `game.go` — Game struct (engine, transform, viewProj, mvp, vertices, indices, cubeBindID). newGame() creates colored cube (24 verts, 36 indices), sets perspective+view. update() rotates cube, computes MVP, calls engine.Render().
- `game_api_impl.go` — `//export game_init/frame/event/cleanup` via cgo. Global `var game *Game`.
- `engine_api_adapter_cgo.go` — `//go:build !js && !wasm`. Engine is int32. Uses C.engine_init(), C.CBytes() for mesh data, casts Go structs to C structs via unsafe.Pointer.
- `engine_api_adapter_wasm.go` — `//go:build js || wasm`. Uses syscall/js. js.Global().Call() for engine functions, js.CopyBytesToJS for mesh data via Module.HEAPU8.

**Build** (`CMakeLists.txt` files):
- Root: includes game_go + engine subdirs
- engine/: builds triggle executable. Emscripten: links as .js, exports _main,_malloc, HEAPU8. Native: links game_go static lib.
- game_go/: Emscripten: `GOOS=js GOARCH=wasm go build -o game.wasm`, copies wasm_exec.js. Native: `go build -buildmode=c-archive` → libgamego.a

**HTML** (`engine/triggle.html`): loads wasm_exec.js, has canvas, empty script tag.

### Key Issues with Current Approach
1. All engine calls go through JS bridge (slow)
2. Requires wasm_exec.js Go runtime
3. syscall/js is awkward — manual HEAPU8 manipulation for memory
4. Different code paths for wasm vs cgo (Engine is js.Value vs int32)

---

## Target Architecture

```
Engine (C++/Sokol) ──Emscripten──> main WASM module
  │  exports: engine API functions + malloc/free
  │  JS glue: loads game.wasm, provides bulk_copy, WASI polyfills
  │  calls: game_init/frame/event/cleanup (game's wasmexports)
  │
Game (Go) ──GOOS=wasip1 -buildmode=c-shared──> game.wasm (reactor)
     imports (go:wasmimport): engine API + bulk_copy
     exports (go:wasmexport): game_init/frame/event/cleanup
     no wasm_exec.js, no syscall/js
```

### How Memory Works

**Problem:** Two WASM modules have separate linear memories.

**Solution (from graphics.gd):**
1. Engine exports `malloc`/`free` (Emscripten already does this via `_malloc`/`_free`)
2. JS glue provides `bulk_copy(dst_in_engine, src_in_game, len)` — copies between the two WebAssembly.Memory buffers
3. Go game: allocates in engine memory via engine_malloc, bulk_copies Go data there, passes engine-memory pointers to engine API
4. For native/cgo: `bulk_copy` = `memcpy`, `engine_malloc` = `malloc`, pointers are real — same code, no branching

---

## New Engine C API

```c
// engine_api.h
#pragma once
#ifdef __cplusplus
extern "C" {
#endif

#include <stdint.h>
#include <stddef.h>

typedef int32_t engine_t;
typedef int32_t mesh_t;

// Lifecycle
engine_t engine_init(void);
int32_t  engine_cleanup(engine_t e);

// Memory management (engine heap)
// On wasm: allocates in engine's linear memory
// On native: just malloc/free
void*    engine_malloc(int32_t size);
void     engine_free(void* ptr);

// Mesh
// All pointers must be in engine memory (allocated via engine_malloc + bulk_copy)
// vertices: float array [x,y,z,r,g,b,a, ...], vert_bytes = num_floats * 4
// indices: uint16 array, idx_bytes = num_indices * 2
mesh_t   engine_mesh_create(engine_t e,
             void* vertices, int32_t vert_bytes,
             void* indices, int32_t idx_bytes);
void     engine_mesh_update(mesh_t m,
             void* vertices, int32_t vert_bytes,
             void* indices, int32_t idx_bytes);
void     engine_mesh_destroy(mesh_t m);

// Render
// mvp: pointer to 16 floats (4x4 matrix) in engine memory
void     engine_frame_begin(engine_t e);
void     engine_draw(engine_t e, mesh_t m, void* mvp);
void     engine_frame_end(engine_t e);

#ifdef __cplusplus
}
#endif
```

### Changes from Current API
- `engine_register_mesh(e, MeshData)` → `engine_mesh_create(e, verts, vbytes, indices, ibytes)` — flat args instead of struct with pointers
- `engine_render(e, RenderArg)` → split into `engine_frame_begin` / `engine_draw` / `engine_frame_end` — engine owns render pass lifecycle
- Added `engine_malloc`/`engine_free` — game allocates in engine heap for cross-module data
- Added `engine_mesh_update` — for dynamic meshes (rubber bands in triggle)
- Added `engine_mesh_destroy` — cleanup individual meshes
- `engine_draw` takes `mesh_t` handle + `void* mvp` (pointer to 16 floats in engine memory)

### Game API (unchanged)
```c
// game_api.h — no changes needed
game_t game_init();
int32_t game_frame(game_t game, double dt);
int32_t game_event(game_t game, GEvent event);
int32_t game_cleanup(game_t game);
```

---

## Go Adapter Layer

### Pointer Type (build-tag split)

```go
// ptr_native.go (//go:build !js && !wasip1)
package main
type Ptr = uintptr  // real C pointer

// ptr_wasm.go (//go:build js || wasip1)
package main
type Ptr = uint32   // offset in engine's linear memory
```

### Engine Interface (shared by both platforms)

```go
// engine.go
type Engine struct {
    handle int32
}

// Methods defined in adapter files, signature uses Ptr:
// func NewEngine() Engine
// func (e Engine) Malloc(size int32) Ptr
// func (e Engine) Free(p Ptr)
// func (e Engine) BulkCopy(dst Ptr, src unsafe.Pointer, len int32)
// func (e Engine) MeshCreate(verts Ptr, vertBytes int32, indices Ptr, idxBytes int32) int32
// func (e Engine) MeshUpdate(mesh int32, verts Ptr, vertBytes int32, indices Ptr, idxBytes int32)
// func (e Engine) MeshDestroy(mesh int32)
// func (e Engine) FrameBegin()
// func (e Engine) Draw(mesh int32, mvp Ptr)
// func (e Engine) FrameEnd()
// func (e Engine) Cleanup() int32
```

### CGO Adapter (`engine_api_adapter_cgo.go`)

```go
//go:build !js && !wasip1

package main

/*
#cgo CFLAGS: -I../engine/include
#include <e/engine_api.h>
#include <string.h>
*/
import "C"
import "unsafe"

func NewEngine() Engine {
    return Engine{handle: int32(C.engine_init())}
}

func (e Engine) Malloc(size int32) Ptr {
    return Ptr(uintptr(C.engine_malloc(C.int(size))))
}

func (e Engine) Free(p Ptr) {
    C.engine_free(unsafe.Pointer(p))
}

func (e Engine) BulkCopy(dst Ptr, src unsafe.Pointer, length int32) {
    C.memcpy(unsafe.Pointer(dst), src, C.size_t(length))
}

func (e Engine) MeshCreate(verts Ptr, vertBytes int32, indices Ptr, idxBytes int32) int32 {
    return int32(C.engine_mesh_create(C.engine_t(e.handle),
        unsafe.Pointer(verts), C.int(vertBytes),
        unsafe.Pointer(indices), C.int(idxBytes)))
}

func (e Engine) Draw(mesh int32, mvp Ptr) {
    C.engine_draw(C.engine_t(e.handle), C.mesh_t(mesh), unsafe.Pointer(mvp))
}

func (e Engine) FrameBegin() {
    C.engine_frame_begin(C.engine_t(e.handle))
}

func (e Engine) FrameEnd() {
    C.engine_frame_end(C.engine_t(e.handle))
}

// ... etc
```

### WASM Adapter (`engine_api_adapter_wasm.go`)

```go
//go:build js || wasip1

package main

import "unsafe"

// Engine API imports (provided by engine WASM exports)
//go:wasmimport env engine_init
func _engine_init() int32

//go:wasmimport env engine_cleanup
func _engine_cleanup(e int32) int32

//go:wasmimport env engine_malloc
func _engine_malloc(size int32) uint32

//go:wasmimport env engine_free
func _engine_free(ptr uint32)

//go:wasmimport env engine_mesh_create
func _engine_mesh_create(e int32, verts uint32, vertBytes int32, indices uint32, idxBytes int32) int32

//go:wasmimport env engine_mesh_update
func _engine_mesh_update(m int32, verts uint32, vertBytes int32, indices uint32, idxBytes int32)

//go:wasmimport env engine_mesh_destroy
func _engine_mesh_destroy(m int32)

//go:wasmimport env engine_frame_begin
func _engine_frame_begin(e int32)

//go:wasmimport env engine_draw
func _engine_draw(e int32, m int32, mvp uint32)

//go:wasmimport env engine_frame_end
func _engine_frame_end(e int32)

// JS-bridged: copies from game WASM memory → engine WASM memory
//go:wasmimport env bulk_copy
func _bulk_copy(dst_engine uint32, src_game uint32, length int32)

func NewEngine() Engine {
    return Engine{handle: _engine_init()}
}

func (e Engine) Malloc(size int32) Ptr {
    return Ptr(_engine_malloc(size))
}

func (e Engine) Free(p Ptr) {
    _engine_free(uint32(p))
}

func (e Engine) BulkCopy(dst Ptr, src unsafe.Pointer, length int32) {
    _bulk_copy(uint32(dst), uint32(uintptr(src)), length)
}

func (e Engine) MeshCreate(verts Ptr, vertBytes int32, indices Ptr, idxBytes int32) int32 {
    return _engine_mesh_create(e.handle, uint32(verts), vertBytes, uint32(indices), idxBytes)
}

func (e Engine) Draw(mesh int32, mvp Ptr) {
    _engine_draw(e.handle, int32(mesh), uint32(mvp))
}

func (e Engine) FrameBegin() { _engine_frame_begin(e.handle) }
func (e Engine) FrameEnd()   { _engine_frame_end(e.handle) }

// ... etc
```

### Game API Exports (`game_api_impl.go`)

Two versions needed:

```go
// game_api_impl_cgo.go (//go:build !js && !wasip1)
// Same as current — uses cgo //export

// game_api_impl_wasm.go (//go:build js || wasip1)
//go:wasmexport game_init
func game_init() int32 { ... }

//go:wasmexport game_frame
func game_frame(g int32, dt float64) int32 { ... }

//go:wasmexport game_event
func game_event(g int32, evType int32, evData int32) int32 { ... }

//go:wasmexport game_cleanup
func game_cleanup(g int32) int32 { ... }
```

Note: `go:wasmexport` can't take C types or structs. game_event signature must use flat scalars (evType + evData) instead of GEvent union on wasm. On cgo side, keep the current GEvent struct.

### Game Code (`game.go`)

```go
// Game code is platform-agnostic. Uses Engine methods + Ptr type.
func (g *Game) registerMesh() int32 {
    vertBytes := unsafe.Slice((*byte)(unsafe.Pointer(&g.vertices[0])), len(g.vertices)*4)
    idxBytes := unsafe.Slice((*byte)(unsafe.Pointer(&g.indices[0])), len(g.indices)*2)

    vPtr := g.engine.Malloc(int32(len(vertBytes)))
    g.engine.BulkCopy(vPtr, unsafe.Pointer(&vertBytes[0]), int32(len(vertBytes)))

    iPtr := g.engine.Malloc(int32(len(idxBytes)))
    g.engine.BulkCopy(iPtr, unsafe.Pointer(&idxBytes[0]), int32(len(idxBytes)))

    g.cubeBindID = g.engine.MeshCreate(vPtr, int32(len(vertBytes)), iPtr, int32(len(idxBytes)))
    g.engine.Free(vPtr)  // engine copied to GPU, safe to free
    g.engine.Free(iPtr)
    return 0
}

func (g *Game) update(dt float32) int32 {
    // ... compute MVP matrix ...

    // Pre-allocate mvpBuf once, reuse per frame
    g.engine.BulkCopy(g.mvpPtr, unsafe.Pointer(&g.mvp[0]), 64)
    g.engine.FrameBegin()
    g.engine.Draw(g.cubeBindID, g.mvpPtr)
    g.engine.FrameEnd()
    return 0
}
```

---

## Engine Implementation Changes

### engine_api_impl.cpp — New Functions

```cpp
// malloc/free wrappers
EXPORT void* engine_malloc(int32_t size) {
    return malloc(size);
}

EXPORT void engine_free(void* ptr) {
    free(ptr);
}

// Flat-argument mesh creation (no struct passing)
EXPORT mesh_t engine_mesh_create(engine_t et, void* vertices, int32_t vert_bytes,
                                  void* indices, int32_t idx_bytes) {
    if (!e || et != 0) return -1;
    MeshData data;
    data.vertices = (float*)vertices;
    data.nv = vert_bytes / sizeof(float);
    data.indices = (uint16_t*)indices;
    data.ni = idx_bytes / sizeof(uint16_t);
    return e->register_mesh(data);
}

EXPORT void engine_frame_begin(engine_t et) {
    if (!e || et != 0) return;
    e->begin_frame();
}

EXPORT void engine_draw(engine_t et, mesh_t m, void* mvp) {
    if (!e || et != 0) return;
    e->draw(m, (mat4s*)mvp);
}

EXPORT void engine_frame_end(engine_t et) {
    if (!e || et != 0) return;
    e->end_frame();
}
```

### engine.cpp — Split render into begin_frame/draw/end_frame

Current `render()` does everything: begin_pass → pipeline → bindings → uniforms → draw → end_pass → commit.

Split into:
- `begin_frame()`: sg_begin_pass (clear + swapchain setup), sg_apply_pipeline
- `draw(mesh, mvp)`: sg_apply_bindings, sg_apply_uniforms, sg_draw
- `end_frame()`: sg_end_pass, sg_commit

This lets game issue multiple draw calls per frame (needed for hex board + pegs + rubber bands).

### main.cpp — Load Game WASM Module

On Emscripten, main.cpp currently calls `game_init()` etc directly (linked at compile time via cgo archive or expecting the symbols from game.wasm loaded by wasm_exec.js).

For wasm2wasm, engine's JS glue must:
1. After Emscripten module is ready, fetch `game.wasm`
2. Instantiate it with imports: engine exports as `env`, WASI polyfills as `wasi_snapshot_preview1`, `bulk_copy` as `env`
3. Call game's `_initialize` (wasip1 reactor init)
4. Wire game exports (game_init/frame/event/cleanup) as function pointers callable from C

This requires a JS-side loader. Engine's `main.cpp` can use `EM_JS` or `EM_ASM` to define the JS bridge, or it can be in the HTML/JS loader script.

### Emscripten Link Flags

Current:
```
-sEXPORTED_FUNCTIONS=_main,_malloc
-sEXPORTED_RUNTIME_METHODS=HEAPU8
```

New:
```
-sEXPORTED_FUNCTIONS=_main,_malloc,_free,_engine_init,_engine_cleanup,_engine_malloc,_engine_free,_engine_mesh_create,_engine_mesh_update,_engine_mesh_destroy,_engine_frame_begin,_engine_draw,_engine_frame_end
-sEXPORTED_RUNTIME_METHODS=wasmMemory
```

Drop HEAPU8 (no longer needed). Export wasmMemory for bulk_copy JS bridge.

### JS Loader (in triggle.html or separate .js)

```js
// After Emscripten module is ready:
Module.onRuntimeInitialized = async function() {
    const engineMemory = Module.wasmMemory;

    // Fetch and compile game WASM
    const gameBytes = await fetch('game/game.wasm').then(r => r.arrayBuffer());
    let gameExports;

    const imports = {
        env: {
            engine_init:         Module._engine_init,
            engine_cleanup:      Module._engine_cleanup,
            engine_malloc:       Module._engine_malloc,
            engine_free:         Module._engine_free,
            engine_mesh_create:  Module._engine_mesh_create,
            engine_mesh_update:  Module._engine_mesh_update,
            engine_mesh_destroy: Module._engine_mesh_destroy,
            engine_frame_begin:  Module._engine_frame_begin,
            engine_draw:         Module._engine_draw,
            engine_frame_end:    Module._engine_frame_end,
            bulk_copy: (dst, src, len) => {
                new Uint8Array(engineMemory.buffer, dst, len)
                    .set(new Uint8Array(gameExports.memory.buffer, src, len));
            },
        },
        wasi_snapshot_preview1: {
            fd_write: (fd, iovs_ptr, iovs_len, nwritten_ptr) => {
                // Minimal: write to console
                return 0;
            },
            clock_time_get: (id, precision, time_ptr) => {
                const now = BigInt(Date.now()) * 1000000n;
                const view = new DataView(gameExports.memory.buffer);
                view.setBigInt64(time_ptr, now, true);
                return 0;
            },
            proc_exit: (code) => { throw new Error(`exit: ${code}`); },
            // ... other WASI stubs as needed
        },
    };

    const { instance } = await WebAssembly.instantiate(gameBytes, imports);
    gameExports = instance.exports;

    // Initialize Go runtime (wasip1 reactor)
    gameExports._initialize();

    // Now engine's main.cpp can call game functions through EM_JS bridges
    // Or: override game_init/frame/event/cleanup in Module to call gameExports
};
```

### Connecting Engine main.cpp to Game WASM Exports

Option A: EM_JS bridge in main.cpp — define game_init/frame/event/cleanup as EM_JS functions that call into JS globals set by the loader.

Option B: The loader script patches game function pointers after game.wasm loads, before sokol_main runs. This requires deferring sokol startup.

Option A is simpler:
```cpp
// main.cpp
#ifdef __EMSCRIPTEN__
EM_JS(int32_t, game_init, (), {
    return Module._gameExports.game_init();
});
EM_JS(int32_t, game_frame, (int32_t g, double dt), {
    return Module._gameExports.game_frame(g, dt);
});
// ... etc
#endif
```

---

## Build Changes

### game_go/CMakeLists.txt (WASM path)

Current:
```cmake
COMMAND GOOS=js GOARCH=wasm ${GO_EXEC} build -o game.wasm
```

New:
```cmake
COMMAND GOOS=wasip1 GOARCH=wasm ${GO_EXEC} build -buildmode=c-shared -o game.wasm
```

- Drop wasm_exec.js copy step
- `-buildmode=c-shared` produces a reactor (exports `_initialize`, no `_start`)

### engine/CMakeLists.txt

Update EXPORTED_FUNCTIONS to include all engine API functions.
Drop HEAPU8 from EXPORTED_RUNTIME_METHODS, add wasmMemory.

---

## File Change Checklist

### Engine C/C++
- [ ] `engine/include/e/engine_api.h` — new API (mesh_create, frame_begin/draw/end, malloc/free)
- [ ] `engine/include/e/game_api.h` — flatten game_event args for wasm compat (or keep as-is for cgo, split for wasm)
- [ ] `engine/src/engine.hpp` — add begin_frame/draw/end_frame methods
- [ ] `engine/src/engine.cpp` — split render() into begin_frame/draw/end_frame, implement mesh_update/destroy
- [ ] `engine/src/engine_api_impl.cpp` — add new C wrappers, engine_malloc/free
- [ ] `engine/src/main.cpp` — add EM_JS bridges for game functions (wasm path)
- [ ] `engine/CMakeLists.txt` — update Emscripten link flags

### Go Game
- [ ] `game_go/ptr_native.go` — new file: `type Ptr = uintptr`
- [ ] `game_go/ptr_wasm.go` — new file: `type Ptr = uint32`
- [ ] `game_go/engine_api_adapter_wasm.go` — rewrite: go:wasmimport + bulk_copy
- [ ] `game_go/engine_api_adapter_cgo.go` — update for new API signatures
- [ ] `game_go/game_api_impl.go` — split into _cgo.go and _wasm.go variants
- [ ] `game_go/game.go` — update for new Engine methods (Malloc+BulkCopy+MeshCreate pattern, FrameBegin/Draw/FrameEnd)
- [ ] `game_go/CMakeLists.txt` — GOOS=wasip1, -buildmode=c-shared, drop wasm_exec.js

### HTML/JS
- [ ] `engine/triggle.html` — new JS loader (instantiate game.wasm, provide bulk_copy + WASI polyfills)

---

## Unresolved Questions

1. **go:wasmexport + GEvent union** — wasm exports can't take C unions/structs. Need to flatten game_event signature for wasm. Two options: (a) split into game_event_keyboard(type, keycode, is_down, is_repeat), or (b) pass event as packed int32s. For cgo, keep the struct.

2. **Per-frame MVP allocation** — calling engine_malloc every frame is wasteful. Better: pre-allocate a persistent 64-byte slot in engine memory at init, reuse via bulk_copy each frame.

3. **Go 1.25 wasip1 + -buildmode=c-shared** — needs verification. If unsupported, fallback: GOOS=js with go:wasmimport (works since Go 1.21 on js target, but still needs some form of wasm_exec.js or polyfill for Go runtime).

4. **Deferred sokol startup** — sokol_main returns sapp_desc immediately. If game.wasm loading is async (fetch), need to either: (a) load game.wasm in JS before Emscripten module starts, or (b) defer game_init call until game.wasm is ready. Option (a) is cleaner — fetch game.wasm first, then start engine.

5. **Engine memory growth** — Emscripten may grow memory (realloc). After growth, JS TypedArray views are invalidated. bulk_copy must re-derive views each call (don't cache).
