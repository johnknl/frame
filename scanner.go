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

// Validator validates a decoded frame.
type Validator[HT Header] interface {
	Validate(f *Frame[HT]) error
}

// ScannerOption configures scanner behavior.
type ScannerOption[HT IndexedHeader] func(*Scanner[HT])

// WithScannerValidator sets the frame validator.
func WithScannerValidator[HT IndexedHeader](v Validator[HT]) ScannerOption[HT] {
	return func(s *Scanner[HT]) {
		s.v = v
	}
}

// WithScannerOffset sets the starting byte offset.
func WithScannerOffset[HT IndexedHeader](offset int64) ScannerOption[HT] {
	return func(s *Scanner[HT]) {
		s.offset = offset
	}
}

// WithScannerIndex sets the expected index for the first scanned frame.
func WithScannerIndex[HT IndexedHeader](index uint32) ScannerOption[HT] {
	return func(s *Scanner[HT]) {
		s.nextIdx = index
	}
}

// Scanner scans frames sequentially from a reader and validates frame indexes.
type Scanner[HT IndexedHeader] struct {
	v       Validator[HT]
	err     error
	r       *Reader[HT]
	f       *Frame[HT]
	offset  int64
	nextIdx uint32
}

// NewScanner creates a new scanner using the provided reader and options.
// Defaults are offset 0, index 0, and no validator.
func NewScanner[HT IndexedHeader](r *Reader[HT], options ...ScannerOption[HT]) *Scanner[HT] {
	s := &Scanner[HT]{
		r: r,
	}

	for _, option := range options {
		option(s)
	}

	return s
}

// Scan advances the scanner to the next frame.
func (s *Scanner[HT]) Scan() bool {
	if s.err != nil {
		return false
	}

	if s.f != nil {
		s.f.Return()
		s.f = nil
	}

	f, err := s.r.Read(s.offset)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return false
		}

		s.err = err
		return false
	}

	idx := f.Header.Index()
	if idx != s.nextIdx {
		f.Return()
		s.err = ErrInvalidFrameIndex
		return false
	}

	if s.v != nil {
		err = s.v.Validate(f)
		if err != nil {
			f.Return()
			s.err = err
			return false
		}
	}

	s.offset += int64(s.r.headerSize) + int64(f.Header.PayloadLength())
	s.nextIdx++
	s.f = f

	return true
}

// SeekOffset repositions the scanner to offset and clears the current frame and error.
// The expected frame index is unchanged.
func (s *Scanner[HT]) SeekOffset(offset int64) {
	if s.f != nil {
		s.f.Return()
		s.f = nil
	}

	s.offset = offset
	s.err = nil
}

// Seek repositions the scanner using io.Seeker semantics.
func (s *Scanner[HT]) Seek(offset int64, whence int) (int64, error) {
	base := int64(0)

	switch whence {
	case io.SeekStart:
		base = 0
	case io.SeekCurrent:
		base = s.offset
	default:
		return 0, errors.New("unsupported whence")
	}

	next := base + offset
	if next < 0 {
		return 0, errors.New("negative offset")
	}

	s.SeekOffset(next)

	return s.offset, nil
}

// SeekIndex scans forward from the current offset until index is found.
// On success, the scanner is positioned so the next Scan reads that frame.
func (s *Scanner[HT]) SeekIndex(index uint32) error {
	if s.err != nil {
		return s.err
	}

	if s.f != nil {
		s.f.Return()
		s.f = nil
	}

	offset := s.offset

	for {
		f, err := s.r.Read(offset)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return io.EOF
			}

			s.err = err
			return err
		}

		if s.v != nil {
			err = s.v.Validate(f)
			if err != nil {
				f.Return()
				s.err = err
				return err
			}
		}

		idx := f.Header.Index()
		if idx == index {
			f.Return()
			s.offset = offset
			s.nextIdx = index
			return nil
		}

		if idx > index {
			f.Return()
			s.err = ErrInvalidFrameIndex
			return s.err
		}

		offset += int64(s.r.headerSize) + int64(f.Header.PayloadLength())
		f.Return()
	}
}

// Offset returns the scanner's current byte offset.
// This is the offset where the next Scan starts.
func (s *Scanner[HT]) Offset() int64 {
	return s.offset
}

// Index returns the expected frame index for the next Scan.
func (s *Scanner[HT]) Index() uint32 {
	return s.nextIdx
}

// Frame returns the frame loaded by the most recent successful Scan call.
func (s *Scanner[HT]) Frame() *Frame[HT] {
	return s.f
}

// Close returns the currently held frame to its pool.
func (s *Scanner[HT]) Close() {
	if s.f == nil {
		return
	}

	s.f.Return()
	s.f = nil
}

// Err returns the first non-EOF error encountered by Scan.
func (s *Scanner[HT]) Err() error {
	return s.err
}
