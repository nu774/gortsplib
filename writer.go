package gortsplib

import (
	"context"
	"sync/atomic"

	"github.com/smallnest/chanx"
)

// this struct contains a queue that allows to detach the routine that is reading a stream
// from the routine that is writing a stream.
type writer struct {
	running atomic.Bool
	ubc     atomic.Pointer[chanx.UnboundedChan[func()]]
	ctx     context.Context
	cancel  context.CancelFunc
}

func (w *writer) allocateBuffer(size int) {
	if w.ubc.Load() == nil {
		ctx, cancel := context.WithCancel(context.Background())
		ubc := chanx.NewUnboundedChan[func()](ctx, size)
		_ = cancel
		if w.ubc.CompareAndSwap(nil, ubc) {
			w.ctx, w.cancel = ctx, cancel
		}
	}
}

func (w *writer) start() {
	if !w.running.CompareAndSwap(false, true) {
		w.allocateBuffer(32)
		go w.run()
	}
}

func (w *writer) stop() {
	if ubc := w.ubc.Load(); ubc != nil {
		if w.ubc.CompareAndSwap(ubc, nil) {
			w.running.Store(false)
			w.cancel()
		}
	}
}

func (w *writer) run() {
	for fn := range w.ubc.Load().Out {
		fn()
	}
}

func (w *writer) queue(cb func()) {
	if ubc := w.ubc.Load(); ubc != nil {
		ubc.In <- cb
	}
}
