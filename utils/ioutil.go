// Copyright (c) 2015-2021 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

// Package ioutil implements some I/O utility functions which are not covered
// by the standard library.
package utils

import (
	"archive/tar"
	"context"
	"fmt"
	"github.com/klauspost/compress/gzip"
	"io"
	"os"
	"sync"
	"time"
)

// WriteOnCloser implements io.WriteCloser and always
// executes at least one write operation if it is closed.
//
// This can be useful within the context of HTTP. At least
// one write operation must happen to send the HTTP headers
// to the peer.
type WriteOnCloser struct {
	io.Writer
	hasWritten bool
}

func (w *WriteOnCloser) Write(p []byte) (int, error) {
	w.hasWritten = true
	return w.Writer.Write(p)
}

// Close closes the WriteOnCloser. It behaves like io.Closer.
func (w *WriteOnCloser) Close() error {
	if !w.hasWritten {
		_, err := w.Write(nil)
		if err != nil {
			return err
		}
	}
	if closer, ok := w.Writer.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// HasWritten returns true if at least one write operation was performed.
func (w *WriteOnCloser) HasWritten() bool { return w.hasWritten }

// WriteOnClose takes an io.Writer and returns an ioutil.WriteOnCloser.
func WriteOnClose(w io.Writer) *WriteOnCloser {
	return &WriteOnCloser{w, false}
}

type ioret struct {
	n   int
	err error
}

// DeadlineWriter deadline writer with context
type DeadlineWriter struct {
	io.WriteCloser
	timeout time.Duration
	err     error
}

// NewDeadlineWriter wraps a writer to make it respect given deadline
// value per Write(). If there is a blocking write, the returned Writer
// will return whenever the timer hits (the return values are n=0
// and err=context.Canceled.)
func NewDeadlineWriter(w io.WriteCloser, timeout time.Duration) io.WriteCloser {
	return &DeadlineWriter{WriteCloser: w, timeout: timeout}
}

func (w *DeadlineWriter) Write(buf []byte) (int, error) {
	if w.err != nil {
		return 0, w.err
	}

	c := make(chan ioret, 1)
	t := time.NewTimer(w.timeout)
	defer t.Stop()

	go func() {
		n, err := w.WriteCloser.Write(buf)
		c <- ioret{n, err}
		close(c)
	}()

	select {
	case r := <-c:
		w.err = r.err
		return r.n, r.err
	case <-t.C:
		w.err = context.Canceled
		return 0, context.Canceled
	}
}

// Close closer interface to close the underlying closer
func (w *DeadlineWriter) Close() error {
	return w.WriteCloser.Close()
}

// LimitWriter implements io.WriteCloser.
//
// This is implemented such that we want to restrict
// an enscapsulated writer upto a certain length
// and skip a certain number of bytes.
type LimitWriter struct {
	io.Writer
	skipBytes int64
	wLimit    int64
}

// Write implements the io.Writer interface limiting upto
// configured length, also skips the first N bytes.
func (w *LimitWriter) Write(p []byte) (n int, err error) {
	n = len(p)
	var n1 int
	if w.skipBytes > 0 {
		if w.skipBytes >= int64(len(p)) {
			w.skipBytes -= int64(len(p))
			return n, nil
		}
		p = p[w.skipBytes:]
		w.skipBytes = 0
	}
	if w.wLimit == 0 {
		return n, nil
	}
	if w.wLimit < int64(len(p)) {
		n1, err = w.Writer.Write(p[:w.wLimit])
		w.wLimit -= int64(n1)
		return n, err
	}
	n1, err = w.Writer.Write(p)
	w.wLimit -= int64(n1)
	return n, err
}

// Close closes the LimitWriter. It behaves like io.Closer.
func (w *LimitWriter) Close() error {
	if closer, ok := w.Writer.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// LimitedWriter takes an io.Writer and returns an ioutil.LimitWriter.
func LimitedWriter(w io.Writer, skipBytes int64, limit int64) *LimitWriter {
	return &LimitWriter{w, skipBytes, limit}
}

type nopCloser struct {
	io.Writer
}

func (nopCloser) Close() error { return nil }

// NopCloser returns a WriteCloser with a no-op Close method wrapping
// the provided Writer w.
func NopCloser(w io.Writer) io.WriteCloser {
	return nopCloser{w}
}

// SkipReader skips a given number of bytes and then returns all
// remaining data.
type SkipReader struct {
	io.Reader

	skipCount int64
}

func (s *SkipReader) Read(p []byte) (int, error) {
	l := int64(len(p))
	if l == 0 {
		return 0, nil
	}
	for s.skipCount > 0 {
		if l > s.skipCount {
			l = s.skipCount
		}
		n, err := s.Reader.Read(p[:l])
		if err != nil {
			return 0, err
		}
		s.skipCount -= int64(n)
	}
	return s.Reader.Read(p)
}

// NewSkipReader - creates a SkipReader
func NewSkipReader(r io.Reader, n int64) io.Reader {
	return &SkipReader{r, n}
}

var copyBufPool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, 32*1024)
		return &b
	},
}

// Copy is exactly like io.Copy but with re-usable buffers.
func Copy(dst io.Writer, src io.Reader) (written int64, err error) {
	bufp := copyBufPool.Get().(*[]byte)
	buf := *bufp
	defer copyBufPool.Put(bufp)

	return io.CopyBuffer(dst, src, buf)
}

// SameFile returns if the files are same.
func SameFile(fi1, fi2 os.FileInfo) bool {
	if !os.SameFile(fi1, fi2) {
		return false
	}
	if !fi1.ModTime().Equal(fi2.ModTime()) {
		return false
	}
	if fi1.Mode() != fi2.Mode() {
		return false
	}
	return fi1.Size() == fi2.Size()
}

// DirectioAlignSize - DirectIO alignment needs to be 4K. Defined here as
// directio.AlignSize is defined as 0 in MacOS causing divide by 0 error.
const DirectioAlignSize = 4096

type TarFile struct {
	Path string
}
type FileItem struct {
	Name string
	Size int64
}

func (f *TarFile) ListFiles() []FileItem {
	res := []FileItem{}
	tarFile, _ := os.Open(f.Path)
	defer tarFile.Close()
	r, err := gzip.NewReader(tarFile)
	var reader io.Reader
	if err != nil {
		f, _ := os.Open(f.Path)
		defer f.Close()
		reader = f
	} else {
		reader = r
	}
	tarReader := tar.NewReader(reader)
	for {
		header, err := tarReader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
		}
		if header.Typeflag == tar.TypeDir {
			continue
		}
		res = append(res, FileItem{
			Name: header.Name,
			Size: header.Size,
		})
	}
	return res
}

func (f *TarFile) ExtractFile(fileName string, dst io.Writer) error {
	tarFile, _ := os.Open(f.Path)
	defer tarFile.Close()
	r, err := gzip.NewReader(tarFile)
	var reader io.Reader
	if err != nil {
		tarFile.Close()
		f, _ := os.Open(f.Path)
		reader = f
		defer f.Close()
	} else {
		reader = r
	}
	tarReader := tar.NewReader(reader)
	for {
		header, err := tarReader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if header.Name == fileName {
			io.Copy(dst, tarReader)
			return nil
		}
	}
	return fmt.Errorf("not found file:%v", fileName)
}

func (f *TarFile) ExtractFileTo(fileNames []string, cal func(fileName string, reader io.Reader)) error {
	tarFile, _ := os.Open(f.Path)
	defer tarFile.Close()
	r, err := gzip.NewReader(tarFile)
	var reader io.Reader
	if err != nil {
		tarFile.Close()
		f, _ := os.Open(f.Path)
		reader = f
		defer f.Close()
	} else {
		reader = r
	}
	tarReader := tar.NewReader(reader)
	for {
		header, err := tarReader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		for _, name := range fileNames {
			if header.Name == name {
				cal(name, tarReader)
				return nil
			}
		}

	}
	return nil
}
