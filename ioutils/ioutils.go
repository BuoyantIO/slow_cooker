package ioutils

import "io"

// Borrowed from https://github.com/docker/docker/blob/master/pkg/ioutils/writers.go

type writeCloserWrapper struct {
	io.Writer
	closer func() error
}

func (r *writeCloserWrapper) Close() error {
	return r.closer()
}

// NewWriteCloserWrapper returns a new io.WriteCloser.
func NewWriteCloserWrapper(r io.Writer, closer func() error) io.WriteCloser {
	return &writeCloserWrapper{
		Writer: r,
		closer: closer,
	}
}
