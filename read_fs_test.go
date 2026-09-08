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
	"testing"

	"github.com/johnknl/frame/internal/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestReader_ReadMapsPartialPayloadToUnexpectedEOF(t *testing.T) {
	t.Parallel()

	payload := []byte("abcd")
	h := NewTestHeader(0, 0, payload)

	f := mocks.NewMockReaderAt(t)
	f.EXPECT().ReadAt(mock.Anything, int64(0)).RunAndReturn(func(dst []byte, _ int64) (int, error) {
		copy(dst, h[:])
		return len(h), nil
	})
	f.EXPECT().ReadAt(mock.Anything, int64(TestHeaderSize)).RunAndReturn(func(dst []byte, _ int64) (int, error) {
		copy(dst[:2], payload[:2])
		return 2, io.EOF
	})

	r := NewReader(f, NewTestPool(4, 1024), ^uint32(0))
	_, err := r.Read(0)
	require.ErrorIs(t, err, io.ErrUnexpectedEOF)
}

func TestReadHeaderPropagatesReadFault(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("disk read fault")
	f := mocks.NewMockReaderAt(t)
	f.EXPECT().ReadAt(mock.Anything, int64(0)).Return(0, wantErr)

	r := NewReader(f, NewTestPool(4, 1024), ^uint32(0))
	_, err := r.ReadHeader(0)
	require.ErrorIs(t, err, wantErr)
}
