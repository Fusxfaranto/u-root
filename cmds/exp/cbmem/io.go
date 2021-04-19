// Copyright 2016-2021 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build linux

package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"syscall"
)

type offsetReader struct {
	name string
	base int64
	r    io.ReaderAt
}

var _ io.ReaderAt = &offsetReader{}

// ReadAt implements io.ReaderAt.
// The offset is adjusted by the base. After that
// it is bytes.Buffer's job to deal with all the
// corner cases.
// Note that this is not a section reader, since the offset
// is maintained. This makes it easy to use ROM addresses
// without adjusting them.
// For one example, the coreboot MEM_CONSOLE tag has an absolute
// address in it. Once a proper offset reader is created,
// that absolute address is used unchanged.
func (o *offsetReader) ReadAt(b []byte, i int64) (int, error) {
	// This is the line that makes it "not a section reader".
	i -= o.base
	n, err := o.r.ReadAt(b, i)
	if err != nil && err != io.EOF {
		return n, fmt.Errorf("Reading at #%x for %d bytes: %v", i, len(b), err)
	}
	return n, err
}

func mapit(f *os.File, addr int64, sz int) (io.ReaderAt, error) {
	ba := (addr >> 12) << 12
	basz := sz + int(addr-ba)
	b, err := syscall.Mmap(int(f.Fd()), ba, basz, syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return nil, fmt.Errorf("mmap %d bytes at %#x: %v", sz, addr, err)
	}
	return bytes.NewReader(b[:sz]), nil
}

func newOffsetReader(f *os.File, off int64, sz int) (*offsetReader, error) {
	r, err := mapit(f, off, sz)
	if err != nil {
		return nil, err
	}
	return &offsetReader{base: off, r: r}, nil
}

// readOneSize reads an entry of any type. This Size variant is for
// the console log only, though we know of no case in which it is
// larger than 1M. We really want the SectionReader as a way to ReadAt
// for the binary.Read. Any meaningful limit will be enforced by the kernel.
func readOneSize(r io.ReaderAt, i interface{}, o int64, n int64) error {
	err := binary.Read(io.NewSectionReader(r, o, n), binary.LittleEndian, i)
	if err != nil {
		return fmt.Errorf("Trying to read section for %T: %v", r, err)
	}
	return nil
}

// readOneSize reads an entry of any type, limited to 64K.
func readOne(r io.ReaderAt, i interface{}, o int64) error {
	return readOneSize(r, i, o, 65536)
}
