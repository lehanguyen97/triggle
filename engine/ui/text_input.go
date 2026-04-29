package ui

import (
	"unicode/utf8"

	"triggle/engine/emath"
	"triggle/engine/text"
)

const (
	textInputPadX float32 = 4
	textInputPadY float32 = 3
)

const (
	caretIdlePause      float32 = 0.5
	caretBlinkHalfPhase float32 = 0.5
)

func caretVisible(clock, lastEditAt float32) bool {
	idle := clock - lastEditAt
	if idle < caretIdlePause {
		return true
	}
	phase := idle - caretIdlePause
	return int(phase/caretBlinkHalfPhase)%2 == 0
}

// TextInputHeight is the outer logical-pixel height of one row.
// Converts the font's physical-px metrics to lp using scale.
func TextInputHeight(font *text.Font, pxSize int32, scale float32) float32 {
	if scale <= 0 {
		scale = 1
	}
	lpSize := float32(pxSize) / scale
	if font == nil {
		return lpSize + textInputPadY*2
	}
	m := font.Metrics(pxSize)
	asc := m.Ascent / scale
	dsc := m.Descent / scale
	h := asc + dsc
	if h <= 0 {
		h = lpSize
	}
	return h + textInputPadY*2
}

// TextInput is a single-line text field.
type TextInput struct {
	BaseNode

	Value    string
	Binding  StringBinding
	MaxBytes int
	OnChange func(string)
	OnSubmit func(string)
	Style    TextStyle

	// Internal editing state; Value tracks buf after each Event.
	inited             bool
	buf                []byte
	caret              int
	wasFocus           bool
	lastPaintFocus     bool
	blinkClock         float32
	lastEditAt         float32
	lastCaret, lastLen int
	scrollX            float32
	caretW             float32
	caretCachedAt      int
	caretCachedLen     int
}

type StringBinding struct {
	Get func() string
	Set func(string)
}

// SetValue replaces the text and reseeds the caret; does not run OnChange.
func (t *TextInput) SetValue(s string) {
	t.inited = true
	t.Value = s
	t.buf = append(t.buf[:0], s...)
	t.caret = len(t.buf)
	t.lastCaret = t.caret
	t.lastLen = len(t.buf)
	t.Invalidate()
}

// RequestFocus is a convenience; queues focus in App.
func (t *TextInput) RequestFocus() {
	if a := t.App(); a != nil {
		a.RequestTextFocus(t)
	}
}

// Children implements Node.
func (t *TextInput) Children() []Node { return nil }

// Measure implements Node.
func (t *TextInput) Measure(c Constraints) Size {
	_ = c
	if t.app == nil {
		return Size{}
	}
	t.ensureInit()
	cn := normConstraints(c)
	style := resolveRootTextStyle(t.app, t.Style)
	if style.Font == nil {
		return Size{}
	}
	h := TextInputHeight(style.Font, style.SizePx, t.app.vpState.UIScale)
	return Size{W: cn.MaxW, H: h}
}

// Place implements Node.
func (t *TextInput) Place(outer emath.Rect) {
	t.BaseNode.rect = outer
}

// Event implements Node (only meaningful when App routes focus here).
func (t *TextInput) Event(_ *Event, ec *EventCtx) bool {
	if t.app == nil || t.app.Font() == nil {
		return false
	}
	t.ensureInit()
	af := t.app
	id := t.WidgetID()
	focused := af.focus == t

	// Host poll (WASM)
	hostOwned := false
	if buf, caret, ok := af.consumeTextInputPoll(id); ok {
		t.buf = append(t.buf[:0], buf...)
		if t.MaxBytes > 0 && len(t.buf) > t.MaxBytes {
			t.buf = t.buf[:t.MaxBytes]
		}
		if int(caret) > len(t.buf) {
			caret = int32(len(t.buf))
		}
		if caret < 0 {
			caret = 0
		}
		t.caret = int(caret)
		hostOwned = true
	}

	if !hostOwned && focused {
		editTextInput(t, &ec.Frame, t.MaxBytes)
	}

	// Caret / blink; Enter / Escape
	if focused {
		af.publishTextInput(id, t.rect, t.buf, t.caret)
		if !t.wasFocus {
			t.blinkClock = 0
			t.lastEditAt = 0
		}
		t.blinkClock += ec.DT
		if t.caret != t.lastCaret || len(t.buf) != t.lastLen {
			t.lastEditAt = t.blinkClock
		}
		t.lastCaret = t.caret
		t.lastLen = len(t.buf)
		t.fireSubmitOnEnter(&ec.Frame)
		for _, ev := range ec.Frame.KeyEvents {
			if ev.Down && ev.Key == KeyEscape {
				ec.App.setFocusNode(nil)
			}
		}
	} else {
		t.blinkClock = 0
	}

	t.applyEdits()
	t.wasFocus = focused
	return true
}

// applyEdits propagates the current buffer to Value, then notifies Binding.Set
// and OnChange when the value actually changed.
func (t *TextInput) applyEdits() {
	prev := t.Value
	t.Value = string(t.buf)
	if t.Binding.Set != nil && t.Value != prev {
		t.Binding.Set(t.Value)
	}
	if t.Value != prev && t.OnChange != nil {
		t.OnChange(t.Value)
	}
}

// fireSubmitOnEnter calls OnSubmit once per Enter keydown in frame, with the
// current pre-applyEdits buffer. Caller is responsible for any post-submit
// buffer reset (see ChatBox.handleSubmit).
func (t *TextInput) fireSubmitOnEnter(frame *InputFrame) {
	if t.OnSubmit == nil || frame == nil {
		return
	}
	for _, ev := range frame.KeyEvents {
		if ev.Down && ev.Key == KeyEnter {
			t.OnSubmit(string(t.buf))
		}
	}
}

func (t *TextInput) ensureInit() {
	if t.Binding.Get != nil {
		v := t.Binding.Get()
		if !t.inited || v != t.Value {
			t.Value = v
			t.buf = append(t.buf[:0], v...)
			if t.caret > len(t.buf) {
				t.caret = len(t.buf)
			}
		}
	}
	if t.inited {
		return
	}
	t.buf = append(t.buf[:0], t.Value...)
	t.caret = len(t.buf)
	t.lastCaret = t.caret
	t.lastLen = len(t.buf)
	t.inited = true
}

// Paint implements Node.
func (t *TextInput) Paint(pc *PaintCtx) {
	if t.app == nil {
		return
	}
	t.ensureInit()
	af := t.app
	r := t.rect
	style := resolveRootTextStyle(t.app, t.Style)
	if style.Font == nil {
		return
	}
	px := style.SizePx
	lineH := TextInputHeight(style.Font, px, t.app.vpState.UIScale) - textInputPadY*2
	foc := af.focus == t
	prevPaint := t.lastPaintFocus
	if prevPaint && !foc {
		t.app.DropVolatileText(text.OwnerID(t.WidgetID()))
	}
	t.lastPaintFocus = foc

	bgCol := t.app.theme.Colors[ColorBase]
	if bgCol.A == 0 {
		bgCol = emath.Color{R: 22, G: 22, B: 28, A: 255}
	}
	if foc {
		bgCol = emath.Color{R: 36, G: 38, B: 54, A: 255}
	}
	pc.Rect(r, bgCol)

	col := style.Color
	textY := r.Y + textInputPadY
	contentW := r.W - textInputPadX*2
	if contentW < 0 {
		contentW = 0
	}

	caret := t.caret
	if caret > len(t.buf) {
		caret = len(t.buf)
	}
	if caret > 0 {
		if caret != t.caretCachedAt || len(t.buf) != t.caretCachedLen {
			t.caretCachedAt = caret
			t.caretCachedLen = len(t.buf)
			t.caretW = t.app.MeasureText(string(t.buf[:caret]), t.Style).Width
		}
	} else {
		t.caretW = 0
		t.caretCachedAt = 0
		t.caretCachedLen = len(t.buf)
	}

	totalW := t.app.MeasureText(string(t.buf), t.Style).Width
	if t.caretW-t.scrollX < 0 {
		t.scrollX = t.caretW
	} else if t.caretW-t.scrollX > contentW {
		t.scrollX = t.caretW - contentW
	}
	maxScroll := totalW - contentW
	if maxScroll < 0 {
		maxScroll = 0
	}
	if t.scrollX > maxScroll {
		t.scrollX = maxScroll
	}
	if t.scrollX < 0 {
		t.scrollX = 0
	}

	clipR := emath.Rect{X: r.X + textInputPadX, Y: r.Y, W: contentW, H: r.H}
	pc.PushClip(clipR)
	textX := r.X + textInputPadX - t.scrollX
	if foc {
		pc.VolatileText(text.OwnerID(t.WidgetID()), string(t.buf), t.Style, textX, textY)
	} else {
		pc.Text(string(t.buf), t.Style, textX, textY)
	}
	if foc && caretVisible(t.blinkClock, t.lastEditAt) {
		caretX := r.X + textInputPadX + t.caretW - t.scrollX
		quad := col
		pc.Rect(emath.Rect{X: caretX, Y: textY, W: 1, H: lineH}, quad)
	}
	pc.PopClip()

}

func editTextInput(t *TextInput, in *InputFrame, maxBytes int) {
	for _, ev := range in.KeyEvents {
		if !ev.Down {
			continue
		}
		switch ev.Key {
		case KeyBackspace:
			if t.caret > 0 {
				_, sz := utf8.DecodeLastRune(t.buf[:t.caret])
				t.buf = append(t.buf[:t.caret-sz], t.buf[t.caret:]...)
				t.caret -= sz
			}
		case KeyDelete:
			if t.caret < len(t.buf) {
				_, sz := utf8.DecodeRune(t.buf[t.caret:])
				t.buf = append(t.buf[:t.caret], t.buf[t.caret+sz:]...)
			}
		case KeyLeft:
			if t.caret > 0 {
				_, sz := utf8.DecodeLastRune(t.buf[:t.caret])
				t.caret -= sz
			}
		case KeyRight:
			if t.caret < len(t.buf) {
				_, sz := utf8.DecodeRune(t.buf[t.caret:])
				t.caret += sz
			}
		case KeyHome:
			t.caret = 0
		case KeyEnd:
			t.caret = len(t.buf)
		}
	}
	if in.Text != "" {
		if maxBytes > 0 && len(t.buf)+len(in.Text) > maxBytes {
			return
		}
		tb := []byte(in.Text)
		t.buf = append(append(t.buf[:t.caret], tb...), t.buf[t.caret:]...)
		t.caret += len(tb)
	}
}
