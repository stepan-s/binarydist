package binarydist

import (
	"bytes"
	"math/rand"
	"testing"
)

func TestRoundtrip(t *testing.T) {
	tests := []struct {
		name     string
		old, new []byte
	}{
		{"empty_to_empty", nil, nil},
		{"empty_to_data", nil, []byte("hello world")},
		{"data_to_empty", []byte("hello world"), nil},
		{"identical", []byte("hello world"), []byte("hello world")},
		{"small_change", []byte("hello world"), []byte("hello World!")},
		{"testdata", mustReadAll(mustOpen("testdata/sample.old")), mustReadAll(mustOpen("testdata/sample.new"))},
		{"random_1k", randBytes(1000, 1), randBytes(1000, 2)},
		{"random_10k", randBytes(10000, 3), randBytes(10000, 4)},
		{"random_similar", randSimilarOld(10000, 5), randSimilarNew(10000, 5)},
		{"all_zeros", bytes.Repeat([]byte{0}, 1000), bytes.Repeat([]byte{0}, 1000)},
		{"repeated_pattern", bytes.Repeat([]byte("abcdef"), 500), bytes.Repeat([]byte("abcdef"), 600)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patch, err := diffBytes(tt.old, tt.new)
			if err != nil {
				t.Fatalf("diff: %v", err)
			}

			got, err := patchBytes(tt.old, patch)
			if err != nil {
				t.Fatalf("patch: %v", err)
			}

			if !bytes.Equal(got, tt.new) {
				t.Fatalf("roundtrip mismatch: got %d bytes, want %d bytes", len(got), len(tt.new))
			}
		})
	}
}

func patchBytes(old, patchData []byte) ([]byte, error) {
	var out bytes.Buffer
	err := Patch(bytes.NewReader(old), &out, bytes.NewReader(patchData))
	if err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func randBytes(n int, seed int64) []byte {
	r := rand.New(rand.NewSource(seed))
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(r.Intn(256))
	}
	return b
}

func randSimilarOld(n int, seed int64) []byte {
	r := rand.New(rand.NewSource(seed))
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(r.Intn(256))
	}
	return b
}

func randSimilarNew(n int, seed int64) []byte {
	old := randSimilarOld(n, seed)
	r := rand.New(rand.NewSource(seed + 1000))
	for i := 0; i < n/10; i++ {
		pos := r.Intn(n)
		old[pos] = byte(r.Intn(256))
	}
	return old
}
