//go:build linux && cgo

package ptyutil

/*
#define _XOPEN_SOURCE 600
#include <fcntl.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/ioctl.h>
*/
import "C"
import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"unsafe"
)

func openpty(rows, cols uint16) (int, int, error) {
	master := C.posix_openpt(C.O_RDWR | C.O_NOCTTY)
	if master < 0 {
		return 0, 0, fmt.Errorf("posix_openpt failed")
	}
	if C.grantpt(master) != 0 || C.unlockpt(master) != 0 {
		C.close(master)
		return 0, 0, fmt.Errorf("grantpt/unlockpt failed")
	}
	name := C.GoString(C.ptsname(master))
	slave, err := syscall.Open(name, syscall.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		C.close(master)
		return 0, 0, err
	}
	var win C.struct_winsize
	win.ws_row = C.ushort(rows)
	win.ws_col = C.ushort(cols)
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(master), uintptr(C.TIOCSWINSZ), uintptr(unsafe.Pointer(&win)))
	if errno != 0 {
		C.close(master)
		syscall.Close(slave)
		return 0, 0, errno
	}
	return int(master), slave, nil
}

// StartWithSize starts cmd on a new PTY.
func StartWithSize(cmd *exec.Cmd, rows, cols uint16) (*os.File, error) {
	master, slave, err := openpty(rows, cols)
	if err != nil {
		return nil, err
	}
	return startWithFds(cmd, master, slave)
}

// Setsize updates PTY window size.
func Setsize(f *os.File, rows, cols uint16) error {
	var win C.struct_winsize
	win.ws_row = C.ushort(rows)
	win.ws_col = C.ushort(cols)
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(C.TIOCSWINSZ), uintptr(unsafe.Pointer(&win)))
	if errno != 0 {
		return errno
	}
	return nil
}
