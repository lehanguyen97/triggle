package ui

import (
	"triggle/engine/text"
	"triggle/engine/ui/theme"
)

// Label draws a single-line label at the top-left of the current content rect.
func (c *Context) Label(s string) {
	if c == nil || c.font == nil || s == "" {
		return
	}
	r := c.ContentRect()
	col := c.theme.Colors[theme.ColorText]
	c.font.Draw(&c.enc, s, r.X, r.Y, c.theme.BodyPx,
		text.Color{R: col.R, G: col.G, B: col.B, A: col.A})
}
