package binarydist

import (
	"bytes"
	"encoding/binary"
	"io"
	"io/ioutil"
)

func matchlen(a, b []byte) (i int) {
	for i < len(a) && i < len(b) && a[i] == b[i] {
		i++
	}
	return i
}

func search(I []int32, obuf, nbuf []byte, st, en int) (pos, n int) {
	for en-st >= 2 {
		pivot := st + (en-st)/2
		if bytes.Compare(obuf[I[pivot]:], nbuf) < 0 {
			st = pivot
		} else {
			en = pivot
		}
	}

	x := matchlen(obuf[I[st]:], nbuf)
	y := matchlen(obuf[I[en]:], nbuf)
	if x > y {
		return int(I[st]), x
	}
	return int(I[en]), y
}

// Diff computes the difference between old and new, according to the bsdiff
// algorithm, and writes the result to patch.
func Diff(old, new io.Reader, patch io.Writer) error {
	obuf, err := ioutil.ReadAll(old)
	if err != nil {
		return err
	}

	nbuf, err := ioutil.ReadAll(new)
	if err != nil {
		return err
	}

	pbuf, err := diffBytes(obuf, nbuf)
	if err != nil {
		return err
	}

	_, err = patch.Write(pbuf)
	return err
}

func diffBytes(obuf, nbuf []byte) ([]byte, error) {
	var patch seekBuffer
	err := diff(obuf, nbuf, &patch)
	if err != nil {
		return nil, err
	}
	return patch.buf, nil
}

func diff(obuf, nbuf []byte, patch io.WriteSeeker) error {
	var lenf int
	I := buildSuffixArray(obuf)
	var db, eb []byte

	var hdr header
	hdr.Magic = magic
	hdr.NewSize = int64(len(nbuf))
	err := binary.Write(patch, signMagLittleEndian{}, &hdr)
	if err != nil {
		return err
	}

	// Compute the differences, writing ctrl as we go
	pfbz2, err := newBzip2Writer(patch)
	if err != nil {
		return err
	}
	var scan, pos, length int
	var lastscan, lastpos, lastoffset int
	for scan < len(nbuf) {
		var oldscore int
		scan += length
		for scsc := scan; scan < len(nbuf); scan++ {
			pos, length = search(I, obuf, nbuf[scan:], 0, len(obuf))

			for ; scsc < scan+length; scsc++ {
				if scsc+lastoffset < len(obuf) &&
					obuf[scsc+lastoffset] == nbuf[scsc] {
					oldscore++
				}
			}

			if (length == oldscore && length != 0) || length > oldscore+8 {
				break
			}

			if scan+lastoffset < len(obuf) && obuf[scan+lastoffset] == nbuf[scan] {
				oldscore--
			}
		}

		if length != oldscore || scan == len(nbuf) {
			var s, Sf int
			lenf = 0
			for i := 0; lastscan+i < scan && lastpos+i < len(obuf); {
				if obuf[lastpos+i] == nbuf[lastscan+i] {
					s++
				}
				i++
				if s*2-i > Sf*2-lenf {
					Sf = s
					lenf = i
				}
			}

			lenb := 0
			if scan < len(nbuf) {
				var s, Sb int
				for i := 1; (scan >= lastscan+i) && (pos >= i); i++ {
					if obuf[pos-i] == nbuf[scan-i] {
						s++
					}
					if s*2-i > Sb*2-lenb {
						Sb = s
						lenb = i
					}
				}
			}

			if lastscan+lenf > scan-lenb {
				overlap := (lastscan + lenf) - (scan - lenb)
				s := 0
				Ss := 0
				lens := 0
				for i := 0; i < overlap; i++ {
					if nbuf[lastscan+lenf-overlap+i] == obuf[lastpos+lenf-overlap+i] {
						s++
					}
					if nbuf[scan-lenb+i] == obuf[pos-lenb+i] {
						s--
					}
					if s > Ss {
						Ss = s
						lens = i + 1
					}
				}

				lenf += lens - overlap
				lenb -= lens
			}

			for i := 0; i < lenf; i++ {
				db = append(db, nbuf[lastscan+i]-obuf[lastpos+i])
			}
			for i := 0; i < (scan-lenb)-(lastscan+lenf); i++ {
				eb = append(eb, nbuf[lastscan+lenf+i])
			}

			var buf [8]byte

			signMagLittleEndian{}.PutUint64(buf[:], uint64(int64(lenf)))
			_, err = pfbz2.Write(buf[:])
			if err != nil {
				pfbz2.Close()
				return err
			}

			val := (scan - lenb) - (lastscan + lenf)
			signMagLittleEndian{}.PutUint64(buf[:], uint64(int64(val)))
			_, err = pfbz2.Write(buf[:])
			if err != nil {
				pfbz2.Close()
				return err
			}

			val = (pos - lenb) - (lastpos + lenf)
			signMagLittleEndian{}.PutUint64(buf[:], uint64(int64(val)))
			_, err = pfbz2.Write(buf[:])
			if err != nil {
				pfbz2.Close()
				return err
			}

			lastscan = scan - lenb
			lastpos = pos - lenb
			lastoffset = pos - scan
		}
	}
	err = pfbz2.Close()
	if err != nil {
		return err
	}

	// Compute size of compressed ctrl data
	l64, err := patch.Seek(0, 1)
	if err != nil {
		return err
	}
	hdr.CtrlLen = int64(l64 - 32)

	// Write compressed diff data
	pfbz2, err = newBzip2Writer(patch)
	if err != nil {
		return err
	}
	n, err := pfbz2.Write(db)
	if err != nil {
		pfbz2.Close()
		return err
	}
	if n != len(db) {
		pfbz2.Close()
		return io.ErrShortWrite
	}
	err = pfbz2.Close()
	if err != nil {
		return err
	}

	// Compute size of compressed diff data
	n64, err := patch.Seek(0, 1)
	if err != nil {
		return err
	}
	hdr.DiffLen = n64 - l64

	// Write compressed extra data
	pfbz2, err = newBzip2Writer(patch)
	if err != nil {
		return err
	}
	n, err = pfbz2.Write(eb)
	if err != nil {
		pfbz2.Close()
		return err
	}
	if n != len(eb) {
		pfbz2.Close()
		return io.ErrShortWrite
	}
	err = pfbz2.Close()
	if err != nil {
		return err
	}

	// Seek to the beginning, write the header, and close the file
	_, err = patch.Seek(0, 0)
	if err != nil {
		return err
	}
	err = binary.Write(patch, signMagLittleEndian{}, &hdr)
	if err != nil {
		return err
	}
	return nil
}
