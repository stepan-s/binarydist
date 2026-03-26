package binarydist

// buildSuffixArray constructs the suffix array for the given byte slice
// using the SA-IS (Suffix Array by Induced Sorting) algorithm in O(n) time.
// The returned array has length len(data)+1, including the sentinel
// (empty suffix) at position 0.
func buildSuffixArray(data []byte) []int {
	n := len(data)
	if n == 0 {
		return []int{0}
	}

	// Convert bytes to ints with sentinel (0) that is smaller than all byte values (1-256)
	s := make([]int, n+1)
	for i, b := range data {
		s[i] = int(b) + 1
	}
	// s[n] = 0 — sentinel

	sa := make([]int, n+1)
	saisSort(s, sa, 257) // alphabet: 0 (sentinel) + 1..256 (byte values)
	return sa
}

func saisSort(s, sa []int, k int) {
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
	t := make([]bool, n)
	t[n-1] = true // sentinel is S-type
	for i := n - 2; i >= 0; i-- {
		if s[i] < s[i+1] || (s[i] == s[i+1] && t[i+1]) {
			t[i] = true
		}
	}

	// Compute bucket sizes
	bucketSizes := make([]int, k)
	for _, c := range s {
		bucketSizes[c]++
	}

	getBucketStarts := func(dst []int) {
		sum := 0
		for i, sz := range bucketSizes {
			dst[i] = sum
			sum += sz
		}
	}
	getBucketEnds := func(dst []int) {
		sum := 0
		for i, sz := range bucketSizes {
			sum += sz
			dst[i] = sum - 1
		}
	}

	bkt := make([]int, k)

	// isLMS returns true if position i is a Left-Most S-type suffix
	isLMS := func(i int) bool {
		return i > 0 && t[i] && !t[i-1]
	}

	// Initialize SA to -1
	for i := range sa {
		sa[i] = -1
	}

	// Step 1: Place LMS suffixes at the end of their buckets (right to left)
	getBucketEnds(bkt)
	for i := n - 1; i > 0; i-- {
		if isLMS(i) {
			sa[bkt[s[i]]] = i
			bkt[s[i]]--
		}
	}

	// Step 2: Induce L-type suffixes (left to right)
	getBucketStarts(bkt)
	// The sentinel position (n-1) has s[n-1]=0, bucket starts at 0.
	// sa[0] should be set to n-1 if not already, but it was placed as LMS above if applicable.
	// Actually, position n-1 IS LMS (sentinel), placed in step 1.
	for i := 0; i < n; i++ {
		if sa[i] > 0 {
			j := sa[i] - 1
			if !t[j] {
				sa[bkt[s[j]]] = j
				bkt[s[j]]++
			}
		}
	}

	// Step 3: Induce S-type suffixes (right to left)
	getBucketEnds(bkt)
	for i := n - 1; i >= 0; i-- {
		if sa[i] > 0 {
			j := sa[i] - 1
			if t[j] {
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

	sortedLMS := make([]int, 0, lmsCount)
	for i := 0; i < n; i++ {
		if isLMS(sa[i]) {
			sortedLMS = append(sortedLMS, sa[i])
		}
	}

	// Assign names to LMS substrings
	name := 0
	prev := -1
	names := make([]int, n)
	for i := range names {
		names[i] = -1
	}

	for _, pos := range sortedLMS {
		diff := prev == -1
		if !diff {
			// Compare LMS substrings at prev and pos
			for d := 0; ; d++ {
				if prev+d >= n || pos+d >= n ||
					s[prev+d] != s[pos+d] || t[prev+d] != t[pos+d] {
					diff = true
					break
				}
				if d > 0 && (isLMS(prev+d) || isLMS(pos+d)) {
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

	if name < lmsCount {
		// Not all names are unique — recurse
		s1 := make([]int, lmsCount)
		j := 0
		for i := 0; i < n; i++ {
			if names[i] >= 0 {
				s1[j] = names[i]
				j++
			}
		}

		sa1 := make([]int, lmsCount)
		saisSort(s1, sa1, name)

		// Map back to original positions
		lmsPos := make([]int, lmsCount)
		j = 0
		for i := 1; i < n; i++ {
			if isLMS(i) {
				lmsPos[j] = i
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

	getBucketEnds(bkt)
	for i := lmsCount - 1; i >= 0; i-- {
		j := sortedLMS[i]
		sa[bkt[s[j]]] = j
		bkt[s[j]]--
	}

	getBucketStarts(bkt)
	for i := 0; i < n; i++ {
		if sa[i] > 0 {
			j := sa[i] - 1
			if !t[j] {
				sa[bkt[s[j]]] = j
				bkt[s[j]]++
			}
		}
	}

	getBucketEnds(bkt)
	for i := n - 1; i >= 0; i-- {
		if sa[i] > 0 {
			j := sa[i] - 1
			if t[j] {
				sa[bkt[s[j]]] = j
				bkt[s[j]]--
			}
		}
	}
}
