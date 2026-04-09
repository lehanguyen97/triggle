# Plan: Go ↔ host logging and structured errors (graphics.gd–style)

**Goal:** Let Go report **human-readable text** to the native/C++ host (stderr, debugger, or UI) **without** the Go `log` package or `fmt.Errorf` on hot paths where WASM size matters. Mirror the **pointer + length** string passing used by graphics.gd’s `gd_log_error` / `Host.Log.Error` wiring.

**References**

- graphics.gd CGO: `startup/startup_cgo_v2.go` — `unsafe.StringData(s)` + `len(s)` into `C.gd_log_error` / `gd_log_warning`.
- graphics.gd semantics: `classdb/Engine/extra.go` — `Raise` → `Host.Log.Error` with message + function + file + line.
- Triggle today: `game_frame` returns **0** on success, **non-zero** on error (opaque); `main.cpp` logs the int and quits; init failures return `game_init != 0` with no text to host.

## Current gaps

| Concern | Today |
|--------|--------|
| Init failure | `game_init` → `-1`, no string |
| Frame failure | non-zero `game_frame` return, generic stderr line in C++ |
| Go errors | `errors.New("static")` only; no bridge for text to C++ |
| WASM size | Avoid `log`; avoid `fmt` where possible (see prior wasm build comparison) |

## Design principles

1. **Length-delimited UTF-8** — Pass `(ptr, len)` per string (or one packed blob). Do **not** require NUL-terminated Go strings crossing the boundary.
2. **Host owns I/O** — C++/JS prints or forwards; Go does not call `fprintf` directly in the game module (optional tiny `//export` helpers only if needed for WASI).
3. **Layered severity** — Match graphics.gd roughly: **error** vs **warning** vs optional **verbose** (gated by env or build flag later).
4. **No heavy formatting in Go** for the default path — static messages or pre-built `[]byte`/string from small helpers; reserve `fmt` for editor/debug builds if ever needed.

## Proposed C ABI (game host, not GPU backend)

Extend **`game_api.h`** (or a small `game_log.h` included from it) with optional callbacks the **host implements** and the **game calls**:

```c
/* Optional: host registers these once after game_init, or weak symbols / no-op defaults. */
void game_host_log_error(const char *msg, int32_t msg_len,
                         const char *func_name, int32_t func_len,
                         const char *file, int32_t file_len,
                         int32_t line);
void game_host_log_warning(const char *msg, int32_t msg_len, ...);
```

- Empty `func`/`file`/`line` allowed (send `NULL, 0` or `""`, `0`) for simple messages.
- **CGO:** Go implementation calls C with `(*C.char)(unsafe.Pointer(unsafe.StringData(s)))` and `C.int32_t(len(s))` for non-empty strings; for empty use `NULL, 0`.

**WASM (wasip1 + JS glue):** Same contract via `wasmimport`:

- `go:wasmimport` `game_host_log_error(msg_ptr, msg_len, ...)` implemented in JS or in `triggle.html` / Emscripten preamble: read UTF-8 from **game** linear memory and `console.error` / append DOM log.

**Native `main.cpp`:** Provide `game_host_log_error` that `fwrite`/`fprintf` to stderr with a `triggle:` prefix, optionally include file/line when non-empty.

## Go side (`game/` or thin `game/log.go`)

- **`LogError(msg string)`** / **`LogErrorAt(msg, funcName, file string, line int)`** — call host; `funcName`/`file`/`line` from `runtime.Caller` when desired (optional; costs a bit—use only on slow paths).
- **`LogWarning`** — same pattern.
- **`LogThenFrameFail(msg string)`** (optional) — log to host then return non-zero from `update` so C++ can still print the numeric code if desired.
- Keep **`errors.New`** for in-Go propagation; call **`LogError`** once at the boundary where you turn failure into a frame/init result.

## Integration with `game_frame` return value

- Any **non-zero** return means failure; host treats it as opaque (today Go uses `-1` in most paths).
- **`LogError` immediately before `return -1`** in `rebuild*` / border upload failures if host logging exists; C++ still quits when `fr != 0`.

## `game_init` failure text

- Option A: **`game_init` returns `-1`** and host calls **`game_get_last_error(ptr_out, len_out)`** that copies a NUL-terminated or `(ptr,len)` from a static Go buffer filled by `newGame` on failure.
- Option B: **`game_host_log_error` invoked from Go** during failed `newGame` before return `-1` (graphics.gd-style: log then signal failure).

Prefer **B** for one mechanism only; ensure host no-ops or prints on all platforms.

## Work items (ordered)

1. **Declarations** — Add `game_host_log_*` to `game_api.h`; stub no-op implementations in `backend/` (or link-time weak defaults).
2. **CGO** — `game/log_host_cgo.go`: call C shims with `StringData` + length.
3. **WASM** — `game/log_host_wasm.go`: `go:wasmimport` + document loader stub in `triggle.html` / CMake copied assets.
4. **`main.cpp`** — Implement real stderr logging; optionally **`sapp_quit`** only after log (already partly there for frame errors).
5. **Wire game** — `newGame` abort path + mesh rebuild failures: `LogError` + keep `errors.New` for return values where applicable.
6. **Docs / CLAUDE** — Link this file; note “no `log`/`fmt` on hot path for wasm size.”

## Non-goals (for this plan)

- Full `BACKEND_ERR_*` strings from C++ into Go (separate track in `go-runtime-api-plan.md`).
- Rich stack trace capture across WASM (possible later via `runtime.Stack` into host buffer; size/latency cost).

## Open questions

- **Rate limit** — spam from per-frame failures: gate logs or log once per fatal path.
- **UTF-8 in MSVC console** — Windows may need wide-char or UTF-8 console mode; document.
- **Threading** — Today game runs on main thread; if that changes, logging must stay serial or locked on host.
