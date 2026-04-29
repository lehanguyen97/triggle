// Package ui is a retained-mode UI: widgets persist across frames, layout runs
// before paint, and commands are emitted to the internal paint stream.
package ui

import (
	"fmt"

	"triggle/engine/backend"
	"triggle/engine/draw2d"
	"triggle/engine/emath"
	"triggle/engine/render"
	"triggle/engine/text"
)

// RootOptions configures a new UI root.
type RootOptions struct {
	Theme *Theme
}

// Root runs the retained tree and records one frame of draw2d commands per Tick.
type Root struct {
	backend      backend.Backend
	theme        *Theme
	fonts        *text.FontSet
	drawCtx      *draw2d.Context
	drawRenderer *draw2d.Renderer

	root       Node
	nodeSerial uint32

	in      InputFrame
	vpState ViewportState
	dt      float32

	layoutDirty bool

	focus *TextInput
	drag  *Window
	hover Node

	wantsMouse  bool
	wantsScroll bool
	wantsText   bool

	pendingTextInputFocus *TextInput

	textSession hostTextSession
}

// hostTextSession bridges the focused TextInput widget to the host's
// text-input session (Backend.TextInput{Begin,End,Poll}).
type hostTextSession struct {
	owner     WidgetID
	pollOwner WidgetID
	pollBuf   []byte
	pollCaret int32
	reqOwner  WidgetID
	reqRect   emath.Rect
	reqBuf    []byte
	reqCaret  int32
}

// NewRoot creates the retained UI root and draw2d renderer.
func NewRoot(be backend.Backend, fonts *text.FontSet, opts RootOptions) (*Root, error) {
	if opts.Theme == nil {
		return nil, fmt.Errorf("ui: RootOptions.Theme required")
	}
	if fonts == nil {
		return nil, fmt.Errorf("ui: font set required")
	}
	fontID := opts.Theme.Text.Font
	if fontID == "" {
		fontID = text.FontDefault
	}
	if fonts.Font(fontID) == nil {
		return nil, fmt.Errorf("ui: font %q not registered", fontID)
	}
	dr, err := draw2d.NewRenderer(be)
	if err != nil {
		return nil, err
	}
	r := &Root{
		backend:      be,
		theme:        opts.Theme,
		fonts:        fonts,
		drawCtx:      draw2d.NewContext(),
		drawRenderer: dr,
	}
	r.vpState.UIScale = 1
	r.vpState.DPIScale = 1
	return r, nil
}

func (a *Root) nextNodeID() uint32 {
	a.nodeSerial++
	if a.nodeSerial == 0 {
		a.nodeSerial = 1
	}
	return a.nodeSerial
}

// SetRoot mounts the root node and assigns fresh ids.
func (a *Root) SetRoot(n Node) {
	a.root = n
	if a.root == nil {
		return
	}
	mountNode(a, a.root)
	a.layoutDirty = true
}

// Theme returns the active theme.
func (a *Root) Theme() *Theme { return a.theme }

// Font returns the active font.
func (a *Root) Font() *text.Font {
	if a == nil || a.theme == nil {
		return nil
	}
	if a.fonts == nil {
		return nil
	}
	return a.fonts.Font(a.theme.Text.Font)
}

// UIScale returns the active logical→physical multiplier.
func (a *Root) UIScale() float32 {
	if a == nil {
		return 1
	}
	if a.vpState.UIScale <= 0 {
		return 1
	}
	return a.vpState.UIScale
}

func (a *Root) SetViewport(fb emath.Rect, dpi float32) {
	if a.vpState.Framebuffer == fb && a.vpState.DPIScale == dpi {
		return
	}
	a.vpState.Framebuffer = fb
	a.vpState.DPIScale = dpi
	a.recomputeViewport()
}

// recomputeViewport: UI is DPI-only responsive. lp viewport = fb / DPIScale;
// widgets keep their lp size at any window size and layout reflows via
// Flex / ScrollView / ClampFrac.
func (a *Root) recomputeViewport() {
	fb := a.vpState.Framebuffer
	dpi := max(a.vpState.DPIScale, 1)
	a.vpState.UIScale = dpi
	if fb.W <= 0 || fb.H <= 0 {
		a.vpState.Design = fb
	} else {
		a.vpState.Design = emath.Rect{W: fb.W / dpi, H: fb.H / dpi}
	}
	a.layoutDirty = true
}

func (a *Root) Viewport() ViewportState { return a.vpState }

// Input returns a pointer to the App's accumulated InputFrame.
func (a *Root) Input() *InputFrame {
	if a == nil {
		return nil
	}
	return &a.in
}

// Tick runs layout, input, and paint for one frame. The caller must NOT reset
// the InputFrame; Tick resets it at the end of the call.
//
// Order (single layout pass per tick — Flutter/Compose model):
//
//	mount/tick → mouse fb→lp → hit-test prior-frame rects → dispatchInput
//	→ runLayout (reads input mutations) → paint.
func (a *Root) Tick(dt float32) {
	if a.fonts != nil {
		a.fonts.BeginFrame()
		defer a.fonts.EndFrame()
	}
	a.dt = dt
	a.wantsMouse = false
	a.wantsScroll = false
	a.beginTextInputPoll()

	if a.root == nil {
		a.setFocusNode(nil)
		a.endTextInputReconcile()
		a.in = a.in.NextFrame()
		return
	}

	mountNode(a, a.root)
	tickNodeTree(a.root, dt)

	mxLp, myLp := a.mouseLp()
	hit := hitTestNode(a.root, mxLp, myLp)
	a.hover = hit
	a.dispatchInput(mxLp, myLp, hit)

	if a.pendingTextInputFocus != nil {
		a.setFocusNode(a.pendingTextInputFocus)
		a.pendingTextInputFocus = nil
	}

	a.runLayout()

	a.drawCtx.Reset(a.vpState.Design)
	a.root.Paint(&PaintCtx{Context: a.drawCtx, Theme: a.theme, Root: a})

	if hit != nil {
		a.wantsMouse = true
	} else if a.drag != nil && a.drag.Dragging {
		a.wantsMouse = true
	}

	a.wantsText = a.focus != nil
	a.endTextInputReconcile()
	a.in = a.in.NextFrame()
}

// mouseLp converts the latest fb-px mouse pos to lp using UIScale.
func (a *Root) mouseLp() (float32, float32) {
	scale := a.vpState.UIScale
	if scale <= 0 {
		scale = 1
	}
	return a.in.MousePos[0] / scale, a.in.MousePos[1] / scale
}

func (a *Root) runLayout() {
	if a.root == nil {
		return
	}
	vp := a.vpState.Design
	c := normConstraints(Constraints{MaxW: vp.W, MaxH: vp.H})
	_ = a.root.Measure(c)
	a.root.Place(vp)
	a.layoutDirty = false
}

// dispatchInput handles drag, scroll, text focus, and routes keys to TextInput.
// Coordinates are lp; hit was computed from prior-frame rects.
func (a *Root) dispatchInput(mx, my float32, hit Node) {
	// Window drag (mutates Window.Pos before runLayout below).
	if a.drag != nil && a.drag.Dragging {
		if a.in.MouseDown&MouseLeft != 0 {
			a.drag.Pos[0] = mx - a.drag.grabX
			a.drag.Pos[1] = my - a.drag.grabY
			a.drag.Invalidate()
		} else {
			a.drag.Dragging = false
			a.drag = nil
		}
	} else if w, ok := hit.(*Window); ok && w.Flags&WindowUseParentRect == 0 && a.in.MousePressed&MouseLeft != 0 {
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

	if a.in.MousePressed&MouseLeft != 0 {
		if t, ok := hit.(*TextInput); ok {
			a.setFocusNode(t)
		} else {
			a.setFocusNode(nil)
		}
	}

	// Scroll routing: find innermost ScrollView under cursor and apply delta.
	if a.in.ScrollDelta[1] != 0 || a.in.ScrollDelta[0] != 0 {
		sv := findScrollTarget(a.root, mx, my)
		if sv != nil {
			step := a.theme.BodyLp * 3
			if step <= 0 {
				step = 48
			}
			before := sv.Offset
			sv.Offset -= a.in.ScrollDelta[1] * step
			if sv.Offset < 0 {
				sv.Offset = 0
			}
			if max := sv.MaxOffset(); sv.Offset > max {
				sv.Offset = max
			}
			if sv.Offset != before {
				a.wantsScroll = true
				sv.Invalidate()
			}
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

// findScrollTarget returns the innermost ScrollView whose rect contains (x,y).
func findScrollTarget(n Node, x, y float32) *ScrollView {
	if n == nil {
		return nil
	}
	b := n.Base()
	if b == nil || !b.rect.Contains(x, y) {
		return nil
	}
	kids := n.Children()
	for i := len(kids) - 1; i >= 0; i-- {
		if deeper := findScrollTarget(kids[i], x, y); deeper != nil {
			return deeper
		}
	}
	if sv, ok := n.(*ScrollView); ok {
		return sv
	}
	return nil
}

func (a *Root) setFocusNode(t *TextInput) {
	if a.focus == t {
		return
	}
	if font := a.Font(); a.focus != nil && font != nil {
		font.DropVolatile(text.OwnerID(a.focus.WidgetID()))
	}
	a.focus = t
}

// RequestTextFocus requests focus to t on a subsequent Tick.
func (a *Root) RequestTextFocus(t *TextInput) {
	a.pendingTextInputFocus = t
}

// FocusedTextInput returns the focused line editor, or nil.
func (a *Root) FocusedTextInput() *TextInput { return a.focus }

// Hover returns the topmost hit node from the prior dispatch (paint-time read).
func (a *Root) Hover() Node {
	if a == nil {
		return nil
	}
	return a.hover
}

// WantsMouse reports whether the UI is consuming pointer for this frame.
func (a *Root) WantsMouse() bool { return a.wantsMouse }

// WantsScroll reports whether the UI consumed scroll this frame.
func (a *Root) WantsScroll() bool { return a.wantsScroll }

// WantsKeyboard is true when a text field has focus.
func (a *Root) WantsKeyboard() bool { return a.focus != nil }

// WantsTextInput is true when a text field has focus.
func (a *Root) WantsTextInput() bool { return a.focus != nil }

// Invalidate requests a relayout; n is reserved for future subtree invalidation.
func (a *Root) Invalidate(n Node) {
	_ = n
	a.layoutDirty = true
}

// Close releases GPU resources and the text-input session.
func (a *Root) Close() {
	b := a.backend
	if a.textSession.owner != 0 {
		b.TextInputEnd()
		a.textSession.owner = 0
	}
	if font := a.Font(); a.focus != nil && font != nil {
		font.DropVolatile(text.OwnerID(a.focus.WidgetID()))
		a.focus = nil
	}
	if a.drawRenderer != nil {
		a.drawRenderer.Release()
		a.drawRenderer = nil
	}
}

func (a *Root) beginTextInputPoll() {
	if a.textSession.owner == 0 {
		a.textSession.pollOwner = 0
		return
	}
	if cap(a.textSession.pollBuf) < 512 {
		a.textSession.pollBuf = make([]byte, 512)
	} else {
		a.textSession.pollBuf = a.textSession.pollBuf[:512]
	}
	var caret int32
	n := a.backend.TextInputPoll(a.textSession.pollBuf, &caret)
	if n < 0 {
		a.textSession.pollOwner = 0
		return
	}
	if int(n) > len(a.textSession.pollBuf) {
		big := make([]byte, n)
		n = a.backend.TextInputPoll(big, &caret)
		if n < 0 {
			a.textSession.pollOwner = 0
			return
		}
		a.textSession.pollBuf = big[:n]
	} else {
		a.textSession.pollBuf = a.textSession.pollBuf[:n]
	}
	a.textSession.pollCaret = caret
	a.textSession.pollOwner = a.textSession.owner
}

func (a *Root) endTextInputReconcile() {
	if a.textSession.reqOwner == 0 {
		if a.textSession.owner != 0 {
			a.backend.TextInputEnd()
			a.textSession.owner = 0
		}
		return
	}
	if a.textSession.owner != a.textSession.reqOwner {
		r := a.textSession.reqRect
		a.backend.TextInputBegin(int32(r.X), int32(r.Y), int32(r.W), int32(r.H), a.textSession.reqBuf, a.textSession.reqCaret)
		a.textSession.owner = a.textSession.reqOwner
	}
	a.textSession.reqOwner = 0
}

// consumeTextInputPoll is for the focused TextInput (WASM host buffer).
func (a *Root) consumeTextInputPoll(id WidgetID) (buf []byte, caret int32, ok bool) {
	if a.textSession.pollOwner == 0 || a.textSession.pollOwner != id {
		return nil, 0, false
	}
	a.textSession.pollOwner = 0
	return a.textSession.pollBuf, a.textSession.pollCaret, true
}

// publishTextInput is called from TextInput when focused.
func (a *Root) publishTextInput(id WidgetID, rect emath.Rect, buf []byte, caret int) {
	if id == 0 {
		return
	}
	a.textSession.reqOwner = id
	a.textSession.reqRect = rect
	a.textSession.reqBuf = append(a.textSession.reqBuf[:0], buf...)
	a.textSession.reqCaret = int32(caret)
}

// mountNode wires unmounted nodes into the root and recurses into Children().
func mountNode(a *Root, n Node) {
	if n == nil {
		return
	}
	if b := n.Base(); b != nil && b.app == nil {
		b.mount(a)
	}
	for _, ch := range n.Children() {
		mountNode(a, ch)
	}
}

func tickNodeTree(n Node, dt float32) {
	if n == nil {
		return
	}
	if t, ok := n.(TickingNode); ok {
		t.TickNode(dt)
	}
	for _, ch := range n.Children() {
		tickNodeTree(ch, dt)
	}
}

// hitTestNode returns the topmost node containing (x,y). Children are walked in
// reverse so later-painted siblings (Stack/Flex z-order) win.
func hitTestNode(n Node, x, y float32) Node {
	if n == nil {
		return nil
	}
	b := n.Base()
	if b == nil || !b.rect.Contains(x, y) {
		return nil
	}
	kids := n.Children()
	for i := len(kids) - 1; i >= 0; i-- {
		if h := hitTestNode(kids[i], x, y); h != nil {
			return h
		}
	}
	return n
}

// Render records UI draw commands into the renderer's open default pass.
func (a *Root) Render(r *render.Server) {
	if a == nil || a.drawRenderer == nil || a.drawCtx == nil {
		return
	}
	a.drawRenderer.RenderToFrame(r, a.drawCtx.List(), a.UIScale())
}
