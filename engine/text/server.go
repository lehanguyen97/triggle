package text

import (
	"container/list"
	"fmt"
	"sync"

	"triggle/engine/backend"
	"triggle/engine/emath"
)

// maxCachedLines caps the LRU shared across all fonts/sizes on one server.
// Not configurable for now; revisit when a real workload overflows it.
const maxCachedLines = 256

// serversMu guards servers. The registry is per-backend, not literally global.
var (
	serversMu sync.Mutex
	servers   = map[backend.Backend]*serverSlot{}
)

type serverSlot struct {
	srv  *textServer
	refs int32
}

// acquireServer returns the server for b, creating it on first use.
func acquireServer(b backend.Backend) (*textServer, error) {
	serversMu.Lock()
	defer serversMu.Unlock()
	slot, ok := servers[b]
	if !ok {
		srv, err := newServer(b)
		if err != nil {
			return nil, err
		}
		slot = &serverSlot{srv: srv}
		servers[b] = slot
	}
	slot.refs++
	return slot.srv, nil
}

// releaseServer decrements the ref for b; closes the server when it hits zero.
func releaseServer(b backend.Backend) {
	serversMu.Lock()
	defer serversMu.Unlock()
	slot, ok := servers[b]
	if !ok {
		return
	}
	slot.refs--
	if slot.refs > 0 {
		return
	}
	slot.srv.close()
	delete(servers, b)
}

// fontEntry tracks one loaded TTF: its path, per-size backend handles, and
// per-size metrics. Handles are opened lazily on first use of a new size.
type fontEntry struct {
	id      int32
	path    string
	handles map[int32]int32 // pxSize -> backend font handle
	metrics map[int32]Metrics
}

// lineKey identifies a cached shaped line (per font, size, content).
type lineKey struct {
	fontID  int32
	pxSize  int32
	content string
}

// volatileKey identifies a volatile shaped line (not part of the shared LRU).
// Owner is an app-supplied stable id (e.g. a widget id).
type volatileKey struct {
	owner  uint32
	fontID int32
	pxSize int32
}

// lineQuad is one glyph (or whole-line on WASM) quad in local pixel space.
type lineQuad struct {
	Dst emath.Rect
	UV  emath.UVRect
}

// cachedLine is one shaped line. On native: quads point into the shared atlas
// (img/samp are atlas handles). On WASM: one per-line texture (img) + shared
// server sampler (samp); destroyed on eviction.
type cachedLine struct {
	key        lineKey
	img        int32
	samp       int32
	glyphs     []lineQuad
	size       emath.Vec2
	perLineImg bool
}

// textServer owns atlas/sampler, per-font per-size backend handles, and the
// shared LRU line cache. One server per backend.
type textServer struct {
	b      backend.Backend
	nextID int32
	fonts  map[int32]*fontEntry
	lines  map[lineKey]*list.Element
	lru    *list.List
	// Volatile lines are kept out of the shared cache so rapidly-changing strings
	// (e.g. text input) don't evict stable UI text.
	volatile map[volatileKey]*cachedLine

	plat platState // per-platform: atlas (native) or sampler (wasm)
}

func newServer(b backend.Backend) (*textServer, error) {
	srv := &textServer{
		b:        b,
		nextID:   1,
		fonts:    make(map[int32]*fontEntry),
		lines:    make(map[lineKey]*list.Element),
		lru:      list.New(),
		volatile: make(map[volatileKey]*cachedLine),
	}
	plat, err := initPlat(b)
	if err != nil {
		return nil, err
	}
	srv.plat = plat
	return srv, nil
}

func (srv *textServer) close() {
	for el := srv.lru.Front(); el != nil; el = el.Next() {
		srv.destroyLine(el.Value.(*cachedLine))
	}
	for _, line := range srv.volatile {
		srv.destroyLine(line)
	}
	srv.volatile = nil
	srv.lines = nil
	srv.lru.Init()
	for _, f := range srv.fonts {
		for _, h := range f.handles {
			srv.b.TextFontClose(h)
		}
	}
	srv.fonts = nil
	srv.plat.close(srv.b)
}

func (srv *textServer) openFont(path string) (int32, error) {
	id := srv.nextID
	srv.nextID++
	srv.fonts[id] = &fontEntry{
		id:      id,
		path:    path,
		handles: make(map[int32]int32),
		metrics: make(map[int32]Metrics),
	}
	return id, nil
}

func (srv *textServer) closeFont(fontID int32) {
	f, ok := srv.fonts[fontID]
	if !ok {
		return
	}
	srv.evictByFont(fontID)
	for k, line := range srv.volatile {
		if k.fontID == fontID {
			srv.destroyLine(line)
			delete(srv.volatile, k)
		}
	}
	for _, h := range f.handles {
		srv.b.TextFontClose(h)
	}
	delete(srv.fonts, fontID)
}

// evictByFont drops all cached lines belonging to fontID (called on font close).
func (srv *textServer) evictByFont(fontID int32) {
	for el := srv.lru.Front(); el != nil; {
		next := el.Next()
		line := el.Value.(*cachedLine)
		if line.key.fontID == fontID {
			srv.lru.Remove(el)
			delete(srv.lines, line.key)
			srv.destroyLine(line)
		}
		el = next
	}
}

// ensureHandle opens (and caches) the backend font handle for (font, pxSize).
// Also caches metrics for that size.
func (srv *textServer) ensureHandle(f *fontEntry, pxSize int32) (int32, error) {
	if h, ok := f.handles[pxSize]; ok {
		return h, nil
	}
	h := srv.b.TextFontOpen(f.path, pxSize)
	if h < 0 {
		return -1, fmt.Errorf("text: open font %q @%dpx", f.path, pxSize)
	}
	f.handles[pxSize] = h
	if bm, ok := srv.b.TextFontMetrics(h); ok {
		f.metrics[pxSize] = Metrics{
			Ascent:     bm.AscentPx,
			Descent:    bm.DescentPx,
			LineHeight: bm.LineSkipPx,
		}
	}
	return h, nil
}

func (srv *textServer) metrics(fontID int32, pxSize int32) Metrics {
	f, ok := srv.fonts[fontID]
	if !ok {
		return Metrics{}
	}
	if m, ok := f.metrics[pxSize]; ok {
		return m
	}
	if _, err := srv.ensureHandle(f, pxSize); err != nil {
		return Metrics{}
	}
	return f.metrics[pxSize]
}

func (srv *textServer) measure(fontID int32, s string, pxSize int32) emath.Vec2 {
	f, ok := srv.fonts[fontID]
	if !ok {
		return emath.Vec2{}
	}
	// Hot path: caret-prefix Measure on the focused text-input runs every
	// blink frame. If the same string was just shaped by getLine, its cached
	// size is already on the cachedLine — reuse it instead of re-crossing
	// HarfBuzz (native) or the JS Canvas measureText (WASM).
	if el, ok := srv.lines[lineKey{fontID: fontID, pxSize: pxSize, content: s}]; ok {
		srv.lru.MoveToFront(el)
		return el.Value.(*cachedLine).size
	}
	h, err := srv.ensureHandle(f, pxSize)
	if err != nil {
		return emath.Vec2{}
	}
	m, ok := srv.b.TextMeasureUTF8(h, s)
	if !ok {
		return emath.Vec2{}
	}
	return emath.Vec2{float32(m.Width26_6) / 64, float32(m.Height26_6) / 64}
}

func (srv *textServer) draw(fontID int32, s string, x, y, pxSize int32, color Color, sink QuadSink) {
	f, ok := srv.fonts[fontID]
	if !ok {
		return
	}
	line, err := srv.getLine(f, s, pxSize)
	if err != nil || line == nil {
		return
	}
	for _, g := range line.glyphs {
		d := g.Dst
		d.X += x
		d.Y += y
		sink.AddTexturedQuad(line.img, line.samp, d, g.UV, color)
	}
}

func (srv *textServer) drawVolatile(fontID int32, owner uint32, s string, x, y, pxSize int32, color Color, sink QuadSink) {
	f, ok := srv.fonts[fontID]
	if !ok {
		return
	}
	line, err := srv.getVolatileLine(f, owner, s, pxSize)
	if err != nil || line == nil {
		return
	}
	for _, g := range line.glyphs {
		d := g.Dst
		d.X += x
		d.Y += y
		sink.AddTexturedQuad(line.img, line.samp, d, g.UV, color)
	}
}

func (srv *textServer) dropVolatile(owner uint32) {
	if owner == 0 || srv.volatile == nil {
		return
	}
	for k, line := range srv.volatile {
		if k.owner == owner {
			srv.destroyLine(line)
			delete(srv.volatile, k)
		}
	}
}

// getLine returns a cached line for (font, pxSize, s), shaping on miss and
// evicting the LRU tail when the cache overflows.
func (srv *textServer) getLine(f *fontEntry, s string, pxSize int32) (*cachedLine, error) {
	k := lineKey{fontID: f.id, pxSize: pxSize, content: s}
	if el, ok := srv.lines[k]; ok {
		srv.lru.MoveToFront(el)
		return el.Value.(*cachedLine), nil
	}
	line, err := srv.shapeLine(f, s, pxSize)
	if err != nil {
		return nil, err
	}
	line.key = k
	el := srv.lru.PushFront(line)
	srv.lines[k] = el
	for srv.lru.Len() > maxCachedLines {
		tail := srv.lru.Back()
		srv.lru.Remove(tail)
		tl := tail.Value.(*cachedLine)
		delete(srv.lines, tl.key)
		srv.destroyLine(tl)
	}
	return line, nil
}

// getVolatileLine returns the latest shaped line for (owner,font,pxSize). The
// line is not inserted into the shared LRU; on content change, the previous
// line is destroyed immediately and replaced.
func (srv *textServer) getVolatileLine(f *fontEntry, owner uint32, s string, pxSize int32) (*cachedLine, error) {
	if srv.volatile == nil {
		srv.volatile = make(map[volatileKey]*cachedLine)
	}
	k := volatileKey{owner: owner, fontID: f.id, pxSize: pxSize}
	old := srv.volatile[k]
	if old != nil && old.key.content == s {
		return old, nil
	}
	line, err := srv.shapeVolatileLine(f, owner, s, pxSize, old)
	if err != nil {
		// Keep the previous line (if any) rather than flickering the widget.
		return old, nil
	}
	if line == nil {
		if old != nil {
			srv.destroyLine(old)
		}
		delete(srv.volatile, k)
		return nil, nil
	}
	line.key = lineKey{fontID: f.id, pxSize: pxSize, content: s}
	if old != nil && old != line && old.img != line.img {
		srv.destroyLine(old)
	}
	srv.volatile[k] = line
	return line, nil
}
