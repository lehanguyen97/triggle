package ui

import (
	"triggle/engine/text"
	"triggle/engine/ui/theme"
)

// Label draws a single-line label and reserves one line of vertical space in
// the current container (full container width, height = font line box).
func (c *Context) Label(s string) {
	if c == nil || c.font == nil || s == "" {
		return
	}
	px := c.theme.BodyPx
	m := c.font.Metrics(px)
	h := m.Ascent + m.Descent
	if h <= 0 {
		h = px
	}
	r := c.LayoutNextRow(h)
	col := c.theme.Colors[theme.ColorText]
	c.font.Draw(&c.enc, s, r.X, r.Y, px,
		text.Color{R: col.R, G: col.G, B: col.B, A: col.A})
}
