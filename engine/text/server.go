package text

import (
	"container/list"
	"fmt"
	"sync"

	"triggle/engine/backend"
	"triggle/engine/emath"
	"triggle/engine/render"
)

// maxCachedLines caps the LRU shared across all fonts/sizes on one server.
// Per-build constant: native shares one glyph atlas across lines (cheap);
// WASM allocates one GPU image per line, so the cap must stay safely under
// sokol's default image_pool_size (128) with margin for the shadow map,
// draw2d white texture, and any material textures. See server_native.go /
// server_wasm.go.

// maxMeasuredLines caps metrics-only lines so repeated MeasureLine calls do not
// repeatedly cross the backend boundary without preparing render resources.
const maxMeasuredLines = 512

// maxUnusedCacheFrames proactively drops derived text data that has not been
// touched for a while. LRU still caps peak size; this clears old font sizes after
// theme/layout changes even if the cap is not reached.
const maxUnusedCacheFrames uint64 = 300

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
	opts    Options
	content string
}

type measuredLine struct {
	key           lineKey
	metrics       Metrics
	lastUsedFrame uint64
}

// volatileKey identifies a volatile shaped line (not part of the shared LRU).
// Owner is an app-supplied stable id (e.g. a widget id). The contract is one
// live entry per owner: the same widget mutating its content/font/size reuses
// (and destroys-then-replaces) its single slot. Including font or opts in the
// key would orphan the previous entry every time SizePx or font changed —
// e.g. on every UIScale step during a window resize.
type volatileKey = uint32

// lineQuad is one glyph (or whole-line on WASM) quad in local pixel space.
type lineQuad struct {
	Dst emath.Rect
	UV  emath.UVRect
}

// cachedLine is one shaped line. On native: quads point into the shared atlas
// (img/samp are atlas handles). On WASM: one per-line texture (img) + shared
// server sampler (samp); destroyed on eviction.
type cachedLine struct {
	key           lineKey
	img           int32
	samp          int32
	glyphs        []lineQuad
	size          emath.Vec2
	line          *Line
	perLineImg    bool
	lastUsedFrame uint64
}

// textServer owns atlas/sampler, per-font per-size backend handles, and the
// shared LRU line cache. One server per backend.
type textServer struct {
	b          backend.Backend
	nextID     int32
	fonts      map[int32]*fontEntry
	lines      map[lineKey]*list.Element
	lru        *list.List
	measures   map[lineKey]*list.Element
	measureLRU *list.List
	// Volatile lines are kept out of the shared cache so rapidly-changing strings
	// (e.g. text input) don't evict stable UI text.
	volatile map[volatileKey]*cachedLine

	frame uint64
	plat  platState // per-platform: atlas (native) or sampler (wasm)
}

func newServer(b backend.Backend) (*textServer, error) {
	srv := &textServer{
		b:          b,
		nextID:     1,
		fonts:      make(map[int32]*fontEntry),
		lines:      make(map[lineKey]*list.Element),
		lru:        list.New(),
		measures:   make(map[lineKey]*list.Element),
		measureLRU: list.New(),
		volatile:   make(map[volatileKey]*cachedLine),
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
	srv.measures = nil
	srv.lru.Init()
	srv.measureLRU.Init()
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
		if line != nil && line.key.fontID == fontID {
			srv.destroyLine(line)
			delete(srv.volatile, k)
		}
	}
	for _, h := range f.handles {
		srv.b.TextFontClose(h)
	}
	delete(srv.fonts, fontID)
}

func (srv *textServer) beginFrame() {
	if srv == nil {
		return
	}
	srv.frame++
	if srv.frame == 0 {
		srv.frame = 1
	}
}

func (srv *textServer) endFrame() {
	if srv == nil {
		return
	}
	srv.sweepUnused(maxUnusedCacheFrames)
}

func (srv *textServer) markLineUsed(line *cachedLine) {
	if line != nil {
		line.lastUsedFrame = srv.frame
	}
}

func (srv *textServer) markMeasureUsed(measured *measuredLine) {
	if measured != nil {
		measured.lastUsedFrame = srv.frame
	}
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
	for el := srv.measureLRU.Front(); el != nil; {
		next := el.Next()
		measured := el.Value.(*measuredLine)
		if measured.key.fontID == fontID {
			srv.measureLRU.Remove(el)
			delete(srv.measures, measured.key)
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
			Ascent:     float32(bm.AscentPx),
			Descent:    float32(bm.DescentPx),
			LineHeight: float32(bm.LineSkipPx),
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

func (srv *textServer) measure(fontID int32, s string, opts Options) Metrics {
	f, ok := srv.fonts[fontID]
	if !ok {
		return Metrics{}
	}
	opts = normalizedOptions(opts)
	pxSize := opts.SizePx
	k := lineKey{fontID: fontID, opts: opts, content: s}
	// Hot path: caret-prefix Measure on the focused text-input runs every
	// blink frame. If the same string was just shaped by getLine, its cached
	// size is already on the cachedLine — reuse it instead of re-crossing
	// HarfBuzz (native) or the JS Canvas measureText (WASM).
	if el, ok := srv.lines[k]; ok {
		srv.lru.MoveToFront(el)
		line := el.Value.(*cachedLine)
		srv.markLineUsed(line)
		return srv.metricsFromSize(fontID, pxSize, line.size)
	}
	if el, ok := srv.measures[k]; ok {
		srv.measureLRU.MoveToFront(el)
		measured := el.Value.(*measuredLine)
		srv.markMeasureUsed(measured)
		return measured.metrics
	}
	h, err := srv.ensureHandle(f, pxSize)
	if err != nil {
		return Metrics{}
	}
	m, ok := srv.b.TextMeasureUTF8(h, s)
	if !ok {
		return Metrics{}
	}
	size := emath.Vec2{float32(m.Width26_6) / 64, float32(m.Height26_6) / 64}
	metrics := srv.metricsFromSize(fontID, pxSize, size)
	srv.rememberMeasure(k, metrics)
	return metrics
}

func (srv *textServer) line(fontID int32, s string, opts Options) *Line {
	f, ok := srv.fonts[fontID]
	if !ok {
		return nil
	}
	opts = normalizedOptions(opts)
	line, err := srv.getLine(f, s, opts)
	if err != nil || line == nil || line.line == nil {
		return nil
	}
	return line.line
}

func (srv *textServer) volatileLine(fontID int32, owner uint32, s string, opts Options) *Line {
	f, ok := srv.fonts[fontID]
	if !ok {
		return nil
	}
	opts = normalizedOptions(opts)
	line, err := srv.getVolatileLine(f, owner, s, opts)
	if err != nil || line == nil || line.line == nil {
		return nil
	}
	return line.line
}

func (srv *textServer) bindRenderableLine(fontID int32, opts Options, line *cachedLine) {
	if line == nil {
		return
	}
	line.line = &Line{
		Metrics:  srv.metricsFromSize(fontID, opts.SizePx, line.size),
		Segments: make([]Segment, 0, len(line.glyphs)),
	}
	for _, g := range line.glyphs {
		line.line.Segments = append(line.line.Segments, Segment{
			Image:   render.ImageHandle(line.img),
			Sampler: render.SamplerHandle(line.samp),
			Dst:     g.Dst,
			UV:      g.UV,
		})
	}
}

func (srv *textServer) metricsFromSize(fontID int32, pxSize int32, size emath.Vec2) Metrics {
	m := srv.metrics(fontID, pxSize)
	m.Width = size[0]
	m.Height = size[1]
	if m.Height <= 0 {
		m.Height = m.Ascent + m.Descent
	}
	if m.Height <= 0 {
		m.Height = float32(pxSize)
	}
	return m
}

func normalizedSize(pxSize int32) int32 {
	if pxSize < 1 {
		return 1
	}
	return pxSize
}

func normalizedOptions(opts Options) Options {
	opts.SizePx = normalizedSize(opts.SizePx)
	return opts
}

func (srv *textServer) rememberMeasure(k lineKey, metrics Metrics) {
	if srv.measures == nil {
		srv.measures = make(map[lineKey]*list.Element)
	}
	if srv.measureLRU == nil {
		srv.measureLRU = list.New()
	}
	if el, ok := srv.measures[k]; ok {
		measured := el.Value.(*measuredLine)
		measured.metrics = metrics
		srv.markMeasureUsed(measured)
		srv.measureLRU.MoveToFront(el)
		return
	}
	measured := &measuredLine{key: k, metrics: metrics}
	srv.markMeasureUsed(measured)
	el := srv.measureLRU.PushFront(measured)
	srv.measures[k] = el
	for srv.measureLRU.Len() > maxMeasuredLines {
		tail := srv.measureLRU.Back()
		srv.measureLRU.Remove(tail)
		measured := tail.Value.(*measuredLine)
		delete(srv.measures, measured.key)
	}
}

func (srv *textServer) dropVolatile(owner uint32) {
	if owner == 0 || srv.volatile == nil {
		return
	}
	if line, ok := srv.volatile[owner]; ok {
		srv.destroyLine(line)
		delete(srv.volatile, owner)
	}
}

func (srv *textServer) sweepUnused(graceFrames uint64) {
	if srv == nil || srv.frame <= graceFrames {
		return
	}
	cutoff := srv.frame - graceFrames
	for el := srv.lru.Front(); el != nil; {
		next := el.Next()
		line := el.Value.(*cachedLine)
		if line.lastUsedFrame <= cutoff {
			srv.lru.Remove(el)
			delete(srv.lines, line.key)
			srv.destroyLine(line)
		}
		el = next
	}
	for el := srv.measureLRU.Front(); el != nil; {
		next := el.Next()
		measured := el.Value.(*measuredLine)
		if measured.lastUsedFrame <= cutoff {
			srv.measureLRU.Remove(el)
			delete(srv.measures, measured.key)
		}
		el = next
	}
	for owner, line := range srv.volatile {
		if line != nil && line.lastUsedFrame <= cutoff {
			srv.destroyLine(line)
			delete(srv.volatile, owner)
		}
	}
}

// getLine returns a cached line for (font, pxSize, s), shaping on miss and
// evicting the LRU tail when the cache overflows.
func (srv *textServer) getLine(f *fontEntry, s string, opts Options) (*cachedLine, error) {
	k := lineKey{fontID: f.id, opts: opts, content: s}
	if el, ok := srv.lines[k]; ok {
		srv.lru.MoveToFront(el)
		line := el.Value.(*cachedLine)
		srv.markLineUsed(line)
		return line, nil
	}
	line, err := srv.shapeLine(f, s, opts.SizePx)
	if err != nil {
		return nil, err
	}
	line.key = k
	srv.markLineUsed(line)
	srv.bindRenderableLine(f.id, opts, line)
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
func (srv *textServer) getVolatileLine(f *fontEntry, owner uint32, s string, opts Options) (*cachedLine, error) {
	if srv.volatile == nil {
		srv.volatile = make(map[volatileKey]*cachedLine)
	}
	old := srv.volatile[owner]
	if old != nil && old.key.fontID == f.id && old.key.opts == opts && old.key.content == s {
		srv.markLineUsed(old)
		return old, nil
	}
	line, err := srv.shapeVolatileLine(f, owner, s, opts.SizePx, old)
	if err != nil {
		// Keep the previous line (if any) rather than flickering the widget.
		srv.markLineUsed(old)
		return old, nil
	}
	if line == nil {
		if old != nil {
			srv.destroyLine(old)
		}
		delete(srv.volatile, owner)
		return nil, nil
	}
	line.key = lineKey{fontID: f.id, opts: opts, content: s}
	srv.markLineUsed(line)
	srv.bindRenderableLine(f.id, opts, line)
	if old != nil && old != line && old.img != line.img {
		srv.destroyLine(old)
	}
	srv.volatile[owner] = line
	return line, nil
}
