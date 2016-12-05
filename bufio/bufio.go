// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package bufio implements buffered I/O.  It wraps an io.Reader or io.Writer
// object, creating another object (Reader or Writer) that also implements
// the interface but provides buffering and some help for textual I/O.
package bufio

import (
	"errors"
	"io"
)

const (
	defaultBufSize = 4096
)

var (
	ErrInvalidUnreadByte = errors.New("bufio: invalid use of UnreadByte")
	ErrInvalidUnreadRune = errors.New("bufio: invalid use of UnreadRune")
	ErrBufferFull        = errors.New("bufio: buffer full")
	ErrNegativeCount     = errors.New("bufio: negative count")
)

// Buffered input.

// Reader implements buffering for an io.Reader object.
type Reader struct {
	buf  []byte
	rd   io.Reader // reader provided by the client
	r, w int       // buf read and write positions
	err  error
}

const minReadBufferSize = 16
const maxConsecutiveEmptyReads = 100

// NewReaderSize returns a new Reader whose buffer has at least the specified
// size. If the argument io.Reader is already a Reader with large enough
// size, it returns the underlying Reader.
func NewReaderSize(rd io.Reader, size int) *Reader {
	// Is it already a Reader?
	b, ok := rd.(*Reader)
	if ok && len(b.buf) >= size {
		return b
	}
	if size < minReadBufferSize {
		size = minReadBufferSize
	}
	r := new(Reader)
	r.reset(make([]byte, size), rd)
	return r
}

// NewReader returns a new Reader whose buffer has the default size.
func NewReader(rd io.Reader) *Reader {
	return NewReaderSize(rd, defaultBufSize)
}

// Reset discards any buffered data, resets all state, and switches
// the buffered reader to read from r.
func (b *Reader) Reset(r io.Reader) {
	b.reset(b.buf, r)
}

// ResetBuffer discards any buffered data, resets all state, and switches
// the buffered reader to read from r.
func (b *Reader) ResetBuffer(r io.Reader, buf []byte) {
	b.reset(buf, r)
}

func (b *Reader) reset(buf []byte, r io.Reader) {
	*b = Reader{
		buf: buf,
		rd:  r,
	}
}

var errNegativeRead = errors.New("bufio: reader returned negative count from Read")

// fill reads a new chunk into the buffer.
func (b *Reader) fill() {
	// Slide existing data to beginning.
	if b.r > 0 {
		copy(b.buf, b.buf[b.r:b.w])
		b.w -= b.r
		b.r = 0
	}

	if b.w >= len(b.buf) {
		panic("bufio: tried to fill full buffer")
	}

	// Read new data: try a limited number of times.
	for i := maxConsecutiveEmptyReads; i > 0; i-- {
		n, err := b.rd.Read(b.buf[b.w:])
		if n < 0 {
			panic(errNegativeRead)
		}
		b.w += n
		if err != nil {
			b.err = err
			return
		}
		if n > 0 {
			return
		}
	}
	b.err = io.ErrNoProgress
}

func (b *Reader) readErr() error {
	err := b.err
	b.err = nil
	return err
}

// Peek returns the next n bytes without advancing the reader. The bytes stop
// being valid at the next read call. If Peek returns fewer than n bytes, it
// also returns an error explaining why the read is short. The error is
// ErrBufferFull if n is larger than b's buffer size.
func (b *Reader) Peek(n int) ([]byte, error) {
	if n < 0 {
		return nil, ErrNegativeCount
	}
	if n > len(b.buf) {
		return nil, ErrBufferFull
	}
	// 0 <= n <= len(b.buf)
	for b.w-b.r < n && b.err == nil {
		b.fill() // b.w-b.r < len(b.buf) => buffer is not full
	}

	var err error
	if avail := b.w - b.r; avail < n {
		// not enough data in buffer
		n = avail
		err = b.readErr()
		if err == nil {
			err = ErrBufferFull
		}
	}
	return b.buf[b.r : b.r+n], err
}

// Pop returns the next n bytes with advancing the reader. The bytes stop
// being valid at the next read call. If Pop returns fewer than n bytes, it
// also returns an error explaining why the read is short. The error is
// ErrBufferFull if n is larger than b's buffer size.
func (b *Reader) Pop(n int) ([]byte, error) {
	if d, err := b.Peek(n); err == nil {
		b.r += n
		return d, err
	} else {
		return nil, err
	}
}

// Discard skips the next n bytes, returning the number of bytes discarded.
//
// If Discard skips fewer than n bytes, it also returns an error.
// If 0 <= n <= b.Buffered(), Discard is guaranteed to succeed without
// reading from the underlying io.Reader.
func (b *Reader) Discard(n int) (discarded int, err error) {
	if n < 0 {
		return 0, ErrNegativeCount
	}
	if n == 0 {
		return
	}
	remain := n
	for {
		skip := b.Buffered()
		if skip == 0 {
			b.fill()
			skip = b.Buffered()
		}
		if skip > remain {
			skip = remain
		}
		b.r += skip
		remain -= skip
		if remain == 0 {
			return n, nil
		}
		if b.err != nil {
			return n - remain, b.readErr()
		}
	}
}

// Read reads data into p.
// It returns the number of bytes read into p.
// It calls Read at most once on the underlying Reader,
// hence n may be less than len(p).
// At EOF, the count will be zero and err will be io.EOF.
func (b *Reader) Read(p []byte) (n int, err error) {
	n = len(p)
	if n == 0 {
		return 0, b.readErr()
	}
	if b.r == b.w {
		if b.err != nil {
			return 0, b.readErr()
		}
		if len(p) >= len(b.buf) {
			// Large read, empty buffer.
			// Read directly into p to avoid copy.
			n, b.err = b.rd.Read(p)
			if n < 0 {
				panic(errNegativeRead)
			}
			return n, b.readErr()
		}
		b.fill() // buffer is empty
		if b.r == b.w {
			return 0, b.readErr()
		}
	}

	// copy as much as we can
	n = copy(p, b.buf[b.r:b.w])
	b.r += n
	return n, nil
}

// Buffered returns the number of bytes that can be read from the current buffer.
func (b *Reader) Buffered() int { return b.w - b.r }

// buffered output



// =================================
func (reader *Reader) ReadBytes(n int) (b []byte, err error) {
	b = make([]byte, n)
	_, reader.err = io.ReadFull(reader.rd, b)
	return b, reader.err
}

func (reader *Reader) ReadString(n int) (string, error) {
	if b, err := reader.ReadBytes(n); err == nil {
		return string(b), nil
	} else {
		return "", err
	}
}

//func (reader *Reader) seek(n int) (b []byte) {
//	if reader.err == nil {
//		b = reader.buf[:n]
//		_, reader.err = io.ReadFull(reader.rd, b)
//		if reader.err == nil {
//			return
//		}
//	}
//	return zero[:n]
//}

func (reader *Reader) ReadByte() (byte, error) {
	if byteReader, ok := reader.rd.(io.ByteReader); ok {
		return byteReader.ReadByte()
	}
	if b, err := reader.Pop(1); err == nil {
		return byte(b[0]), nil
	} else {
		return 0, err
	}
}

func (reader *Reader) ReadUint8() (uint8, error) {
	if b, err := reader.Pop(1); err == nil {
		return uint8(b[0]), nil
	} else {
		return 0, err
	}
}

func (reader *Reader) ReadUint16BE() (uint16, error) {
	if b, err := reader.Pop(2); err == nil {
		return GetUint16BE(b), nil
	} else {
		return 0, err
	}
}

func (reader *Reader) ReadUint16LE() (uint16, error) {
	if b, err := reader.Pop(2); err == nil {
		return GetUint16LE(b), nil
	} else {
		return 0, err
	}
}


func (reader *Reader) ReadUint32BE() (uint32, error) {
	if b, err := reader.Pop(4); err == nil {
		return GetUint32BE(b), nil
	} else {
		return 0, err
	}
}

func (reader *Reader) ReadUint32LE() (uint32, error) {
	if b, err := reader.Pop(4); err == nil {
		return GetUint32LE(b), nil
	} else {
		return 0, err
	}
}

func (reader *Reader) ReadUint64BE() (uint64, error) {
	if b, err := reader.Pop(8); err == nil {
		return GetUint64BE(b), nil
	} else {
		return 0, err
	}

}

func (reader *Reader) ReadUint64LE() (uint64, error) {
	if b, err := reader.Pop(8); err == nil {
		return GetUint64LE(b), nil
	} else {
		return 0, err
	}
}

func (reader *Reader) ReadFloat32BE() (float32, error) {
	if b, err := reader.Pop(4); err == nil {
		return GetFloat32BE(b), nil
	} else {
		return 0, err
	}
}

func (reader *Reader) ReadFloat32LE() (float32, error) {
	if b, err := reader.Pop(4); err == nil {
		return GetFloat32LE(b), nil
	} else {
		return 0, err
	}
}

func (reader *Reader) ReadFloat64BE() (float64, error) {
	if b, err := reader.Pop(8); err == nil {
		return GetFloat64BE(b), nil
	} else {
		return 0, err
	}
}

func (reader *Reader) ReadFloat64LE() (float64, error) {
	if b, err := reader.Pop(8); err == nil {
		return GetFloat64LE(b), nil
	} else {
		return 0, err
	}
}

func (reader *Reader) ReadInt8() (int8, error)  {
	if i, err := reader.ReadUint8(); err == nil {
		return int8(i), nil
	} else {
		return -1, err
	}
}

func (reader *Reader) ReadInt16BE() (int16, error) {
	if i, err := reader.ReadUint16BE(); err == nil {
		return int16(i), nil
	} else {
		return -1, err
	}
}

func (reader *Reader) ReadInt16LE() (int16, error) {
	if i, err := reader.ReadUint16LE(); err == nil {
		return int16(i), nil
	} else {
		return -1, err
	}
}

func (reader *Reader) ReadInt32BE() (int32, error) {
	if i, err := reader.ReadUint32BE(); err == nil {
		return int32(i), nil
	} else {
		return -1, err
	}
}

func (reader *Reader) ReadInt32LE() (int32, error) {
	if i, err := reader.ReadUint32LE(); err == nil {
		return int32(i), nil
	} else {
		return -1, err
	}
}

func (reader *Reader) ReadInt64BE() (int64, error) {
	if i, err := reader.ReadUint64BE(); err == nil {
		return int64(i), nil
	} else {
		return -1, err
	}
}

func (reader *Reader) ReadInt64LE() (int64, error) {
	if i, err := reader.ReadUint64LE(); err == nil {
		return int64(i), nil
	} else {
		return -1, err
	}
}


// ================================

// Writer implements buffering for an io.Writer object.
// If an error occurs writing to a Writer, no more data will be
// accepted and all subsequent writes will return the error.
// After all data has been written, the client should call the
// Flush method to guarantee all data has been forwarded to
// the underlying io.Writer.
type Writer struct {
	err error
	buf []byte
	n   int
	wr  io.Writer
}

// NewWriterSize returns a new Writer whose buffer has at least the specified
// size. If the argument io.Writer is already a Writer with large enough
// size, it returns the underlying Writer.
func NewWriterSize(w io.Writer, size int) *Writer {
	// Is it already a Writer?
	b, ok := w.(*Writer)
	if ok && len(b.buf) >= size {
		return b
	}
	if size <= 0 {
		size = defaultBufSize
	}
	return &Writer{
		buf: make([]byte, size),
		wr:  w,
	}
}

// NewWriter returns a new Writer whose buffer has the default size.
func NewWriter(w io.Writer) *Writer {
	return NewWriterSize(w, defaultBufSize)
}

// Reset discards any unflushed buffered data, clears any error, and
// resets b to write its output to w.
func (b *Writer) Reset(w io.Writer) {
	b.err = nil
	b.n = 0
	b.wr = w
}

// ResetBuffer discards any unflushed buffered data, clears any error, and
// resets b to write its output to w.
func (b *Writer) ResetBuffer(w io.Writer, buf []byte) {
	b.buf = buf
	b.err = nil
	b.n = 0
	b.wr = w
}

// Flush writes any buffered data to the underlying io.Writer.
func (b *Writer) Flush() error {
	err := b.flush()
	return err
}

func (b *Writer) flush() error {
	if b.err != nil {
		return b.err
	}
	if b.n == 0 {
		return nil
	}
	n, err := b.wr.Write(b.buf[0:b.n])
	if n < b.n && err == nil {
		err = io.ErrShortWrite
	}
	if err != nil {
		if n > 0 && n < b.n {
			copy(b.buf[0:b.n-n], b.buf[n:b.n])
		}
		b.n -= n
		b.err = err
		return err
	}
	b.n = 0
	return nil
}

// Available returns how many bytes are unused in the buffer.
func (b *Writer) Available() int { return len(b.buf) - b.n }

// Buffered returns the number of bytes that have been written into the current buffer.
func (b *Writer) Buffered() int { return b.n }

// Write writes the contents of p into the buffer.
// It returns the number of bytes written.
// If nn < len(p), it also returns an error explaining
// why the write is short.
func (b *Writer) Write(p []byte) (nn int, err error) {
	for len(p) > b.Available() && b.err == nil {
		var n int
		if b.Buffered() == 0 {
			// Large write, empty buffer.
			// Write directly from p to avoid copy.
			n, b.err = b.wr.Write(p)
		} else {
			n = copy(b.buf[b.n:], p)
			b.n += n
			b.flush()
		}
		nn += n
		p = p[n:]
	}
	if b.err != nil {
		return nn, b.err
	}
	n := copy(b.buf[b.n:], p)
	b.n += n
	nn += n
	return nn, nil
}

// Peek returns the next n bytes with advancing the writer. The bytes stop
// being used at the next write call. If Peek returns fewer than n bytes, it
// also returns an error explaining why the read is short. The error is
// ErrBufferFull if n is larger than b's buffer size.
func (b *Writer) Peek(n int) ([]byte, error) {
	if n < 0 {
		return nil, ErrNegativeCount
	}
	if n > len(b.buf) {
		return nil, ErrBufferFull
	}
	for b.Available() < n && b.err == nil {
		b.flush()
	}
	if b.err != nil {
		return nil, b.err
	}
	d := b.buf[b.n : b.n+n]
	b.n += n
	return d, nil
}


//================================
func (writer *Writer) WriteBytes(b []byte) (int, error) {
	return writer.Write(b)
}

func (writer *Writer) WriteString(s string) (int, error) {
	return writer.WriteBytes([]byte(s))
}

func (writer *Writer) WriteUint8(v uint8) (int, error) {
	if b, err := writer.Peek(1); err == nil {
		b[0] = v
		return 1, nil
	} else {
		return -1, err
	}
}

func (writer *Writer) WriteUint16BE(v uint16) (int, error) {
	if b, err := writer.Peek(2); err == nil {
		PutUint16BE(b, v)
		return 2, nil
	} else {
		return -1, err
	}
}

func (writer *Writer) WriteUint16LE(v uint16) (int, error) {
	if b, err := writer.Peek(2); err == nil {
		PutUint16LE(b, v)
		return 2, nil
	} else {
		return -1, err
	}
}

func (writer *Writer) WriteUint32BE(v uint32) (int, error) {
	if b, err := writer.Peek(4); err == nil {
		PutUint32BE(b, v)
		return 4, nil
	} else {
		return -1, err
	}
}

func (writer *Writer) WriteUint32LE(v uint32) (int, error) {
	if b, err := writer.Peek(4); err == nil {
		PutUint32LE(b, v)
		return 4, nil
	} else {
		return -1, err
	}
}

func (writer *Writer) WriteUint64BE(v uint64) (int, error) {
	if b, err := writer.Peek(8); err == nil {
		PutUint64BE(b, v)
		return 8, nil
	} else {
		return -1, err
	}
}

func (writer *Writer) WriteUint64LE(v uint64) (int, error) {
	if b, err := writer.Peek(8); err == nil {
		PutUint64LE(b, v)
		return 8, nil
	} else {
		return -1, err
	}
}

func (writer *Writer) WriteFloat32BE(v float32) (int, error) {
	if b, err := writer.Peek(4); err == nil {
		PutFloat32BE(b, v)
		return 4, nil
	} else {
		return -1, err
	}
}

func (writer *Writer) WriteFloat32LE(v float32) (int, error) {
	if b, err := writer.Peek(4); err == nil {
		PutFloat32LE(b, v)
		return 4, nil
	} else {
		return -1, err
	}
}


func (writer *Writer) WriteFloat64BE(v float64) (int, error) {
	if b, err := writer.Peek(8); err == nil {
		PutFloat64BE(b, v)
		return 8, nil
	} else {
		return -1, err
	}
}

func (writer *Writer) WriteFloat64LE(v float64) (int, error) {
	if b, err := writer.Peek(8); err == nil {
		PutFloat64LE(b, v)
		return 8, nil
	} else {
		return -1, err
	}
}

func (writer *Writer) WriteInt8(v int8)  (int, error)    { return writer.WriteUint8(uint8(v)) }
func (writer *Writer) WriteInt16BE(v int16) (int, error) { return writer.WriteUint16BE(uint16(v)) }
func (writer *Writer) WriteInt16LE(v int16) (int, error) { return writer.WriteUint16LE(uint16(v)) }
func (writer *Writer) WriteInt32BE(v int32) (int, error) { return writer.WriteUint32BE(uint32(v)) }
func (writer *Writer) WriteInt32LE(v int32) (int, error) { return writer.WriteUint32LE(uint32(v)) }
func (writer *Writer) WriteInt64BE(v int64) (int, error) { return writer.WriteUint64BE(uint64(v)) }
func (writer *Writer) WriteInt64LE(v int64) (int, error) { return writer.WriteUint64LE(uint64(v)) }
func (writer *Writer) WriteUintBE(v uint)   (int, error) { return writer.WriteUint64BE(uint64(v)) }
func (writer *Writer) WriteUintLE(v uint)   (int, error) { return writer.WriteUint64LE(uint64(v)) }
//================================
