package ui

import (
	"fmt"

	"triggle/engine/geom"
	"triggle/engine/text"
)

// UIFont caches shaped Lines for immediate-mode widgets.
// Callers supply a stable cacheKey; on content change the Line is reshaped in place.
// Empty cacheKey = shape+close each call (slow, always correct).
type UIFont struct {
	face   *text.Face
	cache  map[string]*uiFontEntry
	closed bool
}

type uiFontEntry struct {
	content string
	line    *text.Line
}

// NewUIFont wraps a Face with a per-key Line cache.
func NewUIFont(face *text.Face) *UIFont {
	if face == nil {
		return nil
	}
	return &UIFont{
		face:  face,
		cache: make(map[string]*uiFontEntry),
	}
}

// Metrics returns the underlying face metrics.
func (u *UIFont) Metrics() text.Metrics {
	if u == nil || u.face == nil {
		return text.Metrics{}
	}
	return u.face.Metrics()
}

// MeasureLine returns width/height in pixels without caching a Line.
func (u *UIFont) MeasureLine(s string) (geom.Vec2, error) {
	if u == nil || u.face == nil {
		return geom.Vec2{}, fmt.Errorf("ui: nil UIFont")
	}
	return u.face.MeasureLine(s)
}

// Evict closes and removes the cached line under cacheKey.
func (u *UIFont) Evict(cacheKey string) {
	if u == nil {
		return
	}
	if e, ok := u.cache[cacheKey]; ok {
		if e.line != nil {
			e.line.Close()
		}
		delete(u.cache, cacheKey)
	}
}

// EvictAll closes and removes all cached lines (face is not closed).
func (u *UIFont) EvictAll() {
	if u == nil {
		return
	}
	for k, e := range u.cache {
		if e != nil && e.line != nil {
			e.line.Close()
		}
		delete(u.cache, k)
	}
}

// Close releases all cached lines. The Face is owned by the caller and must be closed separately.
func (u *UIFont) Close() {
	if u == nil || u.closed {
		return
	}
	u.closed = true
	u.EvictAll()
	u.face = nil
}

// drawLine shapes (or reshapes on content change) and emits quads.
// cacheKey == "" → no cache: shape, draw, close. Correct but slow; widgets should supply a key.
func (u *UIFont) drawLine(sink text.QuadSink, cacheKey, content string, x, y float32, color text.Color) {
	if u == nil || u.face == nil || sink == nil || content == "" {
		return
	}
	if cacheKey == "" {
		line, err := u.face.ShapeLine(content)
		if err != nil {
			return
		}
		line.Draw(sink, x, y, color)
		line.Close()
		return
	}
	e := u.cache[cacheKey]
	if e == nil {
		line, err := u.face.ShapeLine(content)
		if err != nil {
			return
		}
		e = &uiFontEntry{content: content, line: line}
		u.cache[cacheKey] = e
	} else if e.content != content {
		if e.line != nil {
			e.line.Close()
		}
		line, err := u.face.ShapeLine(content)
		if err != nil {
			delete(u.cache, cacheKey)
			return
		}
		e.content = content
		e.line = line
	}
	e.line.Draw(sink, x, y, color)
}
