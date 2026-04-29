package ui

import "triggle/engine/text"

// MeasureText returns the lp size of s after merging style with the active
// theme + viewport scale. Returns zero metrics if no font is registered for
// the resolved style. The text engine works in physical px; we feed it
// SizeLp*UIScale and convert returned metrics back to lp.
func (r *Root) MeasureText(s string, style TextStyle) text.Metrics {
	resolved := resolveRootTextStyle(r, style)
	if resolved.Font == nil || s == "" {
		return text.Metrics{}
	}
	m := resolved.Font.MeasureLine(s, text.Options{SizePx: resolved.SizePx})
	return scaleMetricsToLp(m, r.vpState.UIScale)
}

func (r *Root) DropVolatileText(owner text.OwnerID) {
	if r == nil || r.fonts == nil || owner == 0 {
		return
	}
	r.fonts.DropVolatile(owner)
}

func (pc *PaintCtx) Text(s string, style TextStyle, x, y float32) {
	if pc == nil || pc.Context == nil {
		return
	}
	resolved := resolveRootTextStyle(pc.Root, style)
	if resolved.Font == nil || s == "" {
		return
	}
	line := resolved.Font.Line(s, text.Options{SizePx: resolved.SizePx})
	pc.Context.TextLineScaled(line, x, y, scaleInv(pc.Root), resolved.Color)
}

func (pc *PaintCtx) VolatileText(owner text.OwnerID, s string, style TextStyle, x, y float32) {
	if pc == nil || pc.Context == nil || owner == 0 {
		return
	}
	resolved := resolveRootTextStyle(pc.Root, style)
	if resolved.Font == nil || s == "" {
		return
	}
	line := resolved.Font.VolatileLine(owner, s, text.Options{SizePx: resolved.SizePx})
	pc.Context.TextLineScaled(line, x, y, scaleInv(pc.Root), resolved.Color)
}

// scaleInv returns 1/UIScale for converting fb-px text segment offsets to lp.
func scaleInv(r *Root) float32 {
	if r == nil || r.vpState.UIScale <= 0 {
		return 1
	}
	return 1.0 / r.vpState.UIScale
}

func scaleMetricsToLp(m text.Metrics, scale float32) text.Metrics {
	if scale <= 0 || scale == 1 {
		return m
	}
	inv := 1.0 / scale
	m.Width = m.Width * inv
	m.Height = m.Height * inv
	m.Ascent = m.Ascent * inv
	m.Descent = m.Descent * inv
	return m
}
