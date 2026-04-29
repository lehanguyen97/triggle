# Native Host Swap — SDL3 (unscheduled)

Replace the bespoke callback-driven native windowing lib with SDL3 (or any lib exposing IME composition, clipboard, and explicit text-input control). WASM host is independent and untouched.

## Why
- Real in-canvas IME preedit for CJK / Japanese users.
- Native clipboard (absent today).
- Linux input-method support (ibus / fcitx).
- Headroom for controller input, drag-drop, multi-window, explicit vsync.

## What stays
- The C event ABI into Go — flattened scalars keyed by event type. New event kinds can be added; the call shape does not change.
- The GPU API. Sokol GFX is unchanged.
- The Go side — widgets, scene, UI, game — unchanged.

## What gets redrawn (native only)
- **Main-loop ownership.** Today the lib owns `main` and calls back. Target: we own `main`, the lib becomes a library, text-input and clipboard controls become synchronous calls from the UI layer instead of callbacks.
- **GPU surface provider.** The glue layer handing the GPU backend its per-frame environment + swapchain disappears with the lib. The host layer wires it directly — Metal view + next-drawable on macOS, GL context + default framebuffer on Linux. This is the only piece with real implementation weight; the rest is plumbing.
- **Text-input control.** Today there is no start/stop primitive — text events just arrive. Target exposes explicit start (passing a rect so the system IME anchors its candidate popup near the caret), stop, and a separate soft-keyboard toggle for touch surfaces.
- **Clipboard.** Simple get/set pair on the host. Ctrl-C / Ctrl-V in a focused text widget route through it.

## New event kinds
- Preedit-composition event for in-progress IME strings. Widget renders inline near caret.
- Commit events keep the existing text-event shape.

## Trade-offs
- Larger native binary and one more dev dependency.
- We own vsync and frame pacing instead of inheriting them.
- The macOS Metal glue is the only piece with real implementation risk.

## Independent follow-up — bidi
Native shapes individual runs today but does not run the Unicode bidi algorithm; mixed LTR/RTL renders in logical order. Fix path when needed: slot a bidi analyzer between the input string and the shaper to produce visual runs. No host-layer change.

## Open questions
- macOS surface via our own Metal glue, or drop to GL3 to halve the glue code (cost: lose Metal backend)?
- Ship clipboard with the swap, or as a separate beat?
- X11 only, or commit to Wayland at the same time?
