package binarydist

import (
	"bytes"
	"compress/bzip2"
	"encoding/binary"
	"errors"
	"io"
	"io/ioutil"
)

var ErrCorrupt = errors.New("corrupt patch")

// Patch applies patch to old, according to the bspatch algorithm,
// and writes the result to new (compatibility version).
func Patch(old io.Reader, new io.Writer, patch io.Reader) error {
	obuf, err := ioutil.ReadAll(old)
	if err != nil {
		return err
	}

	return PatchRS(bytes.NewReader(obuf), new, patch)
}

type SumReader struct {
	a io.Reader
	b io.Reader
}

func (r *SumReader) Read(b []byte) (n int, err error) {
	n, err = r.a.Read(b)
	if err != nil {
		return 0, err
	}

	buf := make([]byte, n)
	j, err := r.b.Read(buf)
	if err != nil {
		return 0, err
	}
	if j != n {
		return 0, ErrCorrupt
	}

	for i := 0; i < n; i++ {
		b[i] += buf[i]
	}
	return n, nil
}

// Patch applies patch to old, according to the bspatch algorithm,
// and writes the result to new (optimized version for low memory usage, especially for big files).
func PatchRS(old io.ReadSeeker, new io.Writer, patch io.Reader) error {
	var hdr header
	err := binary.Read(patch, signMagLittleEndian{}, &hdr)
	if err != nil {
		return err
	}
	if hdr.Magic != magic {
		return ErrCorrupt
	}
	if hdr.CtrlLen < 0 || hdr.DiffLen < 0 || hdr.NewSize < 0 {
		return ErrCorrupt
	}

	ctrlbuf := make([]byte, hdr.CtrlLen)
	_, err = io.ReadFull(patch, ctrlbuf)
	if err != nil {
		return err
	}
	cpfbz2 := bzip2.NewReader(bytes.NewReader(ctrlbuf))

	diffbuf := make([]byte, hdr.DiffLen)
	_, err = io.ReadFull(patch, diffbuf)
	if err != nil {
		return err
	}
	dpfbz2 := bzip2.NewReader(bytes.NewReader(diffbuf))

	// The entire rest of the file is the extra block.
	epfbz2 := bzip2.NewReader(patch)

	var oldpos, newpos int64
	for newpos < hdr.NewSize {
		var ctrl struct{ Add, Copy, Seek int64 }
		err = binary.Read(cpfbz2, signMagLittleEndian{}, &ctrl)
		if err != nil {
			return err
		}

		// 1. Copy from patch + old to new
		if ctrl.Add > 0 {
			// Sanity-check
			if newpos+ctrl.Add > hdr.NewSize {
				return ErrCorrupt
			}

			_, err = io.CopyN(new, &SumReader{dpfbz2, old}, ctrl.Add)
			if err != nil {
				return ErrCorrupt
			}

			// Adjust pointers
			newpos += ctrl.Add
			oldpos += ctrl.Add
		}

		// 2. Copy extra from patch to new
		if ctrl.Copy > 0 {
			// Sanity-check
			if newpos+ctrl.Copy > hdr.NewSize {
				return ErrCorrupt
			}

			_, err = io.CopyN(new, epfbz2, ctrl.Copy)
			if err != nil {
				return ErrCorrupt
			}

			// Adjust pointers
			newpos += ctrl.Copy
		}

		// Adjust pointers
		oldpos += ctrl.Seek

		_, err := old.Seek(oldpos, io.SeekStart)
		if err != nil {
			return ErrCorrupt
		}
	}

	return nil
}
