# Go runtime API (game + engine in one WASM)

**Goal:** One Go module (`game` + `triggle/engine` runtime) talks to a separate C++/Sokol WASM that exposes a thin GPU bridge. Not in scope: merging the two WASM binaries; moving glTF parsing out of C++ yet.

## Layout (as in repo)

- `game/` — wasip1 `main`, rules, board; imports `triggle/engine`, submits draws.
- `engine/` — Go module only: `backend/`, `gfx/`, `shader/`, `render/`.
- `backend/` — C++/Emscripten + Sokol (`include/e/backend_api.h`, `src/api_impl.cpp`).

## 1) Bridge (`engine/backend`) — `GPU` / `Backend`

Handle-bound (`engine_t` inside `Backend`); methods do not pass an engine handle.

**Meshes / metadata**

- `MeshCreate` / `MeshDestroy` — GPU mesh lifecycle.
- `MeshIndexCount` / `MeshIndexType` — still exposed; **draw path avoids per-frame calls** when metadata is registered (below).
- `MeshInfo(mesh) (MeshInfo, bool)` — **one-shot** read: `IndexCount`, `IndexType`. WASM: backend writes 8 bytes to `out`; Go `Malloc` + `BulkCopyBack` into `struct`.

**glTF** — `GltfLoad`, `GltfUnload`, `GltfPrimitiveCount`, `GltfPrimitiveMesh` (parse/upload stays C++).

**Rest** — shaders, pipelines, images, samplers, passes, bind, uniforms, `DrawElements`, `Commit` (see `host.go` / `backend_api.h`).

## 2) Render runtime (`engine/render`)

**SceneDrawable** — per-submit intent: `Mesh`, `Model`, `MaterialID`, `Ambient`. No index fields on the struct; index count/type live in the renderer cache.

**Renderer** (`types.go`)

- `BeginFrame(cam, lights)`, `SubmitShadow(d)`, `SubmitMain(d)`, `EndFrame()` — shadow and main queues differ in order/culling.

**PhongRenderer** (concrete implementation)

- **`mesh map[int32]MeshInfo`** — filled at registration; draw uses this (not `MeshIndexCount` / `MeshIndexType` each pass).
- `RegisterMeshInfo(mesh, indexCount, indexType)` — after glTF primitive or any mesh whose metadata the game owns.
- `UploadMesh(verts, u16indices)` — `gfx.UploadMesh` + register `len(indices)` + `IndexUint16`.
- `DestroyMesh` — `MeshDestroy` + delete from map.
- `ForgetMesh` — delete from map only (e.g. after `GltfUnload`, which destroys meshes without going through `DestroyMesh`).
- `meshInfo(handle)` — map hit, else **lazy** one `gpu.MeshInfo` and cache (fallback if something forgot `RegisterMeshInfo`).

**Helpers**

- `LoadGltfPrimitive(gpu, path, prim) → asset, mesh, MeshInfo, ok` — validates via `MeshInfo`; caller should `RegisterMeshInfo` on the renderer.
- `UnloadGltfAsset(gpu, asset)` — then `ForgetMesh(mesh)` for any cached handle from that asset.

## 3) Pipeline families (`pipeline_cache.go`)

`RegisterPipelineFamily(desc) → id` registers u16 and u32 pipeline variants. **`Pipeline(family, indexType)`** picks the variant — no backend query; `indexType` comes from cached `MeshInfo`.

Game code does not branch on u16 vs u32 for glTF; it records backend-provided type at load/register time.

## glTF split

| Layer | Responsibility |
|-------|------------------|
| C++ | Binary glTF, buffers, unpack to Sokol mesh |
| Go | Asset/primitive handles, `LoadGltfPrimitive` / unload, materials later, scene-level lifecycle |

## Error model

Today: sentinel handles / `bool` ok paths. Future: compact `BACKEND_ERR_*` for create/load APIs; keep draw hot path unchanged.

**Game ↔ host messages:** See **`ai/go-host-logging-plan.md`** — pass UTF-8 `(ptr, len)` to C/JS for errors/warnings (graphics.gd pattern), keep WASM small (no `fmt`/`log` on hot paths); `game_frame` returns **0 = OK, non-zero = error** (opaque).

## Future (not implemented here)

`engine/core`, `audio/`, `input/`, `assets/` orchestration; optional rename of C++ tree to a name other than `backend/`—current docs use **`backend/`** for C++ and **`engine/backend`** for the Go bridge package.
