//go:build linux || darwin

package sshsession

import "os"

func readTerminalKey(_ *os.File) (byte, bool, error) {
	return 0, false, nil
}
