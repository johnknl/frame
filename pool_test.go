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

func TestNewDefaultRetentionPolicy(t *testing.T) {
	t.Parallel()

	pol := NewDefaultRetentionPolicy(1024)

	require.Equal(t, 3, pol.Warmup)
	require.Equal(t, 256, pol.Threshold)
	require.Equal(t, 0.5, pol.Ratio)
	require.Equal(t, 1024, pol.Max)
	require.Equal(t, 512, pol.Default)
}

func TestRetentionPolicy_Retainable(t *testing.T) {
	t.Parallel()

	pol := &RetentionPolicy{Max: 64}

	require.True(t, pol.Retainable(0))
	require.True(t, pol.Retainable(64))
	require.False(t, pol.Retainable(65))
}

func TestRetentionPolicy_Reusable(t *testing.T) {
	t.Parallel()

	pol := &RetentionPolicy{Threshold: 4, Ratio: 0.25}

	tests := []struct {
		name     string
		capacity int
		need     int
		want     bool
	}{
		{name: "capacity smaller than need", capacity: 7, need: 8, want: false},
		{name: "capacity delta within threshold", capacity: 12, need: 8, want: true},
		{name: "capacity delta above threshold but ratio within bound", capacity: 12, need: 9, want: true},
		{name: "capacity delta above threshold and ratio above bound", capacity: 16, need: 8, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, pol.Reusable(tt.capacity, tt.need))
		})
	}
}

func TestPool_Get_UsesReusablePolicy(t *testing.T) {
	t.Parallel()

	pol := &RetentionPolicy{
		Warmup:    0,
		Threshold: 4,
		Ratio:     0.25,
		Max:       64,
		Default:   16,
	}

	p := NewPool[TestHeader](pol)
	f := p.Get(8)

	require.NotNil(t, f)
	require.Equal(t, 8, len(f.Payload))
	require.Nil(t, f.pool)
}

func TestPool_Get_RetainsWhenReusable(t *testing.T) {
	t.Parallel()

	pol := &RetentionPolicy{
		Warmup:    0,
		Threshold: 16,
		Ratio:     0.25,
		Max:       64,
		Default:   16,
	}

	p := NewPool[TestHeader](pol)
	f := p.Get(8)

	require.NotNil(t, f)
	require.Equal(t, 8, len(f.Payload))
	require.Equal(t, p, f.pool)
}
