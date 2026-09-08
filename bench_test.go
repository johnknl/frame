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
	"bytes"
	"testing"

	"github.com/johnknl/frame/internal/testutil"
)

func BenchmarkCRC32C_Sum(b *testing.B) {
	const size = 1024
	payload := testutil.BenchmarkPayload(size)
	f := NewTestFrame(0, payload)
	crc := NewCRC32C[TestHeader]()

	b.SetBytes(int64(size + TestHeaderSize))
	b.ResetTimer()

	for range b.N {
		_ = crc.Sum(f)
	}
}

func BenchmarkCRC32C_Validate(b *testing.B) {
	const size = 1024
	payload := testutil.BenchmarkPayload(size)
	f := NewTestFrame(0, payload)
	crc := NewCRC32C[TestHeader]()

	b.SetBytes(int64(size + TestHeaderSize))
	b.ResetTimer()

	for range b.N {
		if err := crc.Validate(f); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReader_Read(b *testing.B) {
	b.Run("full_payload", func(b *testing.B) {
		payload := testutil.BenchmarkPayload(1024)
		raw := buildSingleFrameCorpus(0, payload)
		r := bytes.NewReader(raw)
		reader := NewReader(r, NewTestPool(len(payload), len(payload)), MaxPayloadSize)

		b.SetBytes(int64(TestHeaderSize + len(payload)))
		b.ResetTimer()

		for range b.N {
			fr, err := reader.Read(0)
			if err != nil {
				b.Fatal(err)
			}
			fr.Return()
		}
	})

	b.Run("header_only_limit_zero", func(b *testing.B) {
		payload := testutil.BenchmarkPayload(1024)
		raw := buildSingleFrameCorpus(0, payload)
		r := bytes.NewReader(raw)
		reader := NewReader(r, NewTestPool(len(payload), len(payload)), HeadersOnly)

		b.SetBytes(int64(TestHeaderSize))
		b.ResetTimer()

		for range b.N {
			fr, err := reader.Read(0)
			if err != nil {
				b.Fatal(err)
			}
			fr.Return()
		}
	})
}

func BenchmarkScanner_Scan(b *testing.B) {
	b.Run("default", func(b *testing.B) {
		benchScanner(b, 0, false)
	})

	b.Run("with_validation", func(b *testing.B) {
		benchScanner(b, 0, true)
	})

	b.Run("at_offset", func(b *testing.B) {
		benchScanner(b, 256, false)
	})
}

const scannerFrameCount = 1024
const scannerPayloadSize = 128

func benchScanner(b *testing.B, skip int, validate bool) {
	if skip > scannerFrameCount {
		skip = scannerFrameCount
	}

	buf := buildFrameCorpusFrom(scannerFrameCount, scannerPayloadSize, 0)
	startOffset := int64(skip * (TestHeaderSize + scannerPayloadSize))
	stream := bytes.NewReader(buf)
	pool := NewTestPool(scannerPayloadSize, scannerPayloadSize)
	reader := NewReader(stream, pool, MaxPayloadSize)

	var options []ScannerOption[TestHeader]
	options = append(options,
		WithScannerOffset[TestHeader](startOffset),
		WithScannerIndex[TestHeader](uint32(skip)),
	)
	if validate {
		options = append(options, WithScannerValidator[TestHeader](NewCRC32C[TestHeader]()))
	}

	s := NewScanner(reader, options...)

	framesPerScan := int64(scannerFrameCount - skip)
	bytesPerFrame := int64(TestHeaderSize + scannerPayloadSize)
	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		resetScannerState(s, startOffset, uint32(skip))
		for s.Scan() {
		}
		if err := s.Err(); err != nil {
			b.Fatal(err)
		}
	}

	s.Close()

	testutil.ReportDomainMetrics(b, framesPerScan, framesPerScan*bytesPerFrame)
}

func resetScannerState[HT IndexedHeader](s *Scanner[HT], offset int64, index uint32) {
	s.Close()
	s.offset = offset
	s.nextIdx = index
	s.err = nil
}

func buildSingleFrameCorpus(index uint32, payload []byte) []byte {
	f := NewTestFrame(index, payload)
	raw := make([]byte, TestHeaderSize+len(payload))
	for i := range TestHeaderSize {
		raw[i] = f.Header[i]
	}
	for i := range len(payload) {
		raw[TestHeaderSize+i] = payload[i]
	}

	return raw
}

func buildFrameCorpusFrom(frameCount, payloadSize int, startIndex uint32) []byte {
	payload := testutil.BenchmarkPayload(payloadSize)
	buf := make([]byte, frameCount*(TestHeaderSize+payloadSize))
	pos := 0
	for i := range frameCount {
		fr := NewTestFrame(startIndex+uint32(i), payload)
		for j := range TestHeaderSize {
			buf[pos+j] = fr.Header[j]
		}
		pos += TestHeaderSize
		for j := range payloadSize {
			buf[pos+j] = payload[j]
		}
		pos += payloadSize
	}

	return buf
}
