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

## C ABI (game host, not GPU backend) — **implemented**

**`game_api.h`** — one length-delimited UTF-8 string per call (full line text built in Go; host only prints/routes it):

```c
void backend_log_error(const char *msg, int32_t msg_len);
void backend_log_warning(const char *msg, int32_t msg_len);
```

- Empty message: `NULL, 0`.
- **CGO:** `(*C.char)(unsafe.Pointer(unsafe.StringData(s)))` and `len(s)` when `len > 0`.

**WASM:** `go:wasmimport env backend_log_error(msg_ptr, msg_len)` — `triggle.html` decodes UTF-8 from **game** memory → `console.error` / `console.warn`.

**Native:** `log.cpp` — `fwrite` the span + newline to stderr (no extra formatting; severity is still error vs warning export if we need routing later; today both write to stderr).

## Go side (`engine/hostlog/log_{cgo,wasm}.go`)

- **`hostlog.LogError(msg string)`** / **`hostlog.LogWarning(msg string)`** — pass `msg` through; build prefixes like `triggle: …` in callers if needed.
- **`LogThenFrameFail(msg string)`** (optional) — log to host then return non-zero from `update` so C++ can still print the numeric code if desired.
- Keep **`errors.New`** for in-Go propagation; call **`LogError`** once at the boundary where you turn failure into a frame/init result.

## Integration with `game_frame` return value

- Any **non-zero** return means failure; host treats it as opaque (today Go uses `-1` in most paths).
- **`LogError` immediately before `return -1`** in `rebuild*` / border upload failures if host logging exists; C++ still quits when `fr != 0`.

## `game_init` failure text

- Option A: **`game_init` returns `-1`** and host calls **`game_get_last_error(ptr_out, len_out)`** that copies a NUL-terminated or `(ptr,len)` from a static Go buffer filled by `newGame` on failure.
- Option B: **`backend_log_error` invoked from Go** during failed `newGame` before return `-1` (graphics.gd-style: log then signal failure).

Prefer **B** for one mechanism only; ensure host no-ops or prints on all platforms.

## Work items (ordered)

1. **Declarations** — Add `backend_log_*` to `game_api.h`; implementations in `backend/src/log.cpp`.
2. **CGO** — `engine/hostlog/log_cgo.go`: call C with `StringData` + length (`-I../../backend/include`).
3. **WASM** — `engine/hostlog/log_wasm.go`: `go:wasmimport` + `triggle.html` `env` stubs / CMake copied assets.
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
