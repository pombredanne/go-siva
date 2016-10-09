package siva

import (
	"errors"
	"io"
)

var (
	ErrMissingHeader = errors.New("WriteHeader was not called, or already flushed")
	ErrClosedWriter  = errors.New("Writer is closed")
)

// A Writer provides sequential writing of a siva archive
type Writer struct {
	w        *hashedWriter
	index    Index
	current  *IndexEntry
	position uint64
	closed   bool
}

// NewWriter creates a new Writer writing to w.
func NewWriter(w io.Writer) *Writer {
	return &Writer{
		w: newHashedWriter(w),
	}
}

// WriteHeader writes hdr and prepares to accept the file's contents.
func (w *Writer) WriteHeader(h *Header) error {
	if err := w.flushIfPending(); err != nil {
		return err
	}

	w.current = &IndexEntry{
		Header: (*h),
		Start:  w.position,
	}

	w.index = append(w.index, w.current)
	return nil
}

// Write writes to the current entry in the siva archive, WriteHeader should
// called before, if not returns ErrMissingHeader
func (w *Writer) Write(b []byte) (int, error) {
	if w.current == nil {
		return 0, ErrMissingHeader
	}

	n, err := w.w.Write(b)
	w.position += uint64(n)

	return n, err
}

// Flush finishes writing the current file (optional)
func (w *Writer) Flush() error {
	if w.closed {
		return ErrClosedWriter
	}

	if w.current == nil {
		return ErrMissingHeader
	}

	w.current.Size = w.position - w.current.Start
	w.current.CRC32 = w.w.Checkshum()
	w.w.Reset()

	return nil
}

func (w *Writer) flushIfPending() error {
	if w.closed {
		return ErrClosedWriter
	}

	if w.current == nil {
		return nil
	}

	return w.Flush()
}

// Close closes the siva archive, writing the Index footer to the current writer.
func (w *Writer) Close() error {
	defer func() { w.closed = true }()

	if err := w.flushIfPending(); err != nil {
		return err
	}

	err := w.index.WriteTo(w.w)
	if err == ErrEmptyIndex {
		return nil
	}

	return err
}
