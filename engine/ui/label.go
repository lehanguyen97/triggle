package ui

import "triggle/engine/ui/theme"

// Label draws a single-line label at the top-left of the current content rect.
func (c *Context) Label(s string) {
	if c == nil || c.uiFont == nil || s == "" {
		return
	}
	r := c.ContentRect()
	c.DrawText("", s, r.X, r.Y, c.theme.Colors[theme.ColorText])
}
