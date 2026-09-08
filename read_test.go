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
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReader_Read(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "borrowed.bin")
	f, err := os.Create(path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, f.Close()) })

	payload := []byte("borrow")
	h := NewTestHeader(0, 0, payload)
	_, err = f.Write(h[:])
	require.NoError(t, err)
	_, err = f.Write(payload)
	require.NoError(t, err)

	r := NewReader(f, NewTestPool(4, 1024), MaxPayloadSize)
	b, err := r.Read(0)
	require.NoError(t, err)
	require.Equal(t, uint32(0), b.Header.Index())
	require.Equal(t, payload, b.Payload)
	b.Return()
}

func TestReader_ReadLimitZeroReturnsEmptyPayload(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "borrowed-zero-limit.bin")
	f, err := os.Create(path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, f.Close()) })

	payload := []byte("borrow")
	h := NewTestHeader(0, 0, payload)
	_, err = f.Write(h[:])
	require.NoError(t, err)
	_, err = f.Write(payload)
	require.NoError(t, err)

	r := NewReader(f, NewTestPool(4, 1024), 0)
	b, err := r.Read(0)
	require.NoError(t, err)
	require.Equal(t, uint32(0), b.Header.Index())
	require.Empty(t, b.Payload)
	b.Return()
}

func TestReadAtEOFMapping(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "eof.bin")
	f, err := os.Create(path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, f.Close()) })

	_, err = f.Write([]byte("abc"))
	require.NoError(t, err)

	dst := make([]byte, 5)
	err = readAt(f, dst, 0)
	require.ErrorIs(t, err, io.ErrUnexpectedEOF)
}
