package gortsplib

import (
	"sync"

	"github.com/smallnest/chanx"
)

// this struct contains a queue that allows to detach the routine that is reading a stream
// from the routine that is writing a stream.
type writer struct {
	running bool
	closed  bool
	mu      sync.Mutex
	ubc     *chanx.UnboundedChan[func()]
}

func (w *writer) allocateBuffer(size int) {
	w.ubc = chanx.NewUnboundedChan[func()](size)
}

func (w *writer) start() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.running {
		w.running = true
		go w.run()
	}
}

func (w *writer) stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.closed {
		w.running = false
		w.closed = true
		close(w.ubc.In)
	}
}

func (w *writer) run() {
	for fn := range w.ubc.Out {
		fn()
	}
}

func (w *writer) queue(cb func()) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.closed {
		w.ubc.In <- cb
	}
}
