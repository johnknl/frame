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

const poolWarmupPerBucket = 3

// Pool is a pool of borrowed values that can be reused to avoid allocations.
type Pool[HT Header] struct {
	framePools     []sync.Pool
	framePoolByCap map[int]int
	payloadCaps    []int
}

// NewPool creates a new pool of borrowed values with exponential payload size buckets.
// Buckets start at defaultSize and grow by powers of two up to maxRetainedBucketSize.
func NewPool[HT Header](defaultSize, maxRetainedBucketSize int) *Pool[HT] {
	if defaultSize < 1 {
		defaultSize = 1
	}

	if maxRetainedBucketSize < defaultSize {
		maxRetainedBucketSize = defaultSize
	}

	p := &Pool[HT]{}
	p.payloadCaps = buildPayloadBucketCaps(defaultSize, maxRetainedBucketSize)
	p.framePools = make([]sync.Pool, len(p.payloadCaps))
	p.framePoolByCap = make(map[int]int, len(p.payloadCaps))

	for i, capSize := range p.payloadCaps {
		idx := i
		c := capSize
		p.framePoolByCap[c] = idx

		p.framePools[idx] = sync.Pool{
			New: func() any {
				return &Frame[HT]{
					pool:          p,
					payloadBucket: idx,
					Payload:       make([]byte, 0, c),
				}
			},
		}

		for range poolWarmupPerBucket {
			p.framePools[idx].Put(p.framePools[idx].New())
		}
	}

	return p
}

// Get returns a borrowed frame from the bucket sized for payloadSize.
// The returned value should be released back to the pool.
func (p *Pool[HT]) Get(payloadSize uint32) *Frame[HT] {
	n := int(payloadSize) // #nosec: G115 payload size comes from header uint32
	idx := p.payloadBucketForNeed(n)
	if idx < 0 {
		return &Frame[HT]{
			pool:          nil,
			payloadBucket: -1,
			Payload:       make([]byte, n),
		}
	}

	f := p.framePools[idx].Get().(*Frame[HT]) // nolint:errcheck // safe
	f.pool = p
	f.payloadBucket = idx
	f.Payload = f.Payload[:n]

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
	v.Payload = v.Payload[:0]

	idx := v.payloadBucket
	if idx < 0 || idx >= len(p.framePools) || cap(v.Payload) != p.payloadCaps[idx] {
		idx = p.payloadBucketForCap(cap(v.Payload))
		if idx < 0 {
			return
		}
		v.payloadBucket = idx
	}

	p.framePools[idx].Put(v)
}

func (p *Pool[HT]) payloadBucketForNeed(need int) int {
	for i, c := range p.payloadCaps {
		if need <= c {
			return i
		}
	}

	return -1
}

func (p *Pool[HT]) payloadBucketForCap(c int) int {
	if idx, ok := p.framePoolByCap[c]; ok {
		return idx
	}

	return -1
}

func buildPayloadBucketCaps(minSize, maxSize int) []int {
	caps := make([]int, 0, 8)

	curr := minSize
	for curr < maxSize {
		caps = append(caps, curr)

		next := curr * 2
		if next <= curr {
			break
		}

		curr = next
	}

	if len(caps) == 0 || caps[len(caps)-1] != maxSize {
		caps = append(caps, maxSize)
	}

	return caps
}
