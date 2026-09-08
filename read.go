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

// PayloadReadLimit configures the maximum payload bytes Reader.Read returns.
type PayloadReadLimit uint32

const (
	// MaxPayloadSize means "no read limit" when passed to NewReader.
	MaxPayloadSize = PayloadReadLimit(^uint32(0))

	// HeadersOnly means "read only the header" when passed to NewReader.
	HeadersOnly = PayloadReadLimit(0)
)

// Reader is a reader that uses a pool of borrowed values to avoid allocations.
type Reader[HT Header] struct {
	stream     io.ReaderAt
	pool       *Pool[HT]
	header     HT
	headerView []byte
	limit      uint32
	headerSize int
}

// NewReader creates a new BorrowedFrameReader with the given stream and pool.
// The limit parameter specifies the maximum number of bytes to read for the payload.
func NewReader[HT Header](stream io.ReaderAt, pool *Pool[HT], limit PayloadReadLimit) *Reader[HT] {
	r := &Reader[HT]{
		stream: stream,
		pool:   pool,
		limit:  uint32(limit),
	}
	r.headerSize = len(r.header)
	r.headerView = headerBytes(&r.header)

	return r
}

// Read reads the frame header and payload at the given offset,
// and returns a borrowed value from the pool.
func (r *Reader[HT]) Read(offset int64) (f *Frame[HT], err error) {
	err = readAt(r.stream, r.headerView, offset)
	if err != nil {
		return nil, err
	}

	n := min(r.header.PayloadLength(), r.limit)
	f = r.pool.Get(n)
	f.Header = r.header

	defer func() {
		if err != nil && f != nil {
			f.Return()
		}
	}()

	if n == 0 {
		return f, nil
	}

	err = readAt(r.stream, f.Payload, offset+int64(r.headerSize))
	if err != nil {
		return nil, err
	}

	return f, err
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
