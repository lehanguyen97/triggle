# Asset loading — requirements, decisions, plan

## Layout

- **Game assets** (models, textures, packaged preload trees) live at the **repository root** under `assets/` (e.g. `assets/models/`). They are **not** under `nativebridge/`, `engine/`, or other code trees.
- **Emscripten preload** maps that folder into the VFS (e.g. `assets@/assets` → paths like `/assets/models/foo.glb`). CMake uses **`${CMAKE_SOURCE_DIR}/assets`** (top-level project root).

## Requirements

1. **Web (Emscripten / WASM)**  
   - Ship game data (e.g. models, textures) in a way that works with the **virtual filesystem** and normal C/C++ code paths (`fopen`, `std::ifstream`, engine loaders).  
   - **Startup must be understandable to users**: show a loading state while the browser downloads and unpacks packaged assets and while the engine parses and uploads to the GPU (same idea as native disk read + parse, but the download phase exists only on the web).

2. **Native**  
   - Continue to load from disk or bundled resources; no change to the requirement that **asset parsing lives in the C++ engine** (Go remains orchestration, not glTF/binary parsing).

3. **Future**  
   - Support **async HTTP fetch** of assets (large or optional content, CDN, updates) without blocking the main thread for the whole download.  
   - Not required for the first implementation.

## Decisions

| Topic | Decision |
|--------|----------|
| **Web, first version** | Use **Emscripten preload only** (`--preload-file` / CMake `SHELL: --preload-file ...`): assets are packaged into the `.data` blob, mounted on MEMFS, **no runtime `fetch()` per asset** in v1. |
| **Web, later** | Add **async fetch** (e.g. `emscripten_fetch`, or JS `fetch` + `FS.writeFile` / `load_from_memory`) for optional or large assets. |
| **Parsing** | **C++ engine** owns glTF and other binary formats; Go does not parse asset files. |
| **Loading UI** | **Preload still needs a loading state** on the web: the `.data` file must download; unpack + engine init still take time. Native can show the same phases minus “download” (or only “read disk + parse”). |

## GlTF and engine↔Go boundary (refines plan, does not replace it)

- **`cgltf`**: parse file → CPU buffers + scene graph; **no GPU work**. Engine uploads selected data, then may **`cgltf_free`** — no need to keep full file data in RAM after upload (only GPU + small engine tables). Peak RAM during load may still be large if one `.glb` packs everything; **split/cook assets in a pipeline** later if that becomes a problem (typical in shipped games).

- **Handles**: **one drawable handle per glTF mesh *primitive*** (material + geometry unit). Go keeps **opaque asset id** + whatever **indices/names** you need for gameplay; v1 can **hardcode** per known file; **sidecar metadata** (logical id → file + name or index) later.

- **Blender/tools**: **`name`** on nodes/meshes/materials usually exports; treat as **human lookup**, not a spec-guaranteed unique id — metadata DB can prefer **names** when stable, **indices** as fallback.

- **Go**: **orchestration only** (what to draw, transforms) — **no** glTF parsing.

## Implementation plan (preload-only web)

1. **Emscripten link flags**  
   - Add `--preload-file` (or `--embed-file` for tiny data) pointing at **repo-root** `assets/` so known VFS paths (e.g. `/assets/models/foo.glb`) exist before game code reads them.

2. **Engine**  
   - Implement loaders (e.g. glTF via `cgltf`) that read from **VFS paths** on WASM and from real paths on native; expose **primitive handles** and draw/uniform entry points for Go.  
   - Extend engine API as needed for **2D textures**, **mesh index types** (uint32), etc.

3. **HTML / shell**  
   - Ensure `triggle.html` (or the Emscripten shell) **does not assume “instant” startup** once `.data` grows: optional progress hooks or a minimal spinner until `Module` is ready and `main`/engine init completes (exact hook depends on Emscripten version and whether loading is async).

4. **Documentation**  
   - Keep build instructions in `README.md` updated: which assets are preloaded and where they appear in the VFS.

## Future work (async fetch)

- Add **one** code path: `load_from_memory(ptr, len)` or mount fetched bytes into FS, then call the same parser as preload.  
- **JS**: `fetch(url)` → copy into wasm heap → export; or **C**: `emscripten_fetch` with callback.  
- **UX**: separate “Downloading…” from “Loading…” when the asset is not in the preload bundle.

## References

- Emscripten: [filesystem](https://emscripten.org/docs/porting/files/index.html), [preload](https://emscripten.org/docs/porting/files/packaging_files.html)
- `ai/gltf-rendering-api.md` — **materials / textures / proposed C + Go API** (Blender parity path)
