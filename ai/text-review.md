# Text Pipeline — Review

Living status of the text + text-input stack. Resolved items are in git
history; this doc lists only what's open or planned.

## Open — text pipeline

- **Atlas growth + glyph reclaim.** 512×512 native atlas with no grow / no
  eviction; `closeFont` doesn't prune `glyphAtlas.glyphs`. Required before
  CJK or multi-size loads. Plan: double-and-repack, or rollover with per-line
  refcount.
- **`string(buf)` per frame in `Font.Draw` / `Measure`.** Steady-state typing
  re-allocates every frame. Add `Font.DrawBytes` / `MeasureBytes` that takes
  `[]byte` and only converts on cache miss. Defer until profiling.
- **`lineKey.content` keeps full-string copies in the LRU.** Pixi keys by
  hash. No fix planned.
- **Volatile line leak on widget disappearance.** `Context.End` doesn't sweep
  `c.states` for "wasn't called this frame". Cheap fix: clear focus in
  `End()` if `focusID` not in `focusableRects`. Correct fix: per-id touched
  set + N-frame sweep (microui shape).
- **Hidden `<input>` never removed from DOM.** Single element, fine for SPA;
  expose `dispose` only if hot-reload becomes a workflow.
- **`reqBuf` / `pollBuf` only grow.** Bounded by max-ever-pasted; note for
  future paste-size limit.
- **WASM raster can OOM the backend heap silently.** Add `console.warn` when
  `numBytes > 4MB` before `_backend_malloc`.
- **`gameMem()` allocates a `DataView` per call.** Cache and refresh only on
  `memory.grow`. Low priority.
- **`textServer.fonts` map has no mutex.** UI is single-threaded; document
  the assumption in `engine/text/text.go` package doc.

## Open — text-input layout

- **Caret height = line box.** Some UIs draw caret = ascent only (cleaner
  above baseline, no descender stub when buffer is empty). Decide once
  selection rendering lands — same rect math will need a verdict.
- **No fit-to-content windows.** `BeginWindow` takes a fixed rect from the
  caller; we just compute it from per-component size helpers
  (`ui.TextInputSize` / `ui.LogViewSize`). A real auto-sizing pass (collect
  children, finalize H on End) is the next move if more single-row windows
  show up.
- **Scroll has no horizontal margin.** Caret hugs the right edge once text
  overflows; pushing it 2-3 chars away (Pixi/imgui shape) reads better.
  Defer until a real test catches the awkwardness.

## Follow-ups — ImGui-parity features (priority order)

Gated by selection, which is the dependency for almost everything else.

1. **Caret-from-click hit testing + selection** (shift+arrows, shift+click,
   double-click word). Adds `selStart` to widget state; rendering is one
   `QuadSolid` behind the text.
2. **Clipboard hooks on `Backend`** (`ClipboardGet/Set`). Same shape as the
   text-input host: WASM via async `navigator.clipboard`; native sokol
   returns empty until SDL3 swap. Wire Ctrl/Cmd+C/X/V in the editor.
3. **`changed` / `IsItemEdited` return** — close the open widget API item.
4. **Read-only flag + placeholder hint** — both 5-liners; immediately useful
   for forms.
5. **Tab to next focusable.** Needs `RegisterFocusable` to also record the
   widget id (today it just records rects). Then `End()` cycles focus on Tab.
6. **Word-jump + Ctrl+Backspace** — depends on selection (Ctrl+Shift+Right
   is the usual pair).
7. **Undo/redo** — port stb_textedit-style undo ring or roll a simpler one
   (per-edit snapshot of buf+caret with a 32-step ring).

Multi-line, password, numeric variants — only when an actual screen needs
them.

## References

- Pixi: `~/ws-local/pixijs/src/scene/text/canvas/CanvasTextMetrics.ts`
  (`measureFont`, `METRICS_STRING`, `BASELINE_SYMBOL`).
- Babylon: `~/ws-local/Babylon.js/packages/dev/core/src/Engines/engine.common.ts`
  (`GetFontOffset`); `Control._FontHeightSizes` cache in
  `~/ws-local/Babylon.js/packages/dev/gui/src/2D/controls/control.ts`.
- TextMetrics spec: <https://html.spec.whatwg.org/multipage/canvas.html#textmetrics>
