package ui

import (
	"unicode/utf8"

	"triggle/engine/emath"
	"triggle/engine/text"
	"triggle/engine/ui/cmd"
	"triggle/engine/ui/theme"
)

// TextInputOpt configures TextInput.
// Initial seeds the buffer on the first frame only (IM semantics); callers
// can force a re-seed by using a different label (different WidgetID).
type TextInputOpt struct {
	Initial  string
	MaxBytes int
}

// Widget padding inside the input rect. Promote to theme.Theme when a second
// widget needs the same knobs.
const (
	textInputPadX int32 = 4
	textInputPadY int32 = 3
)

// Caret blink timing (seconds). Pause forces the caret visible right after
// any edit / movement, matching macOS / Windows / GTK / ImGui. The half-period
// is the on (or off) duration; full cycle = 2 × halfPeriod.
const (
	caretIdlePause      float32 = 0.5
	caretBlinkHalfPhase float32 = 0.5
)

// caretVisible decides whether to draw the caret this frame.
//   - Within caretIdlePause of the last edit: always on (lets the eye track
//     the caret while typing/moving).
//   - After the pause: square wave with caretBlinkHalfPhase on / off.
func caretVisible(clock, lastEditAt float32) bool {
	idle := clock - lastEditAt
	if idle < caretIdlePause {
		return true
	}
	phase := idle - caretIdlePause
	return int(phase/caretBlinkHalfPhase)%2 == 0
}

// TextInputHeight returns the outer pixel height of one TextInput row at
// pxSize: line box (ascent+descent) plus vertical padding. Width is set by
// the parent ContentRect, so only height is exposed.
//
//	winH = theme.TitleHeight + theme.Padding.Top +
//	       ui.TextInputHeight(font, px) + theme.Padding.Bottom
func TextInputHeight(font *text.Font, pxSize int32) int32 {
	if font == nil {
		return pxSize + textInputPadY*2
	}
	m := font.Metrics(pxSize)
	h := m.Ascent + m.Descent
	if h <= 0 {
		h = pxSize
	}
	return h + textInputPadY*2
}

// textInputState is the per-widget persistent state.
//
// Buffer source of truth: native edits it directly from KeyEvents+Text;
// WASM gets it refreshed from consumeTextInputPoll before each draw (DOM
// owns it while focused). No per-widget host-attach flag — Context
// reconciles via the host bookkeeping below.
type textInputState struct {
	buf        []byte
	caret      int
	init       bool
	wasFocused bool

	// Caret blink: blinkClock advances by dt while focused; lastEditAt is
	// the clock value at the most recent caret/buffer change. Visible when
	// (blinkClock - lastEditAt) < caretIdlePause OR when the post-pause
	// phase is in its on half. Tracked separately from `wasFocused` so
	// re-focusing immediately shows the caret. lastCaret/lastLen are the
	// previous frame's edit signature for the change detector — len(buf)
	// covers any insert/delete; caret covers movement (arrow keys, Home/
	// End, future mouse-driven caret).
	blinkClock float32
	lastEditAt float32
	lastCaret  int
	lastLen    int

	// scrollX is the horizontal pixel offset applied to the rendered text so
	// the caret stays inside the visible content area. Persists across
	// frames so the view doesn't snap back to 0.
	scrollX int32

	// caretW caches the last measured caret-prefix width; (caretCachedAt,
	// caretCachedLen) is the (caret, bufLen) pair it was measured at.
	// Skips the per-blink-frame string([]byte) alloc + strcmp in steady
	// state. Including bufLen invalidates the cache when the buffer mutates
	// left of the caret without moving it (rare, but cheap).
	caretW         int32
	caretCachedAt  int
	caretCachedLen int
}

// textInputHostState centralizes the host-owned text-input lifecycle.
//
// On WASM the host is a hidden <input> overlay (browser owns IME, selection,
// clipboard). On native sokol the Backend.TextInput* shims are no-ops and
// the widget's own buffer is the source of truth — pollOwner stays 0 so
// consume returns nothing, and reconcile's Begin/End calls hit empty stubs.
// SDL3 native (future) will wire the same shims to SDL_StartTextInput +
// SDL_SetTextInputArea, no engine/ui changes required.
//
// Ownership model (avoids the per-widget attach-blur ordering bug):
//   - Begin: if owner != 0, poll the host into pollBuf/pollCaret, mark pollOwner.
//   - Widget: if its id matches pollOwner, consume (copy into its state).
//   - Widget: if focused, publish its rect+buf+caret as the frame's host request.
//   - End: reconcile req vs owner — same owner is a no-op, different owner
//     re-activates the session, no request with nonzero owner deactivates.
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

// beginTextInputPoll pulls the current host value + caret into
// textInputHost.poll* so the focused widget can consume the snapshot during
// its draw. WASM: browser owns the buffer while focused, Go reads what it
// last committed. Native: Backend.TextInputPoll returns -1 (no host buffer)
// and pollOwner stays 0.
func (c *Context) beginTextInputPoll() {
	if c == nil || c.textInputHost.owner == 0 {
		c.textInputHost.pollOwner = 0
		return
	}
	if cap(c.textInputHost.pollBuf) < 512 {
		c.textInputHost.pollBuf = make([]byte, 512)
	} else {
		c.textInputHost.pollBuf = c.textInputHost.pollBuf[:512]
	}
	var caret int32
	n := c.backend.TextInputPoll(c.textInputHost.pollBuf, &caret)
	if n < 0 {
		c.textInputHost.pollOwner = 0
		return
	}
	if int(n) > len(c.textInputHost.pollBuf) {
		big := make([]byte, n)
		n = c.backend.TextInputPoll(big, &caret)
		if n < 0 {
			c.textInputHost.pollOwner = 0
			return
		}
		c.textInputHost.pollBuf = big[:n]
	} else {
		c.textInputHost.pollBuf = c.textInputHost.pollBuf[:n]
	}
	c.textInputHost.pollCaret = caret
	c.textInputHost.pollOwner = c.textInputHost.owner
}

// endTextInputReconcile drives the one-site host lifecycle call, run from
// Context.End after widgets have (or haven't) published their request.
//
//	reqOwner == 0       → focused widget gone this frame → end session.
//	reqOwner != owner   → (re)begin session, re-seed the shared element.
//	reqOwner == owner   → steady state; element stays put.
func (c *Context) endTextInputReconcile() {
	if c == nil {
		return
	}
	if c.textInputHost.reqOwner == 0 {
		if c.textInputHost.owner != 0 {
			c.backend.TextInputEnd()
			c.textInputHost.owner = 0
		}
		return
	}
	if c.textInputHost.owner != c.textInputHost.reqOwner {
		r := c.textInputHost.reqRect
		c.backend.TextInputBegin(r.X, r.Y, r.W, r.H, c.textInputHost.reqBuf, c.textInputHost.reqCaret)
		c.textInputHost.owner = c.textInputHost.reqOwner
	}
	c.textInputHost.reqOwner = 0
}

// consumeTextInputPoll returns the host poll snapshot taken at Begin, if it
// was for this widget. Second return is the caret byte offset. ok=false on
// native (Poll returns -1, polling is disabled).
func (c *Context) consumeTextInputPoll(id WidgetID) (buf []byte, caret int32, ok bool) {
	if c == nil || c.textInputHost.pollOwner == 0 || c.textInputHost.pollOwner != id {
		return nil, 0, false
	}
	c.textInputHost.pollOwner = 0
	return c.textInputHost.pollBuf, c.textInputHost.pollCaret, true
}

// publishTextInput records the frame's host request from the focused widget.
// Last writer wins — a same-frame focus handoff naturally overwrites the
// previous request so the reconciler in End drives a single Begin.
func (c *Context) publishTextInput(id WidgetID, rect emath.Rect, buf []byte, caret int) {
	if c == nil || id == 0 {
		return
	}
	c.textInputHost.reqOwner = id
	c.textInputHost.reqRect = rect
	c.textInputHost.reqBuf = append(c.textInputHost.reqBuf[:0], buf...)
	c.textInputHost.reqCaret = int32(caret)
}

// editTextInput is the microui-style rune-aware editor: drains KeyEvents +
// committed Text from the InputFrame into the widget's UTF-8 buffer.
//
// Cross-platform but caller-gated: on WASM, sokol is configured with
// html5_bubble_{key,char}_events=true (see backend/src/main.cpp), so the
// hidden <input> consumes keys for IME purposes WITHOUT preventing the same
// events from also reaching the canvas. The widget therefore must NOT call
// this function on a frame where consumeTextInputPoll returned the host's
// authoritative snapshot — otherwise every key applies twice (once in the
// DOM input that we just read, then again here) and the next frame's poll
// snaps the buffer back, producing visible jitter. TextInput gates the
// call accordingly.
func editTextInput(ws *textInputState, in *InputFrame, opt TextInputOpt, focused bool) {
	if !focused {
		return
	}
	for _, ev := range in.KeyEvents {
		if !ev.Down {
			continue
		}
		switch ev.Key {
		case KeyBackspace:
			if ws.caret > 0 {
				_, sz := utf8.DecodeLastRune(ws.buf[:ws.caret])
				ws.buf = append(ws.buf[:ws.caret-sz], ws.buf[ws.caret:]...)
				ws.caret -= sz
			}
		case KeyDelete:
			if ws.caret < len(ws.buf) {
				_, sz := utf8.DecodeRune(ws.buf[ws.caret:])
				ws.buf = append(ws.buf[:ws.caret], ws.buf[ws.caret+sz:]...)
			}
		case KeyLeft:
			if ws.caret > 0 {
				_, sz := utf8.DecodeLastRune(ws.buf[:ws.caret])
				ws.caret -= sz
			}
		case KeyRight:
			if ws.caret < len(ws.buf) {
				_, sz := utf8.DecodeRune(ws.buf[ws.caret:])
				ws.caret += sz
			}
		case KeyHome:
			ws.caret = 0
		case KeyEnd:
			ws.caret = len(ws.buf)
		}
	}
	if in.Text != "" {
		if opt.MaxBytes > 0 && len(ws.buf)+len(in.Text) > opt.MaxBytes {
			return
		}
		t := []byte(in.Text)
		ws.buf = append(ws.buf[:ws.caret], append(t, ws.buf[ws.caret:]...)...)
		ws.caret += len(t)
	}
}

// TextInput draws a single-line editable text field at the top of the
// current content rect. Returns the current value and whether Enter was
// pressed this frame. On WASM the focused frame consumes the host poll
// snapshot and skips the local editor (sokol still bubbles keys to the
// canvas, so running both would double-apply).
func (c *Context) TextInput(label string, opt TextInputOpt) (value string, submitted bool) {
	if c == nil || c.font == nil {
		return "", false
	}
	c.idStack = append(c.idStack, "input:"+label)
	id := c.hashID()
	defer func() { c.idStack = c.idStack[:len(c.idStack)-1] }()

	ws := StateOf[textInputState](c, id)
	if !ws.init {
		ws.buf = append(ws.buf[:0], opt.Initial...)
		ws.caret = len(ws.buf)
		ws.lastCaret = ws.caret
		ws.lastLen = len(ws.buf)
		ws.init = true
	}

	// Size to the font's actual line box (ascent+descent), not BodyPx — the
	// rendered bitmap reaches descent below the baseline, and BodyPx alone
	// clips descenders ('g', 'p', 'y') in the parent window's clip rect.
	// Pixi-style: no extra leading multiplier; widget padding is the leading.
	// TextInputHeight is the public mirror of this math.
	rowH := TextInputHeight(c.font, c.theme.BodyPx)
	lineH := rowH - textInputPadY*2
	r := c.LayoutNextRow(rowH)
	c.RegisterFocusable(r)

	mx, my := int32(c.in.MousePos[0]), int32(c.in.MousePos[1])
	if r.Contains(mx, my) {
		c.MarkHover()
		if c.in.MousePressed&MouseLeft != 0 {
			c.SetFocus(id)
		}
	}

	focused := c.IsFocused(id)

	// WASM: if Context polled the host for us at Begin, adopt the snapshot
	// and skip the local editor — sokol bubbles all key/char events to the
	// canvas (html5_bubble_*=true), so editing keys ALSO reach in.KeyEvents
	// and in.Text. Letting editTextInput run would double-apply each
	// keystroke against the buffer the host already updated.
	hostOwned := false
	if buf, caret, ok := c.consumeTextInputPoll(id); ok {
		ws.buf = append(ws.buf[:0], buf...)
		if opt.MaxBytes > 0 && len(ws.buf) > opt.MaxBytes {
			ws.buf = ws.buf[:opt.MaxBytes]
		}
		if int(caret) > len(ws.buf) {
			caret = int32(len(ws.buf))
		}
		if caret < 0 {
			caret = 0
		}
		ws.caret = int(caret)
		hostOwned = true
	}

	if !hostOwned {
		editTextInput(ws, &c.in, opt, focused)
	}

	// Publish the frame's host-text-input request; Context.End reconciles
	// against its current owner. WASM drives the DOM overlay; native is no-op.
	if focused {
		c.publishTextInput(id, r, ws.buf, ws.caret)

		// Caret blink bookkeeping. Reset the clock when focus is just
		// gained so the caret shows immediately + the idle pause covers
		// the focus moment. Mark "edited" when the (caret, bufLen)
		// signature moved since last frame — cheap, false-negative only
		// when host-poll round-trips an identical signature.
		if !ws.wasFocused {
			ws.blinkClock = 0
			ws.lastEditAt = 0
		}
		ws.blinkClock += c.dt
		if ws.caret != ws.lastCaret || len(ws.buf) != ws.lastLen {
			ws.lastEditAt = ws.blinkClock
		}
		ws.lastCaret = ws.caret
		ws.lastLen = len(ws.buf)

		for _, ev := range c.in.KeyEvents {
			if !ev.Down {
				continue
			}
			if ev.Key == KeyEnter {
				submitted = true
			} else if ev.Key == KeyEscape {
				c.ClearFocus(id)
				focused = false
			}
		}
	}

	bgCol := c.theme.Colors[theme.ColorBase]
	if bgCol.A == 0 {
		bgCol = cmd.Color{R: 22, G: 22, B: 28, A: 255}
	}
	if focused {
		bgCol = cmd.Color{R: 36, G: 38, B: 54, A: 255}
	}
	c.enc.QuadSolid(r, bgCol)

	col := c.theme.Colors[theme.ColorText]
	tc := text.Color{R: col.R, G: col.G, B: col.B, A: col.A}
	textY := r.Y + textInputPadY
	contentW := r.W - textInputPadX*2
	if contentW < 0 {
		contentW = 0
	}

	// Caret prefix width drives both caret rendering and scroll adjustment;
	// measure once per frame and reuse. Cache by (caret, bufLen) so steady
	// blink doesn't re-shape every frame.
	caret := ws.caret
	if caret > len(ws.buf) {
		caret = len(ws.buf)
	}
	if caret > 0 {
		if caret != ws.caretCachedAt || len(ws.buf) != ws.caretCachedLen {
			ws.caretCachedAt = caret
			ws.caretCachedLen = len(ws.buf)
			ws.caretW = int32(c.font.Measure(string(ws.buf[:caret]), c.theme.BodyPx)[0])
		}
	} else {
		ws.caretW = 0
		ws.caretCachedAt = 0
		ws.caretCachedLen = len(ws.buf)
	}

	// Horizontal scroll: keep caret inside [0, contentW]. Simplest
	// single-line behavior — no margin, no easing. Also clamp scrollX so
	// we never scroll past the start, and so the buffer right-aligns to
	// the rect when the total width fits inside contentW after a delete.
	totalW := int32(c.font.Measure(string(ws.buf), c.theme.BodyPx)[0])
	if ws.caretW-ws.scrollX < 0 {
		ws.scrollX = ws.caretW
	} else if ws.caretW-ws.scrollX > contentW {
		ws.scrollX = ws.caretW - contentW
	}
	maxScroll := totalW - contentW
	if maxScroll < 0 {
		maxScroll = 0
	}
	if ws.scrollX > maxScroll {
		ws.scrollX = maxScroll
	}
	if ws.scrollX < 0 {
		ws.scrollX = 0
	}

	// Clip text + caret to the input's content area so scrolled text doesn't
	// bleed out of the rect into the surrounding window padding.
	clipR := emath.Rect{X: r.X + textInputPadX, Y: r.Y, W: contentW, H: r.H}
	c.enc.PushClip(clipR)

	textX := r.X + textInputPadX - ws.scrollX
	if focused {
		c.font.DrawVolatile(&c.enc, uint32(id), string(ws.buf), textX, textY, c.theme.BodyPx, tc)
	} else {
		c.font.Draw(&c.enc, string(ws.buf), textX, textY, c.theme.BodyPx, tc)
	}

	if focused && caretVisible(ws.blinkClock, ws.lastEditAt) {
		caretX := r.X + textInputPadX + ws.caretW - ws.scrollX
		c.enc.QuadSolid(emath.Rect{X: caretX, Y: textY, W: 1, H: lineH}, col)
	}

	c.enc.PopClip()

	if ws.wasFocused && !focused {
		c.font.DropVolatile(uint32(id))
	}
	ws.wasFocused = focused

	return string(ws.buf), submitted
}
