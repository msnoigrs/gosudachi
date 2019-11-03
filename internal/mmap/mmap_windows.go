// +build windows

package mmap

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

func Mmap(fd *os.File, write bool, offset int64, size int64) ([]byte, error) {
	protect := syscall.PAGE_READONLY
	access := syscall.FILE_MAP_READ

	if write {
		protect = syscall.PAGE_READWRITE
		access = syscall.FILE_MAP_WRITE
	}
	fi, err := fd.Stat()
	if err != nil {
		return nil, err
	}

	if fi.Size() < size {
		if err := fd.Truncate(size); err != nil {
			return nil, fmt.Errorf("truncate: %s", err)
		}
	}

	maxsize := size + offset
	maxsizehi := uint32(maxsize >> 32)
	maxsizelo := uint32(maxsize & 0xffffffff)

	handle, err := syscall.CreateFileMapping(syscall.Handle(fd.Fd()), nil,
		uint32(protect), maxsizehi, maxsizelo, nil)
	if err != nil {
		return nil, os.NewSyscallError("CreateFileMapping", err)
	}

	offsethi := uint32(offset >> 32)
	offsetlo := uint32(offset & 0xffffffff)
	addr, err := syscall.MapViewOfFile(handle, uint32(access), offsethi, offsetlo, uintptr(size))
	if addr == 0 {
		return nil, os.NewSyscallError("MapViewOfFile", err)
	}

	if err := syscall.CloseHandle(syscall.Handle(handle)); err != nil {
		return nil, os.NewSyscallError("CloseHandle", err)
	}

	// Slice memory layout
	// Copied this snippet from golang/sys package
	var sl = struct {
		addr uintptr
		len  int
		cap  int
	}{addr, int(size), int(size)}

	// Use unsafe to turn sl into a []byte
	data := *(*[]byte)(unsafe.Pointer(&sl))

	return data, nil
}

func Munmap(b []byte) error {
	return syscall.UnmapViewOfFile(uintptr(unsafe.Pointer(&b[0])))
}

func Madvise(b []byte, readahead bool) error {
	// Do Nothing. We don't care about this setting on Windows
	return nil
}
