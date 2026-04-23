//go:build !js && !wasip1

package text

import (
	"fmt"

	"triggle/engine/backend"
)

// platState on native: one shared glyph atlas + sampler.
type platState struct {
	atlas *glyphAtlas
}

func initPlat(b backend.Backend) (platState, error) {
	ga, err := newGlyphAtlas(b, 512)
	if err != nil {
		return platState{}, err
	}
	return platState{atlas: ga}, nil
}

func (p platState) close(_ backend.Backend) {
	if p.atlas != nil {
		p.atlas.close()
	}
}

// shapeLine shapes s at (font, pxSize), packs glyphs into the atlas, and
// returns a cachedLine whose quads reference the atlas image/sampler.
func (srv *textServer) shapeLine(f *fontEntry, s string, pxSize int32) (*cachedLine, error) {
	h, err := srv.ensureHandle(f, pxSize)
	if err != nil {
		return nil, err
	}
	ascent := f.metrics[pxSize].Ascent
	atlas := srv.plat.atlas
	if atlas == nil {
		return nil, fmt.Errorf("text: nil atlas")
	}
	glyphs, size, err := atlas.shapeRun(h, s, ascent)
	if err != nil {
		return nil, err
	}
	atlas.uploadIfDirty()
	return &cachedLine{
		img:    atlas.atlasImg,
		samp:   atlas.sampler,
		glyphs: glyphs,
		size:   size,
	}, nil
}

// shapeVolatileLine is the volatile-cache variant of shapeLine. On native there
// is no per-line resource to reuse, so it delegates to shapeLine.
func (srv *textServer) shapeVolatileLine(f *fontEntry, owner uint32, s string, pxSize int32, prev *cachedLine) (*cachedLine, error) {
	_, _ = owner, prev
	return srv.shapeLine(f, s, pxSize)
}

// destroyLine: native cached lines reference the shared atlas, so there's
// nothing per-line to free; the atlas lives for the whole server lifetime.
func (srv *textServer) destroyLine(_ *cachedLine) {}
