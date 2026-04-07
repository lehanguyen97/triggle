# Changelog

## 2026-04-07: WASM HTML loader, engine temp strings, board edges

### WASM (`engine/triggle.html` + CMake)

- **`triggle.html` copied to build output** next to `triggle.js` (Emscripten `POST_BUILD`) so `http.server` from the build dir serves the right page.
- **Linker exports**: `EXPORTED_FUNCTIONS` lists only `_main,_malloc,_free`. All `engine_*` C API symbols stay exported via **`EMSCRIPTEN_KEEPALIVE`** in `engine_api_impl.cpp` (no need to duplicate names in CMake).
- **Game module imports**: `bulk_copy` plus **auto-forward** — iterate `Module` for keys `_engine_*` and attach as `engine_*` on the import object (avoids hand-maintaining a long list when the API changes).
- **WASI**: factored into a small `wasiPolyfill()` (fd_write, clock, random, stubs) for Go wasip1 reactor.

### Engine (`engine_api_impl.cpp`)

- **`TempStrings` uses `std::deque` instead of `std::vector`**: multiple `add()` calls feed `sg_shader_desc` with pointers from `c_str()`; vector reallocation could invalidate earlier pointers; deque appends do not invalidate existing element addresses.

### Removed files

- **`engine/src/shader.glsl`**, **`engine/src/shader.h`** — unused after Go-owned Phong/shadow shaders.

### Game (`game_go`)

- **`Board.CanPlace`**: placement allowed if the line adds **at least one new edge** (shared edges with prior bands OK); **`PlaceBand`** only inserts `Edges` for edges not already present (existing edge ownership unchanged).
- **Score peg UI**: pegs arranged in a **5-column grid** per player corner; corners with **score 0** draw nothing (no pegs).

---

## 2026-03-07: Shader API + Shadow Mapping

### New Engine API (thin sokol wrapper)

Replaced hardcoded single-shader engine with low-level graphics API. Engine has zero rendering logic — Go owns all passes, shaders, uniforms, draw calls.

**Binary descriptor pattern**: shader and pipeline creation use serialized binary blobs (built in Go, parsed in C) to avoid 30+ individual C functions. Only 14 new engine functions total.

#### New C functions
- `engine_shader_create(e, desc, len)` — parses binary blob → `sg_make_shader`
- `engine_pipeline_create(e, desc, len)` — parses binary blob → `sg_make_pipeline`
- `engine_image_create_target(e, w, h, fmt)` — creates render target image + attachment/texture views
- `engine_sampler_create(e, min, mag, wrap, cmp)` — creates sampler with optional comparison
- `engine_pass_create(e, color, depth)` — stores attachment view refs for offscreen pass
- `engine_pass_begin(e, pass, clear_depth)` — begin offscreen pass (depth-only for shadow)
- `engine_pass_begin_default(e, r,g,b,a, depth)` — begin swapchain pass
- `engine_pass_end`, `engine_commit`
- `engine_apply_pipeline`, `engine_bind_mesh`, `engine_bind_image`
- `engine_apply_uniforms(e, slot, data, len)` — raw bytes, sokol maps to glUniform calls
- `engine_draw_elements(e, base, count, instances)` — apply_bindings + sg_draw

#### Removed
- `engine_frame_begin`, `engine_draw`, `engine_frame_end` (old hardcoded render path)
- `shader.glsl` / `shader.h` (legacy sokol-shdc color demo; removed from tree — all shaders from Go)

### Sokol API Notes

Using newest sokol with `sg_view` objects:
- `sg_image` + `sg_make_view` → `sg_view` for attachments and texture sampling
- `sg_bindings.views[]` replaces old `images[]`
- `sg_shader_desc.views[].texture` replaces old `images[]`
- `sg_shader_desc.texture_sampler_pairs[]` replaces old `image_sampler_pairs[]`
- `sg_attachments` is now inline struct in `sg_pass` with `sg_view` members
- Image usage flags: `.usage.depth_stencil_attachment = true` replaces `.render_target`

### Go Shader System

**gfx.go** — descriptor builders + helpers:
- `ShaderDesc` / `PipelineDesc` structs → `BuildShaderDesc()` / `BuildPipelineDesc()` serialize to binary
- `CreateShader()` / `CreatePipeline()` — build + send to engine
- `UploadMesh()` — Malloc+BulkCopy+MeshCreate+Free helper
- Constants matching `engine_api.h` enums

**shader_phong.go** — two shaders as GLSL ES 300 strings:
- **Shadow shader**: MVP-only vertex shader, minimal fragment. Used for depth-only shadow pass.
- **Phong shadow shader**: Blinn-Phong lighting + shadow map sampling via `sampler2DShadow`.
  - VS uniforms (UB 0, 192B): model(mat4) + viewProj(mat4) + lightVP(mat4)
  - FS uniforms (UB 1, 36B): lightDir(vec3) + ambient(vec3) + cameraPos(vec3)
  - Uses `SG_UNIFORMLAYOUT_NATIVE` — each uniform declared individually in GLSL, sokol maps via `glGetUniformLocation`

### Demo Scene (game.go)

- Colored cube (6 face colors) rotating above a gray ground plane
- Directional light from upper-right with orthographic shadow projection
- Two-pass rendering:
  1. Shadow pass: depth-only to 1024x1024 texture, front-face culling to reduce acne
  2. Main pass: Phong + shadow map, comparison sampler

### File Changes

| File | Change |
|------|--------|
| `engine/include/e/engine_api.h` | New types, enums, 14 new function decls |
| `engine/src/engine.hpp` | Views vector, removed old render methods |
| `engine/src/engine.cpp` | Removed render/begin_frame/draw/end_frame, updated cleanup |
| `engine/src/engine_api_impl.cpp` | Binary descriptor parsers, all new API wrappers |
| `engine/src/main.cpp` | EM_JS bridges updated (no old draw API) |
| `engine/CMakeLists.txt` | Updated exported functions list |
| `engine/triggle.html` | Updated JS imports for new API |
| `game_go/gfx.go` | **New** — descriptor builders, constants, helpers |
| `game_go/shader_phong.go` | **New** — GLSL source + shader desc factories |
| `game_go/game.go` | Rewritten — two-pass shadow rendering, cube+plane scene |
| `game_go/engine_api_adapter_cgo.go` | Updated for new API |
| `game_go/engine_api_adapter_wasm.go` | Updated for new API |
