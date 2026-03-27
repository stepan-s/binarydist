package binarydist

import (
	"bytes"
	"crypto/rand"
	"testing"
)

var sortT = [][]byte{
	mustRandBytes(1000),
	mustReadAll(mustOpen("test.old")),
	[]byte("abcdefabcdef"),
}

func TestSuffixArray(t *testing.T) {
	for _, s := range sortT {
		I := buildSuffixArray(s)
		for i := 1; i < len(I); i++ {
			if bytes.Compare(s[I[i-1]:], s[I[i]:]) > 0 {
				t.Fatalf("unsorted at %d", i)
			}
		}
	}
}

func mustRandBytes(n int) []byte {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return b
}
