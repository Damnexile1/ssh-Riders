//go:build darwin

package sshsession

const (
	ioctlReadTermios  = 0x40487413
	ioctlWriteTermios = 0x80487414
)
