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

var zeroSlice = make([]byte, 0)

// NewDefaultRetentionPolicy creates a new retention policy with default values
// based on the given maxRetained size.
func NewDefaultRetentionPolicy(maxRetained int) *RetentionPolicy {
	return &RetentionPolicy{
		Warmup:    3,
		Threshold: 256,
		Ratio:     0.5,
		Max:       maxRetained,
		Default:   maxRetained / 2,
	}
}

// RetentionPolicy is the retention policy for a pool.
type RetentionPolicy struct {
	Warmup    int
	Threshold int
	Ratio     float64
	Max       int
	Default   int
}

// Retainable returns true if the capacity is reusable for the need.
func (p *RetentionPolicy) Retainable(capacity int) bool {
	return capacity <= p.Max
}

// Reusable returns true if the capacity is reusable for the need.
func (p *RetentionPolicy) Reusable(capacity, need int) bool {
	if capacity < need {
		return false
	}

	if capacity-need <= p.Threshold {
		return true
	}

	return float64(capacity-need)/float64(capacity) <= p.Ratio
}

// Pool is a pool of borrowed values that can be reused to avoid allocations.
type Pool[HT Header] struct {
	pol            *RetentionPolicy
	pool           sync.Pool
	headerOnlyPool sync.Pool
}

// NewPool creates a new pool of borrowed values with exponential payload size buckets.
// Buckets start at defaultSize and grow by powers of two up to maxRetainedBucketSize.
func NewPool[HT Header](pol *RetentionPolicy) *Pool[HT] {
	p := &Pool[HT]{
		pol: pol,
	}

	p.pool = sync.Pool{
		New: func() any {
			return &Frame[HT]{
				pool:    p,
				Payload: make([]byte, 0, pol.Default),
			}
		},
	}

	p.headerOnlyPool = sync.Pool{
		New: func() any {
			return &Frame[HT]{
				pool:    p,
				Payload: zeroSlice,
			}
		},
	}

	for range pol.Warmup {
		p.pool.Put(p.pool.New())
	}

	return p
}

// Get a value from the pool that needs to hold at least need bytes of payload.
// The returned value should be released back to the pool.
func (p *Pool[HT]) Get(need uint32) *Frame[HT] {
	if need == 0 {
		return p.headerOnlyPool.Get().(*Frame[HT]) // nolint:errcheck // safe
	}

	f := p.pool.Get().(*Frame[HT]) // nolint:errcheck // safe
	n := int(need)
	if !p.pol.Reusable(cap(f.Payload), n) {
		return &Frame[HT]{
			pool:    nil, // don't return to pool, too large
			Payload: make([]byte, n),
		}
	}

	var zeros HT
	f.Header = zeros
	f.Payload = f.Payload[:n]
	f.pool = p

	return f
}

// put returns a borrowed value to the pool. The value should not be used after calling Put.
func (p *Pool[HT]) put(v *Frame[HT]) {
	if v.pool == nil {
		return
	}

	if cap(v.Payload) == 0 {
		p.headerOnlyPool.Put(v)
		return
	}

	if !p.pol.Retainable(cap(v.Payload)) {
		return
	}

	v.pool = nil
	p.pool.Put(v)
}
