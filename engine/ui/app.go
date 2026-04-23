// Package ui is a retained-mode UI: widgets persist across frames, layout runs
// before paint, and commands are emitted to engine/ui/cmd for the renderer.
package ui

import (
	"fmt"

	"triggle/engine/backend"
	"triggle/engine/emath"
	"triggle/engine/text"
	"triggle/engine/ui/cmd"
	"triggle/engine/ui/theme"
)

// AppOptions configures a new UI app.
type AppOptions struct {
	Backend backend.Backend
	Theme   *theme.Theme
	Font    *text.Font
}

// App runs the retained tree and encodes one frame of UI commands per Tick.
type App struct {
	backend backend.Backend
	theme   *theme.Theme
	font    *text.Font
	enc     cmd.Encoder
	white   whiteTex

	root       Node
	nodeSerial uint32

	in       InputFrame
	viewport emath.Rect
	dt       float32

	layoutDirty bool

	focus  *TextInput
	drag   *Window

	wantsMouse bool
	wantsText  bool

	pendingTextInputFocus *TextInput

	textInputHost textInputHostState
}

// textInputHostState is shared with the legacy iui immediate-mode design.
type textInputHostState struct {
	owner     WidgetID
	pollOwner WidgetID
	pollBuf   []byte
	pollCaret int32
	reqOwner  WidgetID
	reqRect   emath.Rect
	reqBuf    []byte
	reqCaret  int32
}

// NewApp creates the app, white texture, and empty root.
func NewApp(opts AppOptions) (*App, error) {
	if opts.Theme == nil {
		return nil, fmt.Errorf("ui: AppOptions.Theme required")
	}
	if opts.Font == nil {
		return nil, fmt.Errorf("ui: AppOptions.Font required")
	}
	w, err := newWhiteTex(opts.Backend)
	if err != nil {
		return nil, err
	}
	return &App{
		backend: opts.Backend,
		theme:   opts.Theme,
		font:    opts.Font,
		white:   w,
	}, nil
}

func (a *App) nextNodeID() uint32 {
	a.nodeSerial++
	if a.nodeSerial == 0 {
		a.nodeSerial = 1
	}
	return a.nodeSerial
}

// SetRoot mounts the root node and assigns fresh ids.
func (a *App) SetRoot(n Node) {
	a.root = n
	if a.root == nil {
		return
	}
	mountNode(a, a.root, nil)
	a.layoutDirty = true
}

// Theme returns the active theme.
func (a *App) Theme() *theme.Theme { return a.theme }

// Font returns the active font.
func (a *App) Font() *text.Font { return a.font }

// Tick runs layout, input, and paint for one frame.
func (a *App) Tick(in InputFrame, viewport emath.Rect, dt float32) {
	a.in = in
	a.viewport = viewport
	a.dt = dt
	a.wantsMouse = false
	a.beginTextInputPoll()
	mx, my := int32(in.MousePos[0]), int32(in.MousePos[1])

	if a.root == nil {
		a.setFocusNode(nil)
		a.endTextInputReconcile()
		return
	}

	a.runLayout()
	a.dispatchInput(mx, my)

	if a.pendingTextInputFocus != nil {
		a.setFocusNode(a.pendingTextInputFocus)
		a.pendingTextInputFocus = nil
	}

	// Re-layout after possible drag
	a.runLayout()

	a.enc.Reset(viewport)
	a.root.Paint(&PaintCtx{Enc: &a.enc, Theme: a.theme, Font: a.font})

	if a.root != nil && hitTestNode(a.root, mx, my) != nil {
		a.wantsMouse = true
	} else if a.drag != nil && a.drag.Dragging {
		a.wantsMouse = true
	}

	a.wantsText = a.focus != nil
	a.endTextInputReconcile()
}

func (a *App) runLayout() {
	if a.root == nil {
		return
	}
	c := normConstraints(Constraints{MaxW: a.viewport.W, MaxH: a.viewport.H})
	_ = a.root.Measure(c)
	a.root.Place(a.viewport)
	a.layoutDirty = false
}

// dispatchInput handles drag, text focus, and routes keys to TextInput.
func (a *App) dispatchInput(mx, my int32) {
	hit := hitTestNode(a.root, mx, my)
	if a.drag != nil && a.drag.Dragging {
		if a.in.MouseDown&MouseLeft != 0 {
			a.drag.Pos[0] = float32(mx - a.drag.grabX)
			a.drag.Pos[1] = float32(my - a.drag.grabY)
			a.drag.Invalidate()
		} else {
			a.drag.Dragging = false
			a.drag = nil
		}
	} else {
		// Start drag from title
		if w, ok := hit.(*Window); ok && a.in.MousePressed&MouseLeft != 0 {
			wx, wy, th, okp := w.titleBarRect()
			if okp {
				tb := emath.Rect{X: wx, Y: wy, W: w.rect.W, H: th}
				if w.Flags&WindowNoTitle == 0 && tb.Contains(mx, my) {
					a.drag = w
					w.Dragging = true
					w.grabX = mx - wx
					w.grabY = my - wy
				}
			}
		}
	}

	if a.in.MousePressed&MouseLeft != 0 {
		if t, ok := hit.(*TextInput); ok {
			a.setFocusNode(t)
		} else {
			a.setFocusNode(nil)
		}
	}

	if a.focus != nil {
		ec := &EventCtx{App: a, Frame: a.in, DT: a.dt, MouseX: mx, MouseY: my}
		b := a.focus.Base()
		ec.LocalX = mx - b.rect.X
		ec.LocalY = my - b.rect.Y
		var ev Event
		ev.Frame = &a.in
		a.focus.Event(&ev, ec)
	}
}

func (a *App) setFocusNode(t *TextInput) {
	if a.focus == t {
		return
	}
	if a.focus != nil && a.font != nil {
		a.font.DropVolatile(uint32(a.focus.WidgetID()))
	}
	a.focus = t
}

// RequestTextFocus requests focus to t on a subsequent Tick.
func (a *App) RequestTextFocus(t *TextInput) {
	a.pendingTextInputFocus = t
}

// FocusedTextInput returns the focused line editor, or nil.
func (a *App) FocusedTextInput() *TextInput { return a.focus }

// Commands returns the UI command stream.
func (a *App) Commands() []cmd.UICmd { return a.enc.Commands() }

// TextureBindings returns bind id to GPU handles (white at 1, then encoder-allocated).
func (a *App) TextureBindings() []TextureBinding {
	dyn := a.enc.Bindings()
	out := make([]TextureBinding, 0, 1+len(dyn))
	out = append(out, TextureBinding{BindID: bindWhite, Image: a.white.image, Sampler: a.white.sampler})
	for i := range dyn {
		d := &dyn[i]
		out = append(out, TextureBinding{BindID: d.BindID, Image: d.Image, Sampler: d.Sampler})
	}
	return out
}

// WantsMouse reports whether the UI is consuming pointer for this frame.
func (a *App) WantsMouse() bool { return a.wantsMouse }

// WantsKeyboard is true when a text field has focus.
func (a *App) WantsKeyboard() bool { return a.focus != nil }

// WantsTextInput is true when a text field has focus.
func (a *App) WantsTextInput() bool { return a.focus != nil }

// Invalidate requests a relayout; n is reserved for future subtree invalidation.
func (a *App) Invalidate(n Node) {
	_ = n
	a.layoutDirty = true
}

// Close releases GPU resources and the text-input session.
func (a *App) Close() {
	b := a.backend
	if a.textInputHost.owner != 0 {
		b.TextInputEnd()
		a.textInputHost.owner = 0
	}
	if a.focus != nil && a.font != nil {
		a.font.DropVolatile(uint32(a.focus.WidgetID()))
		a.focus = nil
	}
	if a.white.image >= 0 {
		b.ImageDestroy(a.white.image)
		a.white.image = -1
	}
	if a.white.sampler >= 0 {
		b.SamplerDestroy(a.white.sampler)
		a.white.sampler = -1
	}
}

func (a *App) beginTextInputPoll() {
	if a.textInputHost.owner == 0 {
		a.textInputHost.pollOwner = 0
		return
	}
	if cap(a.textInputHost.pollBuf) < 512 {
		a.textInputHost.pollBuf = make([]byte, 512)
	} else {
		a.textInputHost.pollBuf = a.textInputHost.pollBuf[:512]
	}
	var caret int32
	n := a.backend.TextInputPoll(a.textInputHost.pollBuf, &caret)
	if n < 0 {
		a.textInputHost.pollOwner = 0
		return
	}
	if int(n) > len(a.textInputHost.pollBuf) {
		big := make([]byte, n)
		n = a.backend.TextInputPoll(big, &caret)
		if n < 0 {
			a.textInputHost.pollOwner = 0
			return
		}
		a.textInputHost.pollBuf = big[:n]
	} else {
		a.textInputHost.pollBuf = a.textInputHost.pollBuf[:n]
	}
	a.textInputHost.pollCaret = caret
	a.textInputHost.pollOwner = a.textInputHost.owner
}

func (a *App) endTextInputReconcile() {
	if a.textInputHost.reqOwner == 0 {
		if a.textInputHost.owner != 0 {
			a.backend.TextInputEnd()
			a.textInputHost.owner = 0
		}
		return
	}
	if a.textInputHost.owner != a.textInputHost.reqOwner {
		r := a.textInputHost.reqRect
		a.backend.TextInputBegin(r.X, r.Y, r.W, r.H, a.textInputHost.reqBuf, a.textInputHost.reqCaret)
		a.textInputHost.owner = a.textInputHost.reqOwner
	}
	a.textInputHost.reqOwner = 0
}

// consumeTextInputPoll is for the focused TextInput (WASM host buffer).
func (a *App) consumeTextInputPoll(id WidgetID) (buf []byte, caret int32, ok bool) {
	if a.textInputHost.pollOwner == 0 || a.textInputHost.pollOwner != id {
		return nil, 0, false
	}
	a.textInputHost.pollOwner = 0
	return a.textInputHost.pollBuf, a.textInputHost.pollCaret, true
}

// publishTextInput is called from TextInput when focused.
func (a *App) publishTextInput(id WidgetID, rect emath.Rect, buf []byte, caret int) {
	if id == 0 {
		return
	}
	a.textInputHost.reqOwner = id
	a.textInputHost.reqRect = rect
	a.textInputHost.reqBuf = append(a.textInputHost.reqBuf[:0], buf...)
	a.textInputHost.reqCaret = int32(caret)
}

// Ensure reqOwner is set every frame when focused — if no publish, end session
// next frame. TextInput must publish in Event.

func mountNode(a *App, n Node, parent Node) {
	if n == nil {
		return
	}
	switch t := n.(type) {
	case *Window:
		t.BaseNode.mount(a)
		t.parent = parent
		if t.Child != nil {
			mountNode(a, t.Child, t)
		}
	case *Column:
		t.BaseNode.mount(a)
		t.parent = parent
		for _, ch := range t.Kids {
			mountNode(a, ch, t)
		}
	case *Padding:
		t.BaseNode.mount(a)
		t.parent = parent
		if t.Child != nil {
			mountNode(a, t.Child, t)
		}
	case *SizedBox:
		t.BaseNode.mount(a)
		t.parent = parent
		if t.Child != nil {
			mountNode(a, t.Child, t)
		}
	case *Stack:
		t.BaseNode.mount(a)
		t.parent = parent
		for _, ch := range t.Kids {
			mountNode(a, ch, t)
		}
	case *Label:
		t.BaseNode.mount(a)
		t.parent = parent
	case *LogView:
		t.BaseNode.mount(a)
		t.parent = parent
	case *TextInput:
		t.BaseNode.mount(a)
		t.parent = parent
	}
}

func hitTestNode(n Node, x, y int32) Node {
	if n == nil {
		return nil
	}
	b := n.Base()
	if !b.rect.Contains(x, y) {
		return nil
	}
	switch t := n.(type) {
	case *Window:
		if t.Child != nil {
			if h := hitTestNode(t.Child, x, y); h != nil {
				return h
			}
		}
		return t
	case *Stack:
		for i := len(t.Kids) - 1; i >= 0; i-- {
			if h := hitTestNode(t.Kids[i], x, y); h != nil {
				return h
			}
		}
		return t
	case *Column:
		for i := len(t.Kids) - 1; i >= 0; i-- {
			if h := hitTestNode(t.Kids[i], x, y); h != nil {
				return h
			}
		}
		return t
	case *Padding:
		if t.Child != nil {
			if h := hitTestNode(t.Child, x, y); h != nil {
				return h
			}
		}
		return t
	case *SizedBox:
		if t.Child != nil {
			if h := hitTestNode(t.Child, x, y); h != nil {
				return h
			}
		}
		return t
	default:
		return t
	}
}

