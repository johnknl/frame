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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScanner_ScanSuccess(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "scan-ok.bin")
	file, err := os.Create(path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, file.Close()) })

	payloads := [][]byte{[]byte("a"), []byte("bb"), []byte("ccc")}
	for idx, payload := range payloads {
		fr := NewTestFrame(uint32(idx), payload)
		_, err = file.Write(fr.Header[:])
		require.NoError(t, err)
		_, err = file.Write(payload)
		require.NoError(t, err)
	}

	r := NewReader(file, NewTestPool(4, 1024), ^uint32(0))
	s := NewScanner(r, WithScannerValidator[TestHeader](NewCRC32C[TestHeader]()))

	count := 0
	for s.Scan() {
		fr := s.Frame()
		require.Equal(t, uint32(count), fr.Header.Index())
		require.Equal(t, payloads[count], fr.Payload)
		count++
	}

	require.Equal(t, len(payloads), count)
	require.NoError(t, s.Err())
}

func TestScanner_ScanInvalidIndex(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "scan-index.bin")
	file, err := os.Create(path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, file.Close()) })

	frames := []*Frame[TestHeader]{
		NewTestFrame(0, []byte("a")),
		NewTestFrame(2, []byte("b")),
	}

	for _, fr := range frames {
		_, err = file.Write(fr.Header[:])
		require.NoError(t, err)
		_, err = file.Write(fr.Payload)
		require.NoError(t, err)
	}

	r := NewReader(file, NewTestPool(4, 1024), ^uint32(0))
	s := NewScanner(r, WithScannerValidator[TestHeader](NewCRC32C[TestHeader]()))

	require.True(t, s.Scan())
	require.False(t, s.Scan())
	require.ErrorIs(t, s.Err(), ErrInvalidFrameIndex)
}

func TestScanner_CloseReleasesCurrentFrame(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "scan-close.bin")
	file, err := os.Create(path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, file.Close()) })

	fr := NewTestFrame(0, []byte("a"))
	_, err = file.Write(fr.Header[:])
	require.NoError(t, err)
	_, err = file.Write(fr.Payload)
	require.NoError(t, err)

	r := NewReader(file, NewTestPool(4, 1024), ^uint32(0))
	s := NewScanner(r, WithScannerValidator[TestHeader](NewCRC32C[TestHeader]()))

	require.True(t, s.Scan())
	require.NotNil(t, s.Frame())

	s.Close()
	require.Nil(t, s.Frame())
}

func TestScanner_ScanFromOffsetUsesFirstIndexAsBaseline(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "scan-offset.bin")
	file, err := os.Create(path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, file.Close()) })

	frames := []*Frame[TestHeader]{
		NewTestFrame(0, []byte("a")),
		NewTestFrame(1, []byte("bb")),
		NewTestFrame(2, []byte("ccc")),
	}

	offset := int64(0)
	for _, fr := range frames {
		_, err = file.Write(fr.Header[:])
		require.NoError(t, err)
		_, err = file.Write(fr.Payload)
		require.NoError(t, err)
	}

	offset += int64(TestHeaderSize + len(frames[0].Payload))

	r := NewReader(file, NewTestPool(4, 1024), ^uint32(0))
	s := NewScanner(
		r,
		WithScannerValidator[TestHeader](NewCRC32C[TestHeader]()),
		WithScannerOffset[TestHeader](offset),
		WithScannerIndex[TestHeader](1),
	)

	require.True(t, s.Scan())
	require.Equal(t, uint32(1), s.Frame().Header.Index())
	require.True(t, s.Scan())
	require.Equal(t, uint32(2), s.Frame().Header.Index())
	require.False(t, s.Scan())
	require.NoError(t, s.Err())
}

func TestScanner_ScanCustomStartIndex(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "scan-start-index.bin")
	file, err := os.Create(path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, file.Close()) })

	frames := []*Frame[TestHeader]{
		NewTestFrame(10, []byte("x")),
		NewTestFrame(11, []byte("yy")),
	}

	for _, fr := range frames {
		_, err = file.Write(fr.Header[:])
		require.NoError(t, err)
		_, err = file.Write(fr.Payload)
		require.NoError(t, err)
	}

	r := NewReader(file, NewTestPool(4, 1024), ^uint32(0))
	s := NewScanner(
		r,
		WithScannerValidator[TestHeader](NewCRC32C[TestHeader]()),
		WithScannerIndex[TestHeader](10),
	)

	require.True(t, s.Scan())
	require.Equal(t, uint32(10), s.Frame().Header.Index())
	require.True(t, s.Scan())
	require.Equal(t, uint32(11), s.Frame().Header.Index())
	require.False(t, s.Scan())
	require.NoError(t, s.Err())
}

func TestScanner_Seek(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "scan-seek.bin")
	file, err := os.Create(path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, file.Close()) })

	frames := []*Frame[TestHeader]{
		NewTestFrame(0, []byte("a")),
		NewTestFrame(1, []byte("bb")),
		NewTestFrame(2, []byte("ccc")),
	}

	offset := int64(0)
	for _, fr := range frames {
		_, err = file.Write(fr.Header[:])
		require.NoError(t, err)
		_, err = file.Write(fr.Payload)
		require.NoError(t, err)
	}

	offset += int64(TestHeaderSize + len(frames[0].Payload))

	r := NewReader(file, NewTestPool(4, 1024), ^uint32(0))
	s := NewScanner(r, WithScannerValidator[TestHeader](NewCRC32C[TestHeader]()))

	require.True(t, s.Scan())
	require.Equal(t, uint32(0), s.Frame().Header.Index())

	s.SeekOffset(offset)
	require.NoError(t, s.Err())
	require.True(t, s.Scan())
	require.Equal(t, uint32(1), s.Frame().Header.Index())
}

func TestScanner_SeekIOSeekerStyle(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "scan-seek-whence.bin")
	file, err := os.Create(path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, file.Close()) })

	frames := []*Frame[TestHeader]{
		NewTestFrame(0, []byte("a")),
		NewTestFrame(1, []byte("bb")),
	}

	for _, fr := range frames {
		_, err = file.Write(fr.Header[:])
		require.NoError(t, err)
		_, err = file.Write(fr.Payload)
		require.NoError(t, err)
	}

	r := NewReader(file, NewTestPool(4, 1024), ^uint32(0))
	s := NewScanner(
		r,
		WithScannerValidator[TestHeader](NewCRC32C[TestHeader]()),
		WithScannerOffset[TestHeader](int64(TestHeaderSize+len(frames[0].Payload))),
		WithScannerIndex[TestHeader](1),
	)

	_, err = s.Seek(0, io.SeekStart)
	require.NoError(t, err)
	require.NoError(t, s.SeekIndex(0))
	require.True(t, s.Scan())
	require.Equal(t, uint32(0), s.Frame().Header.Index())

	_, err = s.Seek(1, io.SeekCurrent)
	require.NoError(t, err)
	err = s.SeekIndex(0)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrInvalidFrameIndex) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF))
}

func TestScanner_SeekIndex(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "scan-seek-index.bin")
	file, err := os.Create(path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, file.Close()) })

	frames := []*Frame[TestHeader]{
		NewTestFrame(0, []byte("a")),
		NewTestFrame(1, []byte("bb")),
		NewTestFrame(2, []byte("ccc")),
	}

	for _, fr := range frames {
		_, err = file.Write(fr.Header[:])
		require.NoError(t, err)
		_, err = file.Write(fr.Payload)
		require.NoError(t, err)
	}

	r := NewReader(file, NewTestPool(4, 1024), ^uint32(0))
	s := NewScanner(r, WithScannerValidator[TestHeader](NewCRC32C[TestHeader]()))

	require.NoError(t, s.SeekIndex(2))
	require.True(t, s.Scan())
	require.Equal(t, uint32(2), s.Frame().Header.Index())

	require.ErrorIs(t, s.SeekIndex(9), io.EOF)
}
