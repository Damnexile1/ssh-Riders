//go:build windows

package sshsession

import (
	"fmt"
	"os"
	"syscall"
)

type termState struct{}

var (
	msvcrtDLL  = syscall.NewLazyDLL("msvcrt.dll")
	getwchProc = msvcrtDLL.NewProc("_getwch")
)

func makeRaw(_ uintptr) (*termState, error) {
	return &termState{}, nil
}

func restore(_ uintptr, _ *termState) error {
	return nil
}

func readTerminalKey(_ *os.File) (byte, bool, error) {
	r1, _, callErr := getwchProc.Call()
	if r1 == 0 {
		if callErr != syscall.Errno(0) {
			return 0, true, callErr
		}
		return 0, true, fmt.Errorf("received empty console key")
	}
	code := rune(r1)
	if code == 0 || code == 224 {
		r2, _, callErr := getwchProc.Call()
		if callErr != syscall.Errno(0) && callErr != nil {
			return 0, true, callErr
		}
		switch rune(r2) {
		case 72:
			return 'w', true, nil
		case 75:
			return 'a', true, nil
		case 77:
			return 'd', true, nil
		case 80:
			return 's', true, nil
		default:
			return 0, true, nil
		}
	}
	return byte(code), true, nil
}
