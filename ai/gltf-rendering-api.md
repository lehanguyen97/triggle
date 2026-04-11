# glTF rendering — API shape (current + proposed)

This doc continues the “Blender-like” / materials discussion: **what exists today**, **what a coherent next API could look like**, and **how Go uses it** without parsing glTF.

## Current API (implemented)

| Layer | Role |
|--------|------|
| **Load** | `engine_gltf_load(e, path)` → `asset` id. Unpacks geometry into **Phong-compatible** interleaved vertices (pos, normal, vertex color or white) + indices; uploads **one `mesh_t` per primitive**. |
| **Query** | `engine_gltf_primitive_count`, `engine_gltf_primitive_mesh`, `engine_mesh_index_count`. |
| **Unload** | `engine_gltf_unload(e, asset)` destroys uploaded meshes. |
| **Draw** | Submitted through renderer command buffer path (`ForwardRenderer` + `RenderProgram`). Shader is currently Phong/Toon style (not glTF PBR yet). |

**Go** mirrors this via `Engine.GltfLoad`, `GltfPrimitiveMesh`, `MeshIndexCount`, etc. Paths are **strings** copied into engine memory for the C loader.

**Not in glTF path yet:** textures, metallic/roughness, tangents, `doubleSided` (cull mode), separate materials per primitive beyond vertex color.

---

## Design principles for the next steps

1. **Engine** owns decoded glTF, GPU resources, and **material/pipeline selection** rules derived from glTF.  
2. **Go** chooses **what** to draw (asset id, primitive index, transform), not **how** accessors map to buffers.  
3. **Stable handles:** `mesh_t`, `material_t` (future), `texture_t` / `image_t` + `sampler_t` (once RGBA upload exists).  
4. **Batching:** optional later (`draw_gltf_primitive` that binds mesh + material + pipeline in one call); v1 can stay explicit like today.

---

## Proposed types (conceptual)

### `material_t` (engine handle)

Bundles **factors** + **texture slots** (or `-1` if unused) + **raster state** derived from glTF:

| glTF / Blender idea | Engine field (example) |
|----------------------|-------------------------|
| `pbrMetallicRoughness.baseColorFactor` | `vec4 base_color_factor` |
| `pbrMetallicRoughness.baseColorTexture` | `texture_handle` + texcoord set index |
| `normalTexture` | `texture_handle`, scale |
| `pbrMetallicRoughness.metallicFactor` / `roughnessFactor` | `float` each |
| `emissiveTexture` / `emissiveFactor` | optional |
| `doubleSided` | `bool` → maps to **pipeline** cull mode (`none` vs `back`) |
| `alphaMode` | opaque / mask / blend → pipeline blend state (later) |

The engine either:

- **A)** Stores a **small struct per material** and you bind uniforms + textures each draw, or  
- **B)** Bakes **static** props into the vertex buffer (only for fixed assets) — usually **not** what you want for Blender parity.

For Blender-like results, **A** + sampled textures is the target.

### `texture_t` / images

Requires **`engine_image_create_2d`** (or similar): RGBA8 blob → `sg_image` + view, plus existing **sampler** API. glTF images (PNG/JPEG) decode **in the engine** (stb_image, etc.) or via **preprocessed** KTX2 later.

### Pipelines (shaders)

| Variant | When |
|---------|------|
| **Phong + albedo texture** | UV + one `sampler2D` (closest to “tinted props”) |
| **glTF metallic-roughness** | Needs tangents, normal map, MR textures — second shader family |

You will have **multiple `pipeline_t`** values (same vertex layout if you extend vertices with UVs + optional tangent frame).

---

## Proposed C API additions (incremental)

**Phase 1 — textures + simple materials**

```c
/* After RGBA upload exists */
image_t engine_image_create_rgba8(engine_t e,
    int32_t w, int32_t h, const void* rgba, int32_t nbytes);

material_t engine_material_create_phong_textured(engine_t e,
    /* factors + which texture slots are bound; or binary blob descriptor */
    ...);

/* Per loaded glTF asset, after load */
int32_t engine_gltf_material_count(engine_t e, int32_t asset);
material_t engine_gltf_primitive_material(engine_t e, int32_t asset, int32_t prim);

int32_t engine_gltf_primitive_double_sided(engine_t e, int32_t asset, int32_t prim); /* 0/1 */
```

**Phase 2 — draw helper (optional)**

```c
void engine_draw_gltf_primitive(engine_t e,
    int32_t asset, int32_t prim,
    const float* model_mat4 /* 64 bytes */,
    pipeline_t pip /* or material resolves pip internally */);
```

Or keep draw orchestration in Go renderer and emit only command-buffer records to backend.

---

## How Go would use it (pattern)

```text
init:
  asset := engine.GltfLoad("/assets/models/prop.glb")
  prim0 := engine.GltfPrimitiveMesh(asset, 0)
  mat0  := engine.GltfPrimitiveMaterial(asset, 0)   // future
  pip   := choosePipeline(engine.GltfPrimitiveDoubleSided(asset, 0))  // main vs no-cull

per frame:
  rnd.SubmitMain(render.SceneDrawable{Mesh: prim0, Program: render.RenderProgramPhong, ...})
  rnd.EndFrame() // encodes + submits frame command buffer
```

**Metadata DB (later):** map `"player_token"` → `{ file, mesh_or_prim_index }`; Go only stores **logical id** and **transform**, not glTF JSON.

---

## How this maps to Blender

| Blender / glTF | Engine |
|------------------|--------|
| Object transform | Your **`model` matrix** (Go or engine node graph later). |
| Principled base color | `baseColorTexture` + factors → **Phong albedo** or full MR shader. |
| Normals / roughness | **Tangent space** + second shader stage; needs **UVs + tangents** in vertex buffer. |
| Viewport lighting | Your **directional + ambient + shadow** will not match HDRI; tune separately. |
| Units / axis | Fixed **export settings** + optional root transform in loader. |

---

## Summary

- **Today:** load → **mesh handles** + Phong vertex layout; draw like any other mesh.  
- **Next:** **`engine_image_create_*`**, **material handles** with factors + texture slots, **UVs** in vertices, **shader** that samples albedo (then MR + normals).  
- **Go:** stays **thin**: paths, handles, transforms, and which drawable to draw; **no** glTF parsing.

See also: `ai/asset-loading.md` (preload, layout), `CLAUDE.md` (host pattern).
