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
	wantsMouse bool
	wantsKb    bool
	wantsText  bool

	layoutStack []emath.Rect
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
	c.enc.Reset(viewport)
	c.wantsMouse = false
	c.wantsKb = false
	c.wantsText = false
}

// End finalizes the frame.
func (c *Context) End() {
	if c == nil {
		return
	}
	if c.activeID != 0 {
		c.wantsMouse = true
	}
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

// Close releases CPU-side context state and the shared white texture.
// Font is owned by the caller and must be closed separately.
func (c *Context) Close() {
	if c == nil {
		return
	}
	b := c.backend
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

// ContentRect is the inner layout rectangle of the current container (or full viewport).
func (c *Context) ContentRect() emath.Rect {
	if c == nil {
		return emath.Rect{}
	}
	if len(c.layoutStack) == 0 {
		return c.viewport
	}
	return c.layoutStack[len(c.layoutStack)-1]
}

func (c *Context) pushLayout(r emath.Rect) {
	if c == nil {
		return
	}
	c.layoutStack = append(c.layoutStack, r)
}

func (c *Context) popLayout() {
	if c == nil || len(c.layoutStack) == 0 {
		return
	}
	c.layoutStack = c.layoutStack[:len(c.layoutStack)-1]
}
