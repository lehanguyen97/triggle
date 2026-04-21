# Triggle UI Design

Immediate-mode UI (microui-shaped) over `engine/text` (renderer-agnostic shaper/raster) and `engine/emath` (Vec2/Rect/UVRect). Same Go code on native (CGO + HB+FT atlas) and WASM (browser per-line raster). Commands flow through `engine/render/ui_renderer.go` into the shared per-frame command buffer — no separate UI submission.

## Using the UI

```go
// init (once)
font, _ := text.OpenFont(backend, fontPath)
th := theme.DefaultTheme()
th.BodyPx  = int32(float32(th.BodyPx)  * dpiScale)   // caller applies DPI; engine does not
th.TitlePx = int32(float32(th.TitlePx) * dpiScale)
ctx, _ := ui.NewContext(ui.ContextOptions{Backend: backend, Theme: th, Font: font})

// per frame
ctx.Begin(inputFrame, emath.Rect{W: winW, H: winH}, dt)
if ctx.BeginWindow("Log", emath.Rect{X: 10, Y: 10, W: 360, H: 220},
    ui.WindowNoResize|ui.WindowNoClose) {
    ctx.LogView(lines, ui.LogViewOpt{MaxVisible: 12, AutoScroll: true})
    ctx.EndWindow()
}
ctx.End()
rnd.SubmitUI(ctx.Commands(), ctx.TextureBindings())
if !ctx.WantsMouse()    { /* game picking / camera */ }
if !ctx.WantsKeyboard() { /* game hotkeys */ }
inputFrame = inputFrame.NextFrame()                  // clear edge bits

// teardown — Context first, then Font
ctx.Close(); font.Close()
```

## Developing a new widget

Widgets are methods on `*ui.Context`, one file per widget in package `ui` (no build tags). Pattern:

```go
// engine/ui/mywidget.go
package ui

import (
    "triggle/engine/text"
    "triggle/engine/ui/theme"
)

type MyResult struct{ Hovered, Clicked bool }

func (c *Context) MyWidget(label string) MyResult {
    // 1. Identity — push a unique string per call site; pop on return.
    c.idStack = append(c.idStack, "mywidget:"+label)
    id := c.hashID()
    defer func() { c.idStack = c.idStack[:len(c.idStack)-1] }()

    // 2. Per-widget state in the typed pool (zero-init first frame).
    type mws struct{ Hover, Armed bool }
    ws := StateOf[mws](c, id)

    // 3. Layout — carve from the current container.
    r := c.ContentRect()

    // 4. Input — MousePressed/Released are edge bits, MouseDown is level.
    mx, my := int32(c.in.MousePos[0]), int32(c.in.MousePos[1])
    ws.Hover = r.Contains(mx, my)
    if ws.Hover { c.MarkHover() }                    // sets WantsMouse
    clicked := false
    if ws.Hover && c.in.MousePressed&MouseLeft != 0 {
        ws.Armed = true; c.SetActive(id)
    }
    if c.in.MouseReleased&MouseLeft != 0 {
        if ws.Armed && ws.Hover { clicked = true }
        ws.Armed = false; c.ClearActive(id)
    }

    // 5. Draw — StyleBox for background, Font for text, Encoder for custom quads.
    st := theme.StateNormal
    switch {
    case ws.Armed: st = theme.StateActive
    case ws.Hover: st = theme.StateHover
    }
    c.theme.StyleBoxes[theme.ClassButton][st].Draw(&c.enc, r)
    col := c.theme.Colors[theme.ColorText]
    c.font.Draw(&c.enc, label, r.X+4, r.Y+4, c.theme.BodyPx,
        text.Color{R: col.R, G: col.G, B: col.B, A: col.A})

    return MyResult{Hovered: ws.Hover, Clicked: clicked}
}
```

Container (clips + owns a child layout rect): in `BeginX`, pair `c.pushLayout(inner)` + `c.enc.PushClip(inner)`; in `EndX`, pair `c.enc.PopClip()` + `c.popLayout()` + the idStack pop. See `window.go`.

Custom drawing (timeline, curve, inspector): take `&c.enc` and emit `QuadSolid` / `AddTexturedQuad` / `font.Draw` directly. Hit-test against your own data-coord rects.

## Non-obvious rules

- **Clip stack is intersected, not stacked verbatim.** `Encoder.PushClip(r)` pushes `intersect(top, r)`.
- **First UI draw each frame re-applies scissor.** `sg_apply_scissor_rect` is pass-scoped and resets on `sg_begin_pass`; `UIProgram.EmitUI` emits a full-viewport scissor before its first quad.
- **Bind id 1 is the context-owned 1×1 white texture.** Solid quads (`Encoder.QuadSolid`) ride it. Ids ≥ 2 are encoder-allocated per `(image, sampler)` per frame — not stable across frames.
- **Straight alpha, swapchain-native color space.** `SrcAlpha, OneMinusSrcAlpha`, no `pow(2.2)`. Promoting to sRGB or premultiplied is a future milestone, not a local tweak.
- **`DPIScale` is pre-applied to `Theme.BodyPx`/`TitlePx` by the caller.** Widgets treat `BodyPx` as framebuffer pixels; the engine never multiplies by DPI.
- **Persistent state lives in `StateOf[T]`.** Widgets are re-entered every frame; locals and return values disappear.
- **Close order: `Context` before `Font`.** `Context.Close` releases the white texture + state map; `Font.Close` tears down per-size handles (and the shared `textServer` when the last `Font` on that backend closes).
