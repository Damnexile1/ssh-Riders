//go:build !linux && !darwin && !windows

package sshsession

import "os"

type termState struct{}

func makeRaw(_ uintptr) (*termState, error) {
	return &termState{}, nil
}

func restore(_ uintptr, _ *termState) error {
	return nil
}

func readTerminalKey(_ *os.File) (byte, bool, error) {
	return 0, false, nil
}
