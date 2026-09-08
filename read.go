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
	"errors"
	"io"
)

// Reader is a reader that uses a pool of borrowed values to avoid allocations.
type Reader[HT Header] struct {
	stream     io.ReaderAt
	pool       *Pool[HT]
	max        uint32
	headerSize int
}

// NewReader creates a new BorrowedFrameReader with the given stream and pool.
// The limit parameter specifies the maximum number of bytes to read for the payload.
func NewReader[HT Header](stream io.ReaderAt, pool *Pool[HT], limit uint32) *Reader[HT] {
	return &Reader[HT]{
		stream:     stream,
		pool:       pool,
		max:        limit,
		headerSize: len(headerBytes(new(HT))),
	}
}

// Read reads the frame header and payload at the given offset,
// and returns a borrowed value from the pool.
func (r *Reader[HT]) Read(offset int64) (f *Frame[HT], err error) {
	defer func() {
		if err != nil && f != nil {
			f.Return()
		}
	}()

	f = r.pool.Get()
	err = r.readHeaderAt(&f.Header, offset)
	if err != nil {
		return nil, err
	}

	if r.max == 0 {
		f.Payload = nil
		return f, nil
	}

	n := min(f.Header.PayloadLength(), r.max)

	// Ensure the borrowed payload slice is large enough to hold the payload
	if uint32(cap(f.Payload)) < n { // #nosec: G115 r.max is uint32
		f.Payload = make([]byte, n)
	} else {
		f.Payload = f.Payload[:n]
	}

	err = readAt(r.stream, f.Payload, offset+int64(r.headerSize))
	if err != nil {
		return nil, err
	}

	return f, err
}

// ReadHeader reads a frame header at offset
func (r *Reader[HT]) ReadHeader(offset int64) (HT, error) {
	var h HT
	err := r.readHeaderAt(&h, offset)
	if err != nil {
		return h, err
	}

	return h, nil
}

func (r *Reader[HT]) readHeaderAt(h *HT, offset int64) error {
	return readAt(r.stream, headerBytes(h), offset)
}

// readAt reads from the file at the given offset into the destination slice
func readAt(stream io.ReaderAt, dst []byte, offset int64) error {
	n, err := stream.ReadAt(dst, offset)
	if err != nil {
		if errors.Is(err, io.EOF) && n > 0 {
			return io.ErrUnexpectedEOF
		}

		return err
	}

	return nil
}
