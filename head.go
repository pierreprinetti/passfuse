package main

import (
	"bytes"
	"io"
)

func FirstLineWriter(w io.Writer) io.Writer {
	return &firstLineWriter{w: w}
}

type firstLineWriter struct {
	done bool
	w    io.Writer
}

func (h *firstLineWriter) Write(p []byte) (int, error) {
	if h.done {
		return len(p), nil
	}
	if i := bytes.IndexRune(p, '\n'); i >= 0 {
		written, err := h.w.Write(p[:i])
		if err != nil || written != i {
			return written, err
		}

		h.done = true
		return len(p), nil
	}
	return h.w.Write(p)
}
