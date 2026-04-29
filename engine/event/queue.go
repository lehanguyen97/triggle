package event

// Queue buffers Events between the host transport and the game frame loop.
// It is a simple grow-on-demand slice (not a fixed ring) — frames typically
// produce <100 events, so the slice cost is negligible and reusing the
// backing array via Drain avoids per-frame allocation.
//
// Concurrency: the host transport calls Push from the same OS thread as
// game_frame on both targets (sokol's main thread on native, the JS main
// thread on WASM). No lock is required.
type Queue struct {
	buf []Event
}

// NewQueue returns an empty queue ready to receive events.
func NewQueue() *Queue { return &Queue{} }

// Push appends an event.
func (q *Queue) Push(ev Event) {
	q.buf = append(q.buf, ev)
}

// Drain calls fn for each queued event in order, then resets the queue.
// Reuses the backing array so steady-state frames allocate nothing.
func (q *Queue) Drain(fn func(*Event)) {
	for i := range q.buf {
		fn(&q.buf[i])
	}
	q.buf = q.buf[:0]
}

// Len reports the number of pending events (rare; mostly for tests).
func (q *Queue) Len() int { return len(q.buf) }

// active is the queue the platform transport pushes into and the runtime
// drains from. Ownership lives on runtime.Host; SetActive is how Host
// registers its queue. C-ABI and go:wasmexport pushers have no receiver,
// so a package-level routing helper is the only viable shape.
var active *Queue

// SetActive registers the queue used by package-level Push and Drain. Called
// once by runtime.Host during init. Passing nil disables routing.
func SetActive(q *Queue) { active = q }

// Push routes an event to the active queue. Called by transport stubs.
func Push(ev Event) {
	if active != nil {
		active.Push(ev)
	}
}

// Drain drains the active queue, calling fn for each event.
func Drain(fn func(*Event)) {
	if active != nil {
		active.Drain(fn)
	}
}
