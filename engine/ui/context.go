// Package ui is the immediate-mode UI front-end.
// Widgets are rebuilt each frame; per-widget persistent state lives in a typed pool (StateOf[T]).
// Commands are emitted into a flat stream (ui/cmd) and batched by the UI renderer.
package ui

import (
	"fmt"
	"unsafe"

	"triggle/engine/backend"
	"triggle/engine/emath"
	"triggle/engine/shader"
	"triggle/engine/text"
	"triggle/engine/ui/cmd"
	"triggle/engine/ui/theme"
)

// WidgetID is a stable hashed identity for retained per-widget state.
type WidgetID uint32

// TextureBinding maps a bind id to backend image/sampler for UI draws.
// Bind id 1 is the context-owned 1×1 white texture. Ids ≥ 2 are encoder-allocated per frame.
type TextureBinding struct {
	BindID  uint32
	Image   int32
	Sampler int32
}

// bindWhite is the reserved bind id for the context-owned 1×1 white texture.
const bindWhite uint32 = 1

// ContextOptions configures a new UI context. All fields are required.
// Font is caller-owned (and shares a TextServer with any other Fonts on the
// same backend); Context does not close it.
type ContextOptions struct {
	Backend backend.Backend
	Theme   *theme.Theme
	Font    *text.Font
}

type whiteTex struct {
	image, sampler int32
}

// Context drives one frame of immediate-mode UI.
type Context struct {
	backend backend.Backend
	theme   *theme.Theme
	font    *text.Font

	enc   cmd.Encoder
	white whiteTex

	in       InputFrame
	viewport emath.Rect
	dt       float32

	idStack []string
	states  map[WidgetID]any

	activeID   WidgetID
	focusID    WidgetID
	wantsMouse bool
	wantsKb    bool
	wantsText  bool

	focusableRects []emath.Rect

	layoutStack []layoutFrame

	textInputHost textInputHostState
}

// layoutFrame is one entry on Context.layoutStack: the slot a container hands
// to its children plus a vertical pen that advances per row. ImGui calls this
// `window->DC` (cursor + content rect); we keep just the pieces widgets need.
//
// rect.H/rect.W may be MaxInt32 for auto-sized windows — the pen still
// advances and EndWindow patches the placeholder bg + clip commands using
// (cursorY, maxX, win* + pad*) once content is measured.
type layoutFrame struct {
	rect    emath.Rect
	startY  int32
	cursorY int32
	maxX    int32
	spacing int32

	// Auto-size-Y patch payload (only set by BeginWindow when the caller
	// requested WindowAutoSizeY). EndWindow rewrites the bgCmd quad and
	// clipCmd push to the measured outer rect once cursorY has advanced.
	// Width auto-size isn't supported yet; it would need a second pair of
	// patch indices for the title bar and is rarely useful in practice.
	autoH   bool
	bgCmd   int
	clipCmd int
	winX    int32
	winY    int32
	winW    int32
	titleH  int32
	padBot  int32
}

// NewContext validates options and creates the shared white texture (bind 1).
func NewContext(opts ContextOptions) (*Context, error) {
	if opts.Theme == nil {
		return nil, fmt.Errorf("ui: ContextOptions.Theme required")
	}
	if opts.Font == nil {
		return nil, fmt.Errorf("ui: ContextOptions.Font required")
	}
	w, err := newWhiteTex(opts.Backend)
	if err != nil {
		return nil, err
	}
	return &Context{
		backend: opts.Backend,
		theme:   opts.Theme,
		font:    opts.Font,
		white:   w,
		states:  make(map[WidgetID]any),
	}, nil
}

func newWhiteTex(b backend.Backend) (whiteTex, error) {
	img := b.ImageCreateTexture(1, 1, shader.PixfmtRGBA8)
	if img < 0 {
		return whiteTex{}, fmt.Errorf("ui: white texture create failed")
	}
	smp := b.SamplerCreate(shader.FilterNearest, shader.FilterNearest, shader.WrapClampToEdge, shader.CmpNone)
	if smp < 0 {
		b.ImageDestroy(img)
		return whiteTex{}, fmt.Errorf("ui: white sampler create failed")
	}
	pix := [4]byte{255, 255, 255, 255}
	b.ImageUpdateRGBA8(img, 1, 1, unsafe.Pointer(&pix[0]), 4)
	return whiteTex{image: img, sampler: smp}, nil
}

// StateOf returns the typed per-widget state slot for id.
// First call zero-initializes; subsequent calls return the same *T.
// Panics if T conflicts with an existing entry under id (indicates a WidgetID hash collision).
func StateOf[T any](c *Context, id WidgetID) *T {
	if c == nil {
		return new(T)
	}
	if c.states == nil {
		c.states = make(map[WidgetID]any)
	}
	if v, ok := c.states[id]; ok {
		t, ok := v.(*T)
		if !ok {
			panic(fmt.Sprintf("ui: StateOf[%T] conflict for id %d (existing %T)", *new(T), id, v))
		}
		return t
	}
	t := new(T)
	c.states[id] = t
	return t
}

func (c *Context) hashID() WidgetID {
	h := uint32(2166136261)
	for _, s := range c.idStack {
		for i := 0; i < len(s); i++ {
			h ^= uint32(s[i])
			h *= 16777619
		}
		h ^= 0x9e3779b9
	}
	if h == 0 {
		h = 1
	}
	return WidgetID(h)
}

// Begin starts a UI frame.
func (c *Context) Begin(in InputFrame, viewport emath.Rect, dt float32) {
	if c == nil {
		return
	}
	c.in = in
	c.viewport = viewport
	c.dt = dt
	c.idStack = c.idStack[:0]
	c.layoutStack = c.layoutStack[:0]
	c.focusableRects = c.focusableRects[:0]
	c.enc.Reset(viewport)
	c.wantsMouse = false
	c.wantsKb = false
	c.wantsText = c.focusID != 0
	c.beginTextInputPoll()
}

// End finalizes the frame.
func (c *Context) End() {
	if c == nil {
		return
	}
	if c.activeID != 0 {
		c.wantsMouse = true
	}
	// Click outside any focusable widget clears focus.
	if c.focusID != 0 && c.in.MousePressed&MouseLeft != 0 {
		mx, my := int32(c.in.MousePos[0]), int32(c.in.MousePos[1])
		hit := false
		for _, r := range c.focusableRects {
			if r.Contains(mx, my) {
				hit = true
				break
			}
		}
		if !hit {
			c.focusID = 0
			c.wantsText = false
		}
	}
	c.endTextInputReconcile()
}

// Commands returns the UI command stream for SubmitUI.
func (c *Context) Commands() []cmd.UICmd { return c.enc.Commands() }

// TextureBindings returns bind id to GPU handles (white at 1, then encoder-allocated ids).
func (c *Context) TextureBindings() []TextureBinding {
	if c == nil {
		return nil
	}
	dyn := c.enc.Bindings()
	out := make([]TextureBinding, 0, 1+len(dyn))
	out = append(out, TextureBinding{
		BindID:  bindWhite,
		Image:   c.white.image,
		Sampler: c.white.sampler,
	})
	for i := range dyn {
		d := &dyn[i]
		out = append(out, TextureBinding{BindID: d.BindID, Image: d.Image, Sampler: d.Sampler})
	}
	return out
}

// Input returns the current frame input snapshot.
func (c *Context) Input() InputFrame { return c.in }

// Theme returns the active theme.
func (c *Context) Theme() *theme.Theme { return c.theme }

// Font returns the active font (widgets draw directly on it; custom views may query metrics).
func (c *Context) Font() *text.Font { return c.font }

// Encoder returns the raw command encoder (for widgets and BeginView).
func (c *Context) Encoder() *cmd.Encoder { return &c.enc }

// MarkHover marks that the pointer is over a UI surface this frame.
func (c *Context) MarkHover() {
	if c != nil {
		c.wantsMouse = true
	}
}

// SetActive sets the active widget (e.g. dragging window).
func (c *Context) SetActive(id WidgetID) {
	if c != nil {
		c.activeID = id
	}
}

// ActiveID returns the active widget id.
func (c *Context) ActiveID() WidgetID {
	if c == nil {
		return 0
	}
	return c.activeID
}

// ClearActive clears active widget if it matches id.
func (c *Context) ClearActive(id WidgetID) {
	if c != nil && c.activeID == id {
		c.activeID = 0
	}
}

// WantsMouse reports whether UI consumed pointing-device focus.
func (c *Context) WantsMouse() bool { return c != nil && c.wantsMouse }

// WantsKeyboard reports whether UI wants key focus.
func (c *Context) WantsKeyboard() bool { return c != nil && c.wantsKb }

// WantsTextInput reports whether a text-editing widget has focus.
func (c *Context) WantsTextInput() bool { return c != nil && c.wantsText }

// SetFocus sets the sticky focus id (used for text input widgets).
func (c *Context) SetFocus(id WidgetID) {
	if c == nil {
		return
	}
	c.focusID = id
	c.wantsText = id != 0
}

// ClearFocus clears focus if it matches id.
func (c *Context) ClearFocus(id WidgetID) {
	if c == nil {
		return
	}
	if c.focusID == id {
		c.focusID = 0
		c.wantsText = false
	}
}

// IsFocused reports whether id holds focus.
func (c *Context) IsFocused(id WidgetID) bool {
	return c != nil && c.focusID == id && id != 0
}

// FocusID returns the currently focused widget id (0 if none).
func (c *Context) FocusID() WidgetID {
	if c == nil {
		return 0
	}
	return c.focusID
}

// RegisterFocusable records rect as a focusable widget this frame (used by End() for click-outside).
func (c *Context) RegisterFocusable(r emath.Rect) {
	if c == nil {
		return
	}
	c.focusableRects = append(c.focusableRects, r)
}

// Close releases CPU-side context state and the shared white texture.
// Font is owned by the caller and must be closed separately.
//
// Drops anything Context created on widgets' behalf BEFORE the white texture,
// so a focused text-input doesn't outlive Close with a stranded DOM overlay
// or a leaked volatile line texture (see ai/text-review.md M2/M3).
func (c *Context) Close() {
	if c == nil {
		return
	}
	b := c.backend
	if c.textInputHost.owner != 0 {
		b.TextInputEnd()
		c.textInputHost.owner = 0
	}
	if c.focusID != 0 && c.font != nil {
		c.font.DropVolatile(uint32(c.focusID))
	}
	if c.white.image >= 0 {
		b.ImageDestroy(c.white.image)
		c.white.image = -1
	}
	if c.white.sampler >= 0 {
		b.SamplerDestroy(c.white.sampler)
		c.white.sampler = -1
	}
	c.states = nil
}

// ContentRect is the inner layout rectangle of the current container (or full
// viewport). For auto-sized containers H may be MaxInt32; widgets that need a
// drawable height should call Avail or LayoutNextRow instead.
func (c *Context) ContentRect() emath.Rect {
	if c == nil {
		return emath.Rect{}
	}
	if len(c.layoutStack) == 0 {
		return c.viewport
	}
	return c.layoutStack[len(c.layoutStack)-1].rect
}

// Avail returns the remaining slot inside the current container — from the
// pen down to the container's bottom edge, full container width. This is the
// ImGui `GetContentRegionAvail` analogue.
func (c *Context) Avail() emath.Rect {
	if c == nil || len(c.layoutStack) == 0 {
		return c.viewport
	}
	f := &c.layoutStack[len(c.layoutStack)-1]
	h := f.rect.Y + f.rect.H - f.cursorY
	if h < 0 {
		h = 0
	}
	return emath.Rect{X: f.rect.X, Y: f.cursorY, W: f.rect.W, H: h}
}

// LayoutNextRow reserves the next vertical slot of height h (full container
// width) and advances the pen by h. Subsequent rows automatically get a
// theme.Spacing gap inserted before their Y. Widgets call this once at the
// top of their build to claim space; the pen then drives auto-sized windows
// and stacked layout.
//
// h <= 0 still reserves a row (zero height) and counts toward the spacing
// gap on the next call (matches ImGui's `ItemSize` semantics).
func (c *Context) LayoutNextRow(h int32) emath.Rect {
	if c == nil {
		return emath.Rect{}
	}
	if len(c.layoutStack) == 0 {
		// No active container: behave like the bare viewport, no pen.
		return c.viewport
	}
	f := &c.layoutStack[len(c.layoutStack)-1]
	if f.cursorY > f.startY {
		f.cursorY += f.spacing
	}
	row := emath.Rect{X: f.rect.X, Y: f.cursorY, W: f.rect.W, H: h}
	if h < 0 {
		h = 0
	}
	right := row.X + row.W
	if right > f.maxX {
		f.maxX = right
	}
	f.cursorY += h
	return row
}

// pushLayout starts a new container frame. Caller fills rect + spacing; this
// initializes the pen and used-extent trackers from rect's top-left.
func (c *Context) pushLayout(f layoutFrame) {
	if c == nil {
		return
	}
	f.startY = f.rect.Y
	f.cursorY = f.rect.Y
	f.maxX = f.rect.X
	c.layoutStack = append(c.layoutStack, f)
}

func (c *Context) popLayout() (layoutFrame, bool) {
	if c == nil || len(c.layoutStack) == 0 {
		return layoutFrame{}, false
	}
	f := c.layoutStack[len(c.layoutStack)-1]
	c.layoutStack = c.layoutStack[:len(c.layoutStack)-1]
	return f, true
}
