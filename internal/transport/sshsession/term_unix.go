//go:build linux || darwin

package sshsession

import (
	"syscall"
	"unsafe"
)

type termState struct {
	state syscall.Termios
}

func makeRaw(fd uintptr) (*termState, error) {
	oldState, err := getTermios(fd)
	if err != nil {
		return nil, err
	}
	newState := *oldState
	newState.Iflag &^= syscall.IGNBRK | syscall.BRKINT | syscall.PARMRK | syscall.ISTRIP | syscall.INLCR | syscall.IGNCR | syscall.ICRNL | syscall.IXON
	newState.Oflag &^= syscall.OPOST
	newState.Lflag &^= syscall.ECHO | syscall.ECHONL | syscall.ICANON | syscall.ISIG | syscall.IEXTEN
	newState.Cflag &^= syscall.CSIZE | syscall.PARENB
	newState.Cflag |= syscall.CS8
	newState.Cc[syscall.VMIN] = 1
	newState.Cc[syscall.VTIME] = 0
	if err := setTermios(fd, &newState); err != nil {
		return nil, err
	}
	return &termState{state: *oldState}, nil
}

func restore(fd uintptr, state *termState) error {
	return setTermios(fd, &state.state)
}

func getTermios(fd uintptr) (*syscall.Termios, error) {
	var state syscall.Termios
	_, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, fd, uintptr(ioctlReadTermios), uintptr(unsafe.Pointer(&state)), 0, 0, 0)
	if errno != 0 {
		return nil, errno
	}
	return &state, nil
}

func setTermios(fd uintptr, state *syscall.Termios) error {
	_, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, fd, uintptr(ioctlWriteTermios), uintptr(unsafe.Pointer(state)), 0, 0, 0)
	if errno != 0 {
		return errno
	}
	return nil
}
