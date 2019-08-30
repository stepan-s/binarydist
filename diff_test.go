package binarydist

import (
	"bytes"
	"crypto/sha256"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"
)

var diffT = []struct {
	old *os.File
	new *os.File
}{
	{
		old: mustWriteRandFile("test.old", 1e3, 1),
		new: mustWriteRandFile("test.new", 1e3, 2),
	},
	{
		old: mustOpen("testdata/sample.old"),
		new: mustOpen("testdata/sample.new"),
	},
}

func fatalForTest(t *testing.T, err error, msg string) {
	t.Helper()
	if err != nil {
		t.Fatal("failed to", msg, err)
	} else if testing.Verbose() {
		t.Log("OK:", msg)
	}
}

func TestDiff(t *testing.T) {
	for _, s := range diffT {
		patchFile, err := ioutil.TempFile("/tmp", "bspatch.")
		fatalForTest(t, err, "create patch file")

		newFile, err := ioutil.TempFile("/tmp", "bspatch.new.")
		fatalForTest(t, err, "create output file")

		newFileName := newFile.Name()
		fatalForTest(t, newFile.Close(), "close output file")

		err = Diff(s.old, s.new, patchFile)
		fatalForTest(t, err, "compute diff")
		fatalForTest(t, patchFile.Close(), "close patch file")

		hash := sha256.New()
		_, err = s.new.Seek(0, 0)
		fatalForTest(t, err, "seek to start of expected output")
		_, err = io.Copy(hash, s.new)
		fatalForTest(t, err, "compute expected hash")
		expectSum := hash.Sum(nil)

		cmd := exec.Command("bspatch", s.old.Name(), newFileName, patchFile.Name())
		cmd.Stderr = os.Stderr

		fatalForTest(t, cmd.Run(), "execute bspatch")

		newFile, err = os.Open(newFileName)
		fatalForTest(t, err, "open output file")
		hash.Reset()
		_, err = io.Copy(hash, newFile)
		fatalForTest(t, err, "hash contents of output file")

		outSum := hash.Sum(nil)

		if !bytes.Equal(expectSum, outSum) {
			t.Errorf("the patched output file %q didn't match the expected output file %q",
				newFile.Name(), s.new.Name())
		}
	}
}
