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
	"encoding/binary"
)

const (
	// TestHeaderSize is the size of a frame header in bytes
	TestHeaderSize = 16
)

// TestHeader is a 16-byte big-endian encoded array, representing a frame header.
//
// offset  size  field
// 0       4     payload length
// 4       4     meta
// 8       4     reserved
// 12      4     checksum
type TestHeader [16]byte

// NextOffset returns the next offset after the current frame, given the current offset
func (h TestHeader) NextOffset(offset int64) int64 {
	return offset + TestHeaderSize + int64(h.PayloadLength())
}

// PayloadLength returns the payload length from the frame header
func (h TestHeader) PayloadLength() uint32 {
	return binary.BigEndian.Uint32(h[0:4])
}

func (h TestHeader) Index() uint32 {
	return binary.BigEndian.Uint32(h[4:8])
}

func (h TestHeader) Checksum() uint32 {
	return binary.BigEndian.Uint32(h[12:16])
}

func (h *TestHeader) SetChecksum(c uint32) {
	binary.BigEndian.PutUint32(h[12:16], c)
}

func (h TestHeader) ChecksumBytes() []byte {
	return h[:12]
}

// NewTestHeader constructs a new TestHeader
func NewTestHeader(m uint32, c uint32, payload []byte) TestHeader {
	var h TestHeader

	binary.BigEndian.PutUint32(h[0:4], uint32(len(payload)))
	binary.BigEndian.PutUint32(h[4:8], m)
	binary.BigEndian.PutUint32(h[12:16], c)

	return h
}
