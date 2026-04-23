package event

// Queue buffers Events between the host transport and the game frame loop.
// It is a simple grow-on-demand slice (not a fixed ring) — frames typically
// produce <100 events, so the slice cost is negligible and reusing the
// backing array via Reset avoids per-frame allocation.
//
// Concurrency: the host transport calls Push from the same OS thread as
// game_frame on both targets (sokol's main thread on native, the JS main
// thread on WASM). No lock is required.
type Queue struct {
	buf []Event
}

// Push appends an event. Called by the platform transport on each host
// callback.
func (q *Queue) Push(ev Event) {
	q.buf = append(q.buf, ev)
}

// Drain calls fn for each queued event in order, then resets the queue.
// Reuses the backing array so steady-state frames allocate nothing.
func (q *Queue) Drain(fn func(Event)) {
	for _, ev := range q.buf {
		fn(ev)
	}
	q.buf = q.buf[:0]
}

// Len reports the number of pending events (rare; mostly for tests).
func (q *Queue) Len() int { return len(q.buf) }

// DefaultQueue is the singleton the platform transports push into. Game code
// drains it once per frame. A singleton matches the host model: there is one
// event stream per process.
var DefaultQueue Queue
