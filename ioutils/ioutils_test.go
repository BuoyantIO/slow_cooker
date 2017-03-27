package ioutils

import (
	"bytes"
	"io"
	"testing"
)

// Borrowed from https://github.com/docker/docker/blob/master/pkg/ioutils/writers_test.go

type nopWriteCloser struct {
	io.Writer
}

func (w *nopWriteCloser) Close() error { return nil }

// NopWriteCloser returns a nopWriteCloser.
func NopWriteCloser(w io.Writer) io.WriteCloser {
	return &nopWriteCloser{w}
}

func TestWriteCloserWrapperClose(t *testing.T) {
	called := false
	writer := bytes.NewBuffer([]byte{})
	wrapper := NewWriteCloserWrapper(writer, func() error {
		called = true
		return nil
	})
	if err := wrapper.Close(); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatalf("writeCloserWrapper should have call the anonymous function.")
	}
}

func TestNopWriteCloser(t *testing.T) {
	writer := bytes.NewBuffer([]byte{})
	wrapper := NopWriteCloser(writer)
	if err := wrapper.Close(); err != nil {
		t.Fatal("NopWriteCloser always return nil on Close.")
	}

}
