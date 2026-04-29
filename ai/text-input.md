# Text Input

Single-line UTF-8 widget with a host-owned editing session.

## Platform split
- **Native (today)**: local editor — keypress maps to buffer edits. No IME preedit surface, no clipboard.
- **WASM**: hidden transparent `<input>` over the canvas. Browser owns IME, clipboard, bidi, mobile keyboard, autofill. Go reads buffer + caret each frame.
- **Native (after the SDL3 swap)**: fills the session primitives with real IME, clipboard, and explicit text-input control. No widget-side change. See `ai/sdl3-native.md`.

## Session bridge
Per tick the root and focused widget exchange state via `Backend.TextInput.{Begin,End,Poll}`:
1. Root polls host → current owner, buffer, caret.
2. Focused widget consumes the poll result, overwriting its own state.
3. Widget skips local key/text editing that frame if consume succeeded — both the browser overlay and the canvas receive keystrokes on WASM; running both editors double-applies.
4. Widget publishes its rect + current buffer + caret.
5. Root reconciles ownership transitions: new owner begins a session, lost owner ends one.

## Events — two kinds, not redundant
- **Text events** carry a UTF-32 codepoint, already shift / IME / dead-key resolved.
- **Key events** carry physical transitions: Backspace, Enter, arrows, modifier hotkeys, held-vs-autorepeat.
- Every windowing lib splits these; we mirror the split. Modifier bits are reserved in key events even if no widget reads them yet.

## Focus model
Sticky focus, distinct from press/drag transient state. Click-out clears. `Root.WantsTextInput` lets the game disable hotkeys while typing.

## Scope
**In:** Latin / numeric / CJK-on-commit, caret nav, backspace, delete, home, end, tab, enter = submit, escape = blur.
**Out:** bidi, selection, clipboard on native, word-jump, undo, multi-line, password, placeholder.

## Caret and layout details
- Caret height = ascent + descent. Body-px alone clips descenders on 'g', 'p', 'y'.
- Horizontal scroll keeps caret in-rect; widget pushes its own clip.
- First-frame seeding only; a different widget identity re-seeds.

## Submit ordering (controlled inputs)
`OnSubmit` runs *inside* `TextInput.Event`, before `applyEdits`. If the handler mutates bound state, `buf` still gets written back via `Binding.Set` at the bottom of `Event`, stomping the clear. Submit handler must:
1. Write to game state (e.g. `chat.Draft = ""`).
2. Reset the input via `t.SetValue("")` so `t.buf` matches the new bound value before `applyEdits` runs.

After return, `applyEdits` sees `Value == ""`, `Binding.Set` is *not* called with stale text, next frame's `ensureInit` resync is a no-op.

## Follow-ups
Selection + clipboard is the highest-leverage next step. Word-jump, undo/redo, caret-from-click, multi-line, password mode all depend on a selection model landing first.
