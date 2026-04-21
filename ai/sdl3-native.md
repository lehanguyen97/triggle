# Native Host Swap — SDL3

Plan for swapping the native windowing lib from the current callback-driven one to SDL3 (or any lib that exposes IME composition, clipboard, and explicit text-input control). WASM host is untouched.

## Why do this

- In-canvas IME preedit for CJK/Japanese users — current lib delivers committed codepoints only, no in-progress composition surface.
- Native clipboard copy/paste — absent today.
- Proper input-method support on Linux.
- Headroom for controller input, drag-drop, multi-window, explicit vsync control.

## What stays vs what gets redrawn

**Stays across the swap.** The C event ABI shape into Go — flattened scalars keyed by event type. New event *kinds* are additive; the call shape does not change. The GPU API is fully unchanged. The Go widget layer reads input snapshots and calls host primitives; it does not know which native lib is underneath.

**Gets redrawn, native-only.** Main-loop ownership, GPU surface provider, text-input trigger (implicit → explicit), clipboard (absent → host primitive).

## Main-loop ownership inversion

Today the lib owns `main` and calls us back per frame. Target: we own `main` and drive the loop; the lib becomes a library.

```c
// today — lib-owns-main
void on_init(void)   { game_init(); }
void on_frame(void)  { game_frame(lib_dt()); }
void on_event(ev*)   { translate(ev); }   // native ev → unified G_EVENT_*
/* entry macro returns a desc; lib takes over */

// target — we-own-main
int main(void) {
    host_init();                           // window + GPU surface
    game_init();
    while (!host_should_quit()) {
        while (host_poll_event(&ev)) translate(&ev);
        game_frame(dt_from_perf_counter());
        host_present();
    }
    game_cleanup();
    host_shutdown();
}
```

Same work, different owner. The inversion matters because it makes text-input and clipboard controls expressible as synchronous calls from the UI layer — things that are not naturally callbacks.

## GPU surface provider

Today the lib ships a glue header handing the GPU backend a ready-made environment + swapchain per frame. That glue vanishes with the swap. The host layer wires it directly instead:

```c
// conceptual — GPU backend consumes the same shapes, host produces them by hand
sg_environment host_gpu_env(void);        // device, pixel formats
sg_swapchain   host_gpu_swapchain(void);  // per-frame drawable / framebuffer
```

Platform reality behind these — create a Metal view on macOS and grab `nextDrawable` each frame; create a GL context on Linux and use the default framebuffer. This is the only piece with real implementation weight; the rest is plumbing. Callback-free reference implementations live inside the GPU backend's own sample tree.

## Event translation

Translator is a plain switch, same shape as today's, with a richer input vocabulary:

```c
switch (native_ev.kind) {
case NATIVE_KEY_DOWN:
    game_event(G_EVENT_KEY_DOWN, mapped_code, mods, ...); break;
case NATIVE_TEXT_COMMIT:                          // final composed codepoint
    game_event(G_EVENT_TEXT, codepoint, ...); break;
case NATIVE_TEXT_PREEDIT:                         // in-progress composition
    game_event(G_EVENT_TEXT_PREEDIT, str_ptr, str_len, caret_byte, ...); break;
/* ...mouse/resize/scroll unchanged... */
}
```

Two Go-facing event kinds added. `G_EVENT_TEXT` is stable across the swap — already defined in the no-preedit phase; the new lib just happens to be the source. `G_EVENT_TEXT_PREEDIT` is net-new; the widget renders the in-progress string inline near the caret. No other event kind changes.

## Explicit text-input control

Today there is no "start / stop receiving text events near this rect" primitive — text events just happen. Target exposes three primitives the UI context calls on focus transitions:

```c
void host_text_input_start(int rect_x, int rect_y, int rect_w, int rect_h);
void host_text_input_stop(void);
void host_show_soft_keyboard(bool on);
```

Reasons for three rather than one toggle. `text_input_start` passes a rect so the system IME anchors its candidate popup near the caret — the pattern Godot and editor-class apps use. `show_soft_keyboard` stays separate because mobile web / iPad has different semantics from desktop IME (the current lib already exposes this as a distinct call, keep the split).

UI context: `text_input_start` on focus gain, `text_input_stop` on focus loss, `show_soft_keyboard(true)` on touch targets.

## Clipboard

Simple pair across the C boundary:

```c
int32_t host_clipboard_get(char* out, int32_t out_cap); // bytes written, -1 on err
int32_t host_clipboard_set(const char* s, int32_t s_len);
```

Go calls these on Ctrl-C / Ctrl-V inside a focused text input on native. WASM routes through the DOM overlay and ignores the host pair.

## Trade-offs

- Larger native binary and one more dev dependency.
- We own vsync and frame pacing instead of inheriting them.
- One platform-specific glue layer for the GPU surface — the only piece with real implementation risk.

## Next step — bidi

Independent from this swap. Native currently shapes individual runs via HarfBuzz but does not run the Unicode bidi algorithm; mixed LTR/RTL renders in logical order. Upgrade path when needed: slot a bidi analyzer between caller string and the shaper to produce visual runs. Does not touch the host layer.

## Unresolved

- macOS GPU surface via our own Metal glue, or drop to GL3 to halve the glue code (cost: lose Metal backend)?
- Ship clipboard primitives with the swap, or as a separate beat?
- X11 only, or commit to Wayland at the same time?
