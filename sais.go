package binarydist

// bitset is a compact bit array using 1 bit per element instead of 1 byte.
type bitset struct {
	data []uint64
}

func newBitset(n int) bitset {
	return bitset{data: make([]uint64, (n+63)/64)}
}

func (b bitset) get(i int) bool {
	return b.data[i/64]&(1<<uint(i%64)) != 0
}

func (b bitset) set(i int) {
	b.data[i/64] |= 1 << uint(i%64)
}

func (b bitset) clear(i int) {
	b.data[i/64] &^= 1 << uint(i%64)
}

// buildSuffixArray constructs the suffix array for the given byte slice
// using the SA-IS (Suffix Array by Induced Sorting) algorithm in O(n) time.
// Uses int32 to reduce memory — supports files up to 2 GB.
// The returned array has length len(data)+1, including the sentinel
// (empty suffix) at position 0.
func buildSuffixArray(data []byte) []int32 {
	n := len(data)
	if n == 0 {
		return []int32{0}
	}

	// Convert bytes to int32 with sentinel (0) that is smaller than all byte values (1-256)
	s := make([]int32, n+1)
	for i, b := range data {
		s[i] = int32(b) + 1
	}
	// s[n] = 0 — sentinel

	sa := make([]int32, n+1)
	saisSort(s, sa, 257) // alphabet: 0 (sentinel) + 1..256 (byte values)
	return sa
}

func saisSort(s, sa []int32, k int) {
	n := len(s)

	if n <= 2 {
		if n == 1 {
			sa[0] = 0
		} else {
			sa[0] = 1
			sa[1] = 0
		}
		return
	}

	// Classify each suffix as S-type (true) or L-type (false)
	t := newBitset(n)
	t.set(n - 1) // sentinel is S-type
	for i := n - 2; i >= 0; i-- {
		if s[i] < s[i+1] || (s[i] == s[i+1] && t.get(i+1)) {
			t.set(i)
		}
	}

	// Compute bucket sizes
	bucketSizes := make([]int32, k)
	for _, c := range s {
		bucketSizes[c]++
	}

	bkt := make([]int32, k)

	getBucketStarts := func() {
		var sum int32
		for i, sz := range bucketSizes {
			bkt[i] = sum
			sum += sz
		}
	}
	getBucketEnds := func() {
		var sum int32
		for i, sz := range bucketSizes {
			sum += sz
			bkt[i] = sum - 1
		}
	}

	// isLMS returns true if position i is a Left-Most S-type suffix
	isLMS := func(i int) bool {
		return i > 0 && t.get(i) && !t.get(i-1)
	}

	// Initialize SA to -1
	for i := range sa {
		sa[i] = -1
	}

	// Step 1: Place LMS suffixes at the end of their buckets (right to left)
	getBucketEnds()
	for i := n - 1; i > 0; i-- {
		if isLMS(i) {
			sa[bkt[s[i]]] = int32(i)
			bkt[s[i]]--
		}
	}

	// Step 2: Induce L-type suffixes (left to right)
	getBucketStarts()
	for i := 0; i < n; i++ {
		if sa[i] > 0 {
			j := sa[i] - 1
			if !t.get(int(j)) {
				sa[bkt[s[j]]] = j
				bkt[s[j]]++
			}
		}
	}

	// Step 3: Induce S-type suffixes (right to left)
	getBucketEnds()
	for i := n - 1; i >= 0; i-- {
		if sa[i] > 0 {
			j := sa[i] - 1
			if t.get(int(j)) {
				sa[bkt[s[j]]] = j
				bkt[s[j]]--
			}
		}
	}

	// Collect sorted LMS suffixes and assign names
	lmsCount := 0
	for i := 1; i < n; i++ {
		if isLMS(i) {
			lmsCount++
		}
	}

	sortedLMS := make([]int32, 0, lmsCount)
	for i := 0; i < n; i++ {
		if isLMS(int(sa[i])) {
			sortedLMS = append(sortedLMS, sa[i])
		}
	}

	// Assign names to LMS substrings
	var name int32
	var prev int32 = -1
	names := make([]int32, n)
	for i := range names {
		names[i] = -1
	}

	for _, pos := range sortedLMS {
		diff := prev == -1
		if !diff {
			// Compare LMS substrings at prev and pos
			for d := int32(0); ; d++ {
				if prev+d >= int32(n) || pos+d >= int32(n) ||
					s[prev+d] != s[pos+d] || t.get(int(prev+d)) != t.get(int(pos+d)) {
					diff = true
					break
				}
				if d > 0 && (isLMS(int(prev+d)) || isLMS(int(pos+d))) {
					break
				}
			}
		}
		if diff {
			name++
		}
		prev = pos
		names[pos] = name - 1
	}

	if name < int32(lmsCount) {
		// Not all names are unique — recurse
		s1 := make([]int32, lmsCount)
		j := 0
		for i := 0; i < n; i++ {
			if names[i] >= 0 {
				s1[j] = names[i]
				j++
			}
		}

		sa1 := make([]int32, lmsCount)
		saisSort(s1, sa1, int(name))

		// Map back to original positions
		lmsPos := make([]int32, lmsCount)
		j = 0
		for i := 1; i < n; i++ {
			if isLMS(i) {
				lmsPos[j] = int32(i)
				j++
			}
		}

		for i := 0; i < lmsCount; i++ {
			sortedLMS[i] = lmsPos[sa1[i]]
		}
	}

	// Final induced sort with correctly sorted LMS suffixes
	for i := range sa {
		sa[i] = -1
	}

	getBucketEnds()
	for i := lmsCount - 1; i >= 0; i-- {
		j := sortedLMS[i]
		sa[bkt[s[j]]] = j
		bkt[s[j]]--
	}

	getBucketStarts()
	for i := 0; i < n; i++ {
		if sa[i] > 0 {
			j := sa[i] - 1
			if !t.get(int(j)) {
				sa[bkt[s[j]]] = j
				bkt[s[j]]++
			}
		}
	}

	getBucketEnds()
	for i := n - 1; i >= 0; i-- {
		if sa[i] > 0 {
			j := sa[i] - 1
			if t.get(int(j)) {
				sa[bkt[s[j]]] = j
				bkt[s[j]]--
			}
		}
	}
}
