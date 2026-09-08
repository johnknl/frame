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

import "errors"

var (
	// ErrPayloadTooLarge is returned when the payload size exceeds the maximum allowed size.
	ErrPayloadTooLarge = errors.New("payload too large")

	// ErrInvalidChecksum is returned when the checksum of a frame does not match the expected value.
	ErrInvalidChecksum = errors.New("invalid checksum")

	// ErrInvalidFrameIndex is returned when the decoded frame index does not match expectations.
	ErrInvalidFrameIndex = errors.New("invalid frame index")

	// ErrInvalidFrameHeader is returned when a frame header is invalid
	ErrInvalidFrameHeader = errors.New("invalid frame header")
)
