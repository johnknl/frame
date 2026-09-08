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

	"github.com/johnknl/frame/internal/testutil"
	"github.com/stretchr/testify/require"
)

const frameFuzzPayloadLimit = 1 << 20

// FuzzReader_Read feeds arbitrary byte streams and offsets to ensure frame reads fail only with expected errors.
func FuzzReader_Read(f *testing.F) {
	f.Add([]byte{}, uint32(0), uint32(0), uint32(0))

	seedHeader := NewTestHeader(1, 0, []byte("ok"))
	f.Add(append(seedHeader[:], []byte("ok")...), uint32(1), uint32(0), uint32(8))

	f.Fuzz(func(t *testing.T, raw []byte, idx uint32, offset uint32, maxLen uint32) {
		// bound and normalize random byte input before persisting it
		raw = frameFuzzNormalizeRaw(raw, idx, offset, maxLen)

		path := filepath.Join(t.TempDir(), "fuzz-frame.bin")
		require.NoError(t, os.WriteFile(path, raw, 0o600))

		file, err := os.Open(path)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, file.Close()) })

		r := NewReader(file, NewTestPool(64, 1<<20), MaxPayloadSize)

		// keep reads within the fuzzed file bounds
		off := frameFuzzNormalizeOffset(offset, len(raw))

		fr, err := r.Read(off)
		if err == nil {
			// successful reads must return a pooled frame
			fr.Return()
			return
		}

		// arbitrary input should fail only with known frame-reader errors
		require.True(t, errors.Is(err, io.EOF) ||
			errors.Is(err, io.ErrUnexpectedEOF) ||
			errors.Is(err, ErrInvalidFrameIndex) ||
			errors.Is(err, ErrInvalidChecksum) ||
			errors.Is(err, ErrInvalidFrameHeader))
	})
}

// FuzzFrame_RoundTrip writes synthesized frames and verifies read/validate
// invariants round-trip.
func FuzzFrame_RoundTrip(f *testing.F) {
	f.Add(uint32(0), []byte(""))
	f.Add(uint32(7), []byte("seed"))

	f.Fuzz(func(t *testing.T, idx uint32, payload []byte) {
		// bound payload size so corpus mutation remains cheap and stable
		payload = testutil.BoundedFuzzBytes(payload, 1<<20, idx)

		// write one synthesized frame for deterministic round-trip checks
		f := NewTestFrame(idx, payload)
		h := f.Header
		path := filepath.Join(t.TempDir(), "roundtrip.bin")
		require.NoError(t, os.WriteFile(path, append(h[:], payload...), 0o600))

		// open the file and read the frame back
		file, err := os.Open(path)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, file.Close()) })

		fr, err := NewReader(file, NewTestPool(64, 1<<20), MaxPayloadSize).Read(0)
		require.NoError(t, err)
		t.Cleanup(fr.Return)

		// decoded frame metadata and payload must match what was written
		require.Equal(t, idx, fr.Header.Index())
		require.Equal(t, payload, fr.Payload)
		crc := NewCRC32C[TestHeader]()
		require.NoError(t, crc.Validate(fr))
		require.Equal(t, int64(TestHeaderSize+len(payload)), fr.Header.NextOffset(0))
	})
}

func frameFuzzNormalizeRaw(raw []byte, idx uint32, offset uint32, maxLen uint32) []byte {
	raw = testutil.BoundedFuzzBytes(raw, frameFuzzPayloadLimit, idx^offset^maxLen)
	if maxLen > frameFuzzPayloadLimit {
		maxLen = frameFuzzPayloadLimit
	}

	if len(raw) <= int(maxLen) {
		return raw
	}

	return testutil.BoundedFuzzBytes(raw, int(maxLen), idx^offset)
}

func frameFuzzNormalizeOffset(offset uint32, rawLen int) int64 {
	off := int64(offset)
	if rawLen > 0 {
		off %= int64(rawLen)
	}

	return off
}
