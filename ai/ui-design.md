# Triggle UI Design

**Retained-mode** UI: game code owns widget values (`&ui.TextInput{...}`), `ui.App` drives layout, hit testing, and host text-input, and emits a flat `engine/ui/cmd` stream. Same stack on native (CGO + HB+FT) and WASM (per-line canvas raster + optional DOM text-input). Commands go through `engine/render/ui_renderer.go` into the shared per-frame command buffer.

The previous **immediate-mode** API (`*ui.Context`, `BeginWindow`, `StateOf[T]`, vertical pen) is preserved in package **`engine/iui`** as a read-only reference copy (not used by the game). Do not add new product features to `iui` unless you are comparing approaches.

## Using the UI

```go
// init (once) — build a tree, then mount
font, _ := text.OpenFont(backend, fontPath)
th := theme.DefaultTheme()
th.BodyPx  = int32(float32(th.BodyPx)  * dpiScale)
th.TitlePx = int32(float32(th.TitlePx) * dpiScale)

app, _ := ui.NewApp(ui.AppOptions{Backend: backend, Theme: th, Font: font})
log := &ui.LogView{MaxVisible: 12, Color: cmd.Color{R: 235, G: 235, B: 240, A: 255}}
name := &ui.TextInput{Value: "hello", MaxBytes: 128}
hud := &ui.Window{
    Title: "HUD", Pos: emath.Vec2{10, 10}, Width: 360,
    Flags: ui.WindowNoResize | ui.WindowNoClose,
    Child: &ui.Padding{Insets: th.Padding, Child: &ui.Column{
        Kids: []ui.Node{log, name},
    }},
}
app.SetRoot(hud)

// per frame — push log data, then tick
log.Lines = lines
app.Tick(inputFrame, emath.Rect{W: winW, H: winH}, dt)
rnd.SubmitUI(app.Commands(), app.TextureBindings())
if !app.WantsMouse()    { /* game picking / camera */ }
if !app.WantsTextInput() { /* game typing */ }
inputFrame = inputFrame.NextFrame()

// teardown — App first, then Font
app.Close(); font.Close()
```

## New widgets and containers

- Implement `ui.Node` on a struct that embeds `ui.BaseNode` (gives `Rect`, `WidgetID`, `Invalidate`).
- `Measure(Constraints) Size` — intrinsic or max constraint size; `Place(emath.Rect)` — absolute frame coords for `BaseNode.rect` and children.
- `Paint(*PaintCtx)` — `enc.QuadSolid`, `font.Draw` / `DrawVolatile` for text fields with focus, `Enc.PushClip`/`PopClip` as needed.
- `Event` is only used for the focused `TextInput` (App routes `KeyEvents` + `Text` there). Other nodes return false from `Event` unless you add dedicated routing.
- **Containers**: `Column` (`Kids`), `Padding`, `SizedBox`, `Stack` (z-order, top hit last in array). **Window** — title bar, drag via `Pos`, content clip, shrink-to-child width/height.
- **Host text-input** (`Backend.TextInput{Begin,End,Poll}`): `App` reconciles a single session; focused `TextInput` publishes `rect` + buffer each frame. WASM uses the hidden `<input>`; native sokol uses no-op stubs (SDL3 later).

## Non-obvious rules

- **Clip stack is intersected.** `Encoder.PushClip(r)` uses `intersect(top, r)`.
- **Bind id 1** is the app-owned 1×1 white texture. Ids `>= 2` are per-frame dynamic binds from text atlas uploads.
- **DPI** is applied by the caller to `Theme.BodyPx` / `TitlePx` before constructing widgets.
- **Close order:** `App.Close()` before `Font.Close()`; `App` ends the text-input host session and drops volatile line rasters.
- For **legacy ImGui-style** layout (`LayoutNextRow`, `WindowAutoSizeY` + `PatchRect`), read `engine/iui` and the previous git history.

## Developing a new widget (sketch)

```go
type MyWidget struct {
    ui.BaseNode
    parent     ui.Node
    Label      string
}
func (m *MyWidget) Children() []ui.Node { return nil }
func (m *MyWidget) Measure(c ui.Constraints) ui.Size { /* ... */ return ui.Size{W: w, H: h} }
func (m *MyWidget) Place(outer emath.Rect) { m.BaseNode.rect = outer }
func (m *MyWidget) Paint(pc *ui.PaintCtx) { /* enc + font */ }
func (m *MyWidget) Event(*ui.Event, *ui.EventCtx) bool { return false }
```

Call `g.uiApp.SetRoot` again if the tree structure changes, or add children with remount in a follow-up (today: build static trees in `init`).
