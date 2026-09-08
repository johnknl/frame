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
	"hash/crc32"
	"math"
)

var crc32cTable = crc32.MakeTable(crc32.Castagnoli)

// CRC32C validates and computes frame checksums using CRC32C (Castagnoli).
type CRC32C[HT Header] struct {
}

// NewCRC32C creates a new CRC32C validator.
func NewCRC32C[HT Header]() *CRC32C[HT] {
	return &CRC32C[HT]{}
}

// Validate checks if the frame header is valid for the given payload
func (v *CRC32C[HT]) Validate(f *Frame[HT]) error {
	size := len(f.Payload)
	if size > math.MaxUint32 {
		// can't check the payload length if it exceeds the maximum uint32 value,
		// since PayloadLength() returns a uint32
		return ErrPayloadTooLarge
	}

	h := f.Header
	if h.PayloadLength() != uint32(size) {
		return ErrInvalidFrameHeader
	}

	if h.Checksum() != v.Sum(f) {
		return ErrInvalidChecksum
	}

	return nil
}

// Sum computes the CRC32C checksum for a frame.
func (v *CRC32C[HT]) Sum(f *Frame[HT]) uint32 {
	sum := crc32.Update(0, crc32cTable, f.Header.ChecksumBytes())
	return crc32.Update(sum, crc32cTable, f.Payload)
}
