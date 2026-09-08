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
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildPayloadBucketCaps(t *testing.T) {
	t.Parallel()

	require.Equal(t, []int{4, 8, 16, 32, 64, 128, 256, 512, 1024}, buildPayloadBucketCaps(4, 1024))
	require.Equal(t, []int{64}, buildPayloadBucketCaps(64, 64))
	require.Equal(t, []int{96, 192, 384, 768, 1024}, buildPayloadBucketCaps(96, 1024))
}

func TestPool_GetSelectsBucketByPayloadSize(t *testing.T) {
	t.Parallel()

	p := NewTestPool(64, 1024)

	f := p.Get(65)
	require.Equal(t, 65, len(f.Payload))
	require.Equal(t, 128, cap(f.Payload))
	f.Return()

	f = p.Get(1024)
	require.Equal(t, 1024, len(f.Payload))
	require.Equal(t, 1024, cap(f.Payload))
	f.Return()
}

func TestPool_GetOverMaxNotRetained(t *testing.T) {
	t.Parallel()

	p := NewTestPool(64, 1024)

	f := p.Get(2048)
	require.Equal(t, 2048, len(f.Payload))
	require.GreaterOrEqual(t, cap(f.Payload), 2048)
	require.Equal(t, -1, p.payloadBucketForCap(cap(f.Payload)))
	f.Return()
}

func TestPool_ReturnReusesFrameFromSameBucket(t *testing.T) {
	t.Parallel()

	p := NewTestPool(64, 1024)

	f := p.Get(200)
	require.Equal(t, 256, cap(f.Payload))
	f.Return()

	g := p.Get(200)
	require.Equal(t, 256, cap(g.Payload))
	g.Return()
}
