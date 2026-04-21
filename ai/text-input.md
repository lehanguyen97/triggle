# Text Input

Widget plan for immediate-mode text input on the current native windowing lib (sokol_app) plus the browser DOM on WASM. Widget-owned buffer, ImGui-shape edit model, commit-only IME on native.

## Goal

Type, edit, and navigate a caret inside a single-line text field. LTR scripts only. Latin / numeric / CJK-on-commit. One widget implementation driving a small platform-split editor under it.

## Non-goals

- Bidi (UAX #9). Same scope ImGui core takes.
- RTL script shaping (Arabic connected forms, Hebrew).
- Complex-script shaping (Devanagari conjuncts, Thai vowel marks).
- IME preedit rendering on native. Reason: current native lib delivers committed codepoints only — no preedit surface. CJK users type, see nothing, committed character appears at caret.
- Selection (shift+arrow), clipboard, word-jump, undo. Deferrable without architectural change.

WASM gets bidi and preedit *free* through the DOM overlay — a bonus, not a goal.

## C-ABI additions

**Character-committed event.** One new event kind carrying a UTF-32 codepoint.

```c
G_EVENT_TEXT   // new value in the existing G_EVENT_* enum; payload = codepoint
```

Reason: every windowing lib delivers this exact shape — sokol `CHAR`, SDL `TEXTINPUT`, GLFW char callback. Lib-neutral name and payload mean the ABI doesn't move when the native lib swaps later.

**Editing key codes.** Add to the existing `GK_*` enum:

```c
GK_BACKSPACE, GK_DELETE, GK_LEFT, GK_RIGHT, GK_HOME, GK_END, GK_TAB
```

Reason: minimum set ImGui `InputText` cares about; every text-edit widget needs these.

**Modifier bits** (Shift / Ctrl / Cmd) packed into unused upper bits of the existing `is_repeat` field of the flattened event. Reason: adding a 10th scalar breaks the flattened-event shape on both platforms; high-bit packing is local. Bits must exist even if no widget reads them yet so deferred features (selection, clipboard) don't move the ABI.

**No preedit event in this phase.** Added during the native-host swap, not now.

## Focus on the UI context

Sticky focus id. Distinct from the existing `activeID` (which is drag/press state — transient).

```go
// added to engine/ui Context
focusID WidgetID

func (c *Context) SetFocus(id WidgetID)       { c.focusID = id; c.wantsText = true }
func (c *Context) ClearFocus(id WidgetID)     { if c.focusID == id { c.focusID = 0 } }
func (c *Context) IsFocused(id WidgetID) bool { return c.focusID == id }
```

`End()` clears focus when a mouse press this frame did not land on any focusable widget. `WantsTextInput()` already returns `focusID != 0`.

Reason: ImGui shape. Focus is sticky (cleared by click-outside, not mouse release), drives `WantsTextInput` so the game disables hotkeys while typing.

## Widget — native path

Microui-shape: widget owns the buffer. Go's mutable text buffer is `[]byte` with UTF-8 and rune-aware caret math via `unicode/utf8`. Caret is a byte index.

```go
type TextInputOpt struct {
    Initial  string  // seeded once, on first frame only
    MaxBytes int     // hard cap on buffer length; 0 = no cap
}

func (c *Context) TextInput(label string, opt TextInputOpt) (value string, submitted bool) {
    c.idStack = append(c.idStack, "input:"+label)
    id := c.hashID()
    defer func() { c.idStack = c.idStack[:len(c.idStack)-1] }()

    type state struct {
        buf   []byte  // UTF-8
        caret int     // byte index into buf
        init  bool
        blink float32
    }
    ws := StateOf[state](c, id)
    if !ws.init {
        ws.buf = append(ws.buf[:0], opt.Initial...)
        ws.caret = len(ws.buf)
        ws.init = true
    }

    r := c.ContentRect()
    mx, my := int32(c.in.MousePos[0]), int32(c.in.MousePos[1])
    if r.Contains(mx, my) {
        c.MarkHover()
        if c.in.MousePressed&MouseLeft != 0 { c.SetFocus(id) }
    } else if c.in.MousePressed&MouseLeft != 0 {
        c.ClearFocus(id)
    }

    if c.IsFocused(id) {
        editTextInput(ws, &c.in, opt)        // platform-split; see below
        ws.blink += c.dt
        if c.in.KeyPressed&KeyEnter != 0 { submitted = true }
        if c.in.KeyPressed&KeyEscape != 0 { c.ClearFocus(id) }
    }

    // draw
    c.theme.StyleBoxes[theme.ClassTextInput][styleStateFor(c, id)].Draw(&c.enc, r)
    col := c.theme.Colors[theme.ColorText]
    tc := text.Color{R: col.R, G: col.G, B: col.B, A: col.A}
    c.font.Draw(&c.enc, string(ws.buf), r.X+4, r.Y+4, c.theme.BodyPx, tc)
    if c.IsFocused(id) && int(ws.blink*2)%2 == 0 {
        caretX := r.X + 4 + int32(c.font.Measure(string(ws.buf[:ws.caret]), c.theme.BodyPx)[0])
        c.enc.QuadSolid(emath.Rect{X: caretX, Y: r.Y + 4, W: 1, H: c.theme.BodyPx}, col)
    }
    return string(ws.buf), submitted
}
```

The native editor — microui-tiny; drains the committed character event and a handful of keys:

```go
// editor_native.go
func editTextInput(ws *state, in *InputFrame, opt TextInputOpt) {
    if in.Text != "" && (opt.MaxBytes == 0 || len(ws.buf)+len(in.Text) <= opt.MaxBytes) {
        b := []byte(in.Text)
        ws.buf = append(ws.buf[:ws.caret], append(b, ws.buf[ws.caret:]...)...)
        ws.caret += len(b)
    }
    if in.KeyPressed&KeyBackspace != 0 && ws.caret > 0 {
        _, sz := utf8.DecodeLastRune(ws.buf[:ws.caret])
        ws.buf = append(ws.buf[:ws.caret-sz], ws.buf[ws.caret:]...); ws.caret -= sz
    }
    if in.KeyPressed&KeyDelete != 0 && ws.caret < len(ws.buf) {
        _, sz := utf8.DecodeRune(ws.buf[ws.caret:])
        ws.buf = append(ws.buf[:ws.caret], ws.buf[ws.caret+sz:]...)
    }
    if in.KeyPressed&KeyLeft != 0 && ws.caret > 0 {
        _, sz := utf8.DecodeLastRune(ws.buf[:ws.caret]); ws.caret -= sz
    }
    if in.KeyPressed&KeyRight != 0 && ws.caret < len(ws.buf) {
        _, sz := utf8.DecodeRune(ws.buf[ws.caret:]); ws.caret += sz
    }
    if in.KeyPressed&KeyHome != 0 { ws.caret = 0 }
    if in.KeyPressed&KeyEnd  != 0 { ws.caret = len(ws.buf) }
}
```

## Widget — WASM path (DOM overlay)

Same widget body. Only the editor differs. Browser owns editing; we render.

```go
// editor_wasm.go
func editTextInput(ws *state, in *InputFrame, opt TextInputOpt) {
    focused := /* widget told us */
    switch {
    case focused && !ws.domAttached:
        hostDomFocus(ws.rect, ws.buf, ws.caret) // position + focus hidden <input>
        ws.domAttached = true
    case !focused && ws.domAttached:
        hostDomBlur()
        ws.domAttached = false
    case focused:
        newBuf, newCaret, changed := hostDomPoll() // read .value + .selectionStart
        if changed {
            ws.buf = append(ws.buf[:0], newBuf...)
            ws.caret = newCaret
        }
    }
}
```

Browser side: a single `<input>` absolutely positioned over the canvas, `opacity: 0.01`, `caret-color: transparent`. Element gets focus when a text widget gains UI focus, blurs on click-out. Poll reads `value` and converts `selectionStart` from UTF-16 code units to UTF-8 byte offset.

Reason to delegate: Pixi / Phaser / Three.js games do this and inherit IME + bidi + clipboard + mobile keyboard + a11y free. Rebuilding any of those in-canvas is weeks we don't need to spend.

## Platform split

One editor function, two implementations, chosen at build tag. The split lives at the editor level, not the widget level — widget code above is shared verbatim. No file layout prescribed here beyond that.

## Next step

When commit-only CJK is no longer acceptable (in-canvas preedit needed, or native clipboard / Linux IME matter), do the host swap — see `sdl3-native.md`. That change is local to the host layer and adds a preedit event; this widget gains a preedit-render branch without a rewrite.

## Unresolved

- Re-entering `TextInput` with a new `Initial` — seed only on first frame (current sketch) or reset buffer when `Initial` changes? First-frame-only is simplest and matches IM semantics; caller can force reset by allocating a new `WidgetID`.
    - Seed only on first frame (no data in state)
- Return a third bool `changed` for "value differs from last frame", or let callers diff against their own prior value?
    - Not sure, do what better or imgui do
- Caret height = `BodyPx` (sketch above) or real `font.Metrics(px).Ascent+Descent`? Only visible with unusual fonts; `BodyPx` is simpler.
    - BodyPx
- Key-event-to-bit mapping for `KeyEnter/KeyEscape` etc. — assign specific bit positions up front in `input.go`, or assign on first use? Specific bit map is cleaner and avoids future collisions.
    - bit map
