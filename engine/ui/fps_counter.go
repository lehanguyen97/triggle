package ui

import (
	"fmt"
	"slices"

	"triggle/engine/emath"
	"triggle/engine/text"
)

// FPSCounter displays rolling-window average FPS and p95 frame time. Samples
// are fed by Overlay.Tick via TickNode(dt); no caller wiring beyond placement.
type FPSCounter struct {
	BaseNode
	Window int         // ring size in samples; default 240
	Color  emath.Color // zero = theme text color

	samples []float32 // seconds per frame
	idx     int
	filled  bool
	sortBuf []float32
	text    string
}

// fpsTemplate is a fixed-width reference used by Measure so layout bounds don't
// jitter as the formatted numbers change.
const fpsTemplate = "FPS 999.9 | p95 99.9ms"

func (f *FPSCounter) ensureBuf() {
	w := f.Window
	if w < 1 {
		w = 240
	}
	if len(f.samples) != w {
		f.samples = make([]float32, w)
		f.sortBuf = make([]float32, w)
		f.idx = 0
		f.filled = false
	}
}

// TickNode records dt into the ring and refreshes the formatted label.
func (f *FPSCounter) TickNode(dt float32) {
	f.ensureBuf()
	f.samples[f.idx] = dt
	f.idx++
	if f.idx >= len(f.samples) {
		f.idx = 0
		f.filled = true
	}
	n := f.idx
	if f.filled {
		n = len(f.samples)
	}
	if n == 0 {
		f.text = "FPS -- | p95 --"
		return
	}
	copy(f.sortBuf[:n], f.samples[:n])
	buf := f.sortBuf[:n]
	slices.Sort(buf)
	var sum float32
	for _, v := range buf {
		sum += v
	}
	avg := sum / float32(n)
	var fps float32
	if avg > 0 {
		fps = 1.0 / avg
	}
	pIdx := int(float32(n) * 0.95)
	if pIdx >= n {
		pIdx = n - 1
	}
	p95ms := buf[pIdx] * 1000.0
	f.text = fmt.Sprintf("FPS %.1f | p95 %.1fms", fps, p95ms)
}

func (f *FPSCounter) Children() []Node { return nil }

func (f *FPSCounter) Measure(c Constraints) Size {
	if f.app == nil {
		return Size{}
	}
	_ = normConstraints(c)
	style := resolveRootTextStyle(f.app, TextStyle{Color: f.Color})
	if style.Font == nil {
		return Size{}
	}
	scale := f.app.vpState.UIScale
	if scale <= 0 {
		scale = 1
	}
	m := style.Font.Metrics(style.SizePx)
	asc := m.Ascent / scale
	dsc := m.Descent / scale
	h := asc + dsc
	if h <= 0 {
		h = style.SizeLp
	}
	tm := f.app.MeasureText(fpsTemplate, TextStyle{Color: f.Color})
	return Size{W: tm.Width, H: h}
}

func (f *FPSCounter) Place(outer emath.Rect) { f.BaseNode.rect = outer }

func (f *FPSCounter) Paint(pc *PaintCtx) {
	if f.app == nil || f.text == "" {
		return
	}
	style := resolveRootTextStyle(f.app, TextStyle{Color: f.Color})
	if style.Font == nil {
		return
	}
	pc.VolatileText(text.OwnerID(f.WidgetID()), f.text, TextStyle{Color: f.Color}, f.rect.X, f.rect.Y)
}

func (f *FPSCounter) Event(_ *Event, _ *EventCtx) bool { return false }
