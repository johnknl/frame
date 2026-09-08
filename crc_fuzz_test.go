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
	"testing"

	"github.com/johnknl/frame/internal/testutil"
	"github.com/stretchr/testify/require"
)

// FuzzHeader_Validate mutates encoded header fields and payload bytes to
// validate checksum/length enforcement.
func FuzzHeader_Validate(f *testing.F) {
	f.Add([]byte{}, uint32(0), uint32(0))
	f.Add([]byte{1}, uint32(1), uint32(1))
	f.Add(make([]byte, 4096), uint32(2), uint32(4096))

	f.Fuzz(func(t *testing.T, payload []byte, idx uint32, declaredLen uint32) {
		// bound payload size to keep fuzz data properly distributed
		payload = testutil.BoundedFuzzBytes(payload, 1<<20, idx^declaredLen)

		crc := NewCRC32C[TestHeader]()

		f := NewTestFrame(idx, payload)

		// freshly generated headers must validate against their source payload
		require.NoError(t, crc.Validate(f))

		// mutating the declared length should fail when it no longer matches payload
		mutatedHeader := f.Header[:]
		binary.BigEndian.PutUint32(mutatedHeader[0:4], declaredLen)
		f.Header = TestHeader(mutatedHeader)

		err := crc.Validate(f)

		if declaredLen != uint32(len(payload)) {
			require.Error(t, err)
		}

		// mutating payload bytes should fail checksum validation
		if len(payload) > 0 {
			// flip one bit so checksum validation will fail
			mutatedPayload := append([]byte(nil), payload...)
			mutatedPayload[0] ^= 0x01
			f.Payload = mutatedPayload

			require.Error(t, crc.Validate(f))
		}
	})
}
