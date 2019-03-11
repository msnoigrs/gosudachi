// +build !windows

package mmap

import (
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

func Mmap(fd *os.File, writable bool, offset int64, size int64) ([]byte, error) {
	mtype := unix.PROT_READ
	if writable {
		mtype |= unix.PROT_WRITE
	}
	return unix.Mmap(int(fd.Fd()), offset, int(size), mtype, unix.MAP_SHARED)
}

func Munmap(b []byte) error {
	return unix.Munmap(b)
}

func Madvise(b []byte, readahead bool) error {
	flags := unix.MADV_NORMAL
	if !readahead {
		flags = unix.MADV_RANDOM
	}
	return madvise(b, flags)
}

// This is required because the unix package does not support the madvise system call on OS X
func madvise(b []byte, advice int) (err error) {
	_, _, e1 := syscall.Syscall(syscall.SYS_MADVISE, uintptr(unsafe.Pointer(&b[0])),
		uintptr(len(b)), uintptr(advice))
	if e1 != 0 {
		err = e1
	}
	return
}
