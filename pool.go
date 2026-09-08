// MIT License
//
// Copyright (C) 2026 John Kleijn
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE

package frame

import (
	"sync"
)

const poolWarmupSize = 16

// Pool is a pool of borrowed values that can be reused to avoid allocations.
type Pool[HT Header] struct {
	pool            sync.Pool
	maxRetainedSize int
	defaultSize     int
}

// NewPool creates a new pool of borrowed values with the given default byte slice size.
func NewPool[HT Header](defaultSize, maxRetainedSize int) *Pool[HT] {
	p := &Pool[HT]{
		defaultSize:     defaultSize,
		maxRetainedSize: maxRetainedSize,
	}

	p.pool = sync.Pool{
		New: func() any {
			return &Frame[HT]{
				pool:    p,
				Payload: make([]byte, 0, defaultSize),
			}
		},
	}

	// minimal warmup to avoid allocations on first use
	for range poolWarmupSize {
		p.pool.Put(p.pool.New())
	}

	return p
}

// Get returns a borrowed frame from the pool. The returned value should be released back to the pool
func (p *Pool[HT]) Get() *Frame[HT] {
	f := p.pool.Get().(*Frame[HT]) // nolint:errcheck // safe
	f.pool = p

	return f
}

// put returns a borrowed value to the pool. The value should not be used after calling Put.
func (p *Pool[HT]) put(v *Frame[HT]) {
	if v.pool == nil {
		return
	}

	var zeros HT
	v.Header = zeros
	v.pool = nil

	if cap(v.Payload) > p.maxRetainedSize {
		v.Payload = make([]byte, 0, p.defaultSize)
	} else {
		v.Payload = v.Payload[:0]
	}

	p.pool.Put(v)
}
