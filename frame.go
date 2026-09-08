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

import "bytes"

// Frame is a wrapper around a byte slice that is borrowed from a pool.
type Frame[HT Header] struct {
	Header        HT
	pool          *Pool[HT]
	Payload       []byte
	payloadBucket int
}

// Clone copies the borrowed value into a new slice and returns it along with the header.
// The frame is returned to the pool after cloning so it should not be used afterwards.
func (f *Frame[HT]) Clone() (HT, []byte) {
	h := f.Header
	payload := bytes.Clone(f.Payload)

	f.Return() // TODO: this may be a bit unexpected although documented

	return h, payload
}

// Set sets the header and value of the borrowed value.
func (f *Frame[HT]) Set(h HT, p []byte) *Frame[HT] {
	f.Header = h
	f.Payload = append(f.Payload[:0], p...)

	return f
}

// Return the borrowed value to the pool.
// It should be called exactly once in the same context where the value
// was obtained, and the value should not be used after calling it.
func (f *Frame[HT]) Return() {
	if f.pool == nil {
		return
	}

	f.pool.put(f)
}
