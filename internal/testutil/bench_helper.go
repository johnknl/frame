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

package testutil

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

const (
	// OneKiB is the number of bytes in a kibibyte (1024 bytes).
	OneKiB = 1 << 10

	// OneMiB is the number of bytes in a mebibyte (1024 kibibytes).
	OneMiB = 1 << 20
)

// BenchmarkPayload generates a byte slice of the specified size for benchmarking purposes.
func BenchmarkPayload(size int) []byte {
	p := make([]byte, size)
	for i := range p {
		p[i] = byte(i)
	}
	return p
}

// ReportDomainMetrics reports the metrics for a benchmark, including operations per second,
// records per second, and megabytes per second.
func ReportDomainMetrics(b *testing.B, recordsPerOp int64, bytesPerOp int64) {
	b.Helper()
	elapsed := b.Elapsed().Seconds()
	if elapsed <= 0 {
		return
	}

	opsPerSec := float64(b.N) / elapsed
	recordsPerSec := float64(recordsPerOp) * opsPerSec
	mbPerSec := float64(bytesPerOp) * opsPerSec / (1024.0 * 1024.0)
	b.ReportMetric(recordsPerSec, "records/s")
	b.ReportMetric(mbPerSec, "MB/s")
	if recordsPerOp > 0 {
		b.ReportMetric(float64(bytesPerOp)/float64(recordsPerOp), "bytes/record")
	}
}

// BenchmarkLabelInt generates a benchmark label string in the format "prefix=n" for reporting purposes.
func BenchmarkLabelInt(prefix string, n int) string {
	return prefix + "=" + strconv.Itoa(n)
}

// RawFileForBench creates a temporary raw file for benchmarking purposes and returns the file handle.
func RawFileForBench(b *testing.B) *os.File {
	b.Helper()
	path := filepath.Join(b.TempDir(), "raw.bin")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0o600)
	if err != nil {
		b.Fatalf("open raw file: %v", err)
	}
	b.Cleanup(func() { _ = f.Close() })
	return f
}
