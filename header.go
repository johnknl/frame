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

import "unsafe"

// Header is a generic interface for frame headers. It is implemented by any type
// that has a fixed size of 8, 16, 32 or 64.
//
// Headers use fixed-size arrays primarily to avoid runtime bound checks in concrete
// implementations. This is unlikely to matter much if at all for performance,
// but it provides easier safety guarantees for the implementer.
type Header interface {
	~[8]byte | ~[16]byte | ~[32]byte | ~[64]byte | ~[128]byte // WARN: see headerBytes()

	// PayloadLength returns the length of the payload in bytes.
	PayloadLength() uint32

	// Checksum returns the checksum of the frame.
	// This should include the header and payload, but not the checksum field itself.
	Checksum() uint32

	// ChecksumBytes returns the bytes of the header that are included for checksumming.
	ChecksumBytes() []byte
}

// IndexedHeader is a header that includes a monotonic frame index.
type IndexedHeader interface {
	Header

	// Index returns the frame index.
	Index() uint32
}

// headerBytes returns the bytes of the header as a slice.
// This relies on Header being constrained to fixed-size byte arrays.
func headerBytes[HT Header](h *HT) []byte {
	return unsafe.Slice((*byte)(unsafe.Pointer(h)), unsafe.Sizeof(*h)) // #nosec: G103 // see above
}
