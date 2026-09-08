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
	"encoding/binary"
	"errors"
	"fmt"
)

const exampleHeaderSize = 16

type exampleHeader [exampleHeaderSize]byte

func (h exampleHeader) PayloadLength() uint32 {
	return binary.BigEndian.Uint32(h[0:4])
}

func (h exampleHeader) Index() uint32 {
	return binary.BigEndian.Uint32(h[4:8])
}

func (h exampleHeader) Checksum() uint32 {
	return binary.BigEndian.Uint32(h[12:16])
}

func (h exampleHeader) ChecksumBytes() []byte {
	return h[:12]
}

func (h *exampleHeader) setChecksum(sum uint32) {
	binary.BigEndian.PutUint32(h[12:16], sum)
}

func ExampleHeader() {
	// A concrete header must implement these methods:
	// PayloadLength() uint32
	// Index() uint32
	// Checksum() uint32
	// ChecksumBytes() []byte
	payload := []byte("hi")
	var h exampleHeader
	binary.BigEndian.PutUint32(h[0:4], uint32(len(payload)))
	binary.BigEndian.PutUint32(h[4:8], 7)

	fmt.Println(h.PayloadLength(), h.Index(), h.Checksum())
	fmt.Println(len(h.ChecksumBytes()))

	// Output:
	// 2 7 0
	// 12
}

func ExampleNewPool() {
	pool := NewPool[exampleHeader](64, 1024)
	payload := []byte("ok")
	var h exampleHeader
	binary.BigEndian.PutUint32(h[0:4], uint32(len(payload)))
	binary.BigEndian.PutUint32(h[4:8], 0)

	f := pool.Get()
	f.Set(h, payload)
	fmt.Println(len(f.Payload), cap(f.Payload) >= 2)
	f.Return()

	g := pool.Get()
	fmt.Println(len(g.Payload))
	g.Return()

	// Output:
	// 2 true
	// 0
}

func ExampleCRC32C_Validate() {
	payload := []byte("hello")
	var h exampleHeader
	binary.BigEndian.PutUint32(h[0:4], uint32(len(payload)))
	binary.BigEndian.PutUint32(h[4:8], 1)
	f := &Frame[exampleHeader]{
		Header:  h,
		Payload: payload,
	}

	crc := NewCRC32C[exampleHeader]()
	h.setChecksum(crc.Sum(f))
	f.Header = h

	fmt.Println(crc.Validate(f) == nil)

	f.Payload[0] = 'H'
	fmt.Println(errors.Is(crc.Validate(f), ErrInvalidChecksum))

	// Output:
	// true
	// true
}

func ExampleScanner() {
	encode := func(index uint32, payload []byte) []byte {
		var h exampleHeader
		binary.BigEndian.PutUint32(h[0:4], uint32(len(payload)))
		binary.BigEndian.PutUint32(h[4:8], index)

		f := &Frame[exampleHeader]{Header: h, Payload: payload}
		crc := NewCRC32C[exampleHeader]()
		h.setChecksum(crc.Sum(f))

		out := make([]byte, 0, exampleHeaderSize+len(payload))
		out = append(out, h[:]...)
		out = append(out, payload...)

		return out
	}

	raw := make([]byte, 0, 2*exampleHeaderSize+3)
	raw = append(raw, encode(0, []byte("a"))...)
	raw = append(raw, encode(1, []byte("bc"))...)

	stream := bytes.NewReader(raw)
	pool := NewPool[exampleHeader](16, 256)
	reader := NewReader(stream, pool, ^uint32(0))

	scanner := NewScanner(
		reader,
		WithScannerValidator[exampleHeader](NewCRC32C[exampleHeader]()),
	)
	defer scanner.Close()

	count := 0
	for scanner.Scan() {
		f := scanner.Frame()
		fmt.Printf("%d:%s\n", f.Header.Index(), string(f.Payload))
		count++
	}

	fmt.Println("err", scanner.Err() == nil, "count", count)

	// Output:
	// 0:a
	// 1:bc
	// err true count 2
}
