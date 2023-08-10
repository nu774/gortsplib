package gortsplib

import (
	"sync/atomic"

	"github.com/smallnest/chanx"
)

// this struct contains a queue that allows to detach the routine that is reading a stream
// from the routine that is writing a stream.
type writer struct {
	running atomic.Bool
	ubc     *chanx.UnboundedChan[func()]
}

func (w *writer) allocateBuffer(size int) {
	w.ubc = chanx.NewUnboundedChan[func()](size)
}

func (w *writer) start() {
	if w.running.CompareAndSwap(false, true) {
		go w.run()
	}
}

func (w *writer) stop() {
	if w.running.CompareAndSwap(true, false) {
		close(w.ubc.In)
	}
}

func (w *writer) run() {
	for fn := range w.ubc.Out {
		fn()
	}
}

func (w *writer) queue(cb func()) {
	w.ubc.In <- cb
}
