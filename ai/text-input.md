# Text Input

Single-line, immediate-mode, widget-owned UTF-8 buffer. Native sokol = local
editor; WASM = hidden `<input>` overlay (browser owns IME, clipboard, bidi).

Code: `engine/ui/text_input.go` (widget body, editor, host bookkeeping all in
one file). Backend split: `engine/backend/host_{native,wasm}.go` implement
`Backend.TextInput{Begin,End,Poll}` (SDL3-named).

## Scope

In: type, edit, navigate caret. Latin / numeric / CJK-on-commit. Backspace,
Delete, Left, Right, Home, End, Tab, Enter (submit), Escape (blur).

Out: bidi, RTL/complex shaping, IME preedit on native (committed-only — no
preedit surface in current native lib), selection, clipboard, word-jump, undo.
WASM gets bidi + preedit free via the DOM overlay.

## C-ABI

- `G_EVENT_TEXT` — UTF-32 codepoint payload. Maps to sokol `CHAR`, SDL
  `TEXTINPUT`, GLFW char callback, browser `input`.
- `GK_*` — editing keys only (`BACKSPACE/DELETE/LEFT/RIGHT/HOME/END/TAB/
  ENTER/ESCAPE`). Character keys are not enumerated; they arrive as `EvText`
  codepoints already mapped through OS layout/dead-keys/IME.
- Modifier bits packed into upper bits of `is_repeat` (Shift/Ctrl/Alt/Cmd =
  0x100/0x200/0x400/0x800). Reserved even if no widget reads them yet.

`G_EVENT_TEXT` and `G_EVENT_KEY_*` are **not redundant**. Text gives committed
characters (post-shift, post-IME, post-deadkey). Keys give physical
transitions (needed for Enter/Backspace/arrows, hotkeys with modifiers,
held-vs-autorepeat). Every windowing lib splits them; we mirror that split.

## Focus on Context

Sticky `focusID` (distinct from `activeID`, which is drag/press transient).
`SetFocus` / `ClearFocus` / `IsFocused`. `End()` clears focus on a click that
missed every `RegisterFocusable` rect. `WantsTextInput` = `focusID != 0` so
the game can disable hotkeys while typing.

## WASM host overlay

A single `<input>` absolutely positioned over the canvas, `opacity: 0.01`,
`caret-color: transparent`, `pointer-events: none`. Element gains focus when
a text widget gains UI focus, blurs on click-out. JS handles IME, paste,
clipboard, mobile keyboard, a11y — rebuilding any of those in-canvas is weeks
we don't need to spend.

Per-frame contract:

1. `Context.Begin` polls the host once → `(pollOwner, pollBuf, pollCaret)`.
2. Widget consumes via `consumeTextInputPoll(id)`; on hit it overwrites its
   own buffer + caret.
3. Widget skips the local editor on the consume-true frame. Necessary because
   sokol bubbles key/char to the canvas (`html5_bubble_*=true`), so editing
   keys also reach `in.KeyEvents` / `in.Text`. Running both = per-keystroke
   double-apply jitter.
4. While focused, widget calls `publishTextInput(id, rect, buf, caret)`.
5. `Context.End` reconciles: same owner = no-op; new owner = `TextInputBegin`
   (re-seeds element); no request + active owner = `TextInputEnd` (blur).

Poll converts `selectionStart` from UTF-16 code units to UTF-8 byte offset.

## Native = no-op stubs

`Backend.TextInput*` are no-ops on sokol; the widget's own buffer is the
source of truth. `consumeTextInputPoll` returns `ok=false` (Poll returns -1),
so the editor runs unconditionally. SDL3 host swap (see `sdl3-native.md`)
fills these stubs in with `SDL_StartTextInput` / `SDL_StopTextInput` /
`SDL_SetTextInputArea` — zero `engine/ui` change required.

## Decisions

- **Initial seeding**: only on first frame (`!ws.init`). Re-seed = caller
  uses a different label (different `WidgetID`).
- **Caret height = ascent + descent** (line box, not `BodyPx`). Rendered
  bitmap reaches descent; `BodyPx` clipped descenders ('g', 'p', 'y') in the
  parent window's clip.
- **Widget H = lineH + 2*padY** with `padX=4, padY=3` package constants.
  Promote to `theme.Theme` when a second widget needs them.
- **Horizontal scroll**: persistent `scrollX`; keeps caret in
  `[0, contentW]`; widget pushes its own clip rect. No margin, no easing.
- **Editor / host arbitration**: widget skips `editTextInput` on any frame
  where `consumeTextInputPoll` returned true. Sokol's `html5_bubble_*=true`
  is why both editors would otherwise fire.

## Follow-ups (deferred — see ai/text-review.md "Follow-ups")

Selection, clipboard, word-jump, undo/redo, caret-from-click, multi-line,
password mode, placeholder, focus traversal, `IsItemEdited`. Order and
gating notes live in text-review.md. Selection + clipboard is the highest
leverage step; everything after depends on selection.
