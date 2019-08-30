package binarydist

import (
	"io"

	"github.com/dsnet/compress/bzip2"
)

func newBzip2Writer(w io.Writer) (io.WriteCloser, error) {
	wc, err := bzip2.NewWriter(w, &bzip2.WriterConfig{
		Level: 9,
	})
	if err != nil {
		return nil, err
	}
	return wc, nil
}
