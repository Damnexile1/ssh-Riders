//go:build !linux && !darwin

package sshsession

type termState struct{}

func makeRaw(_ uintptr) (*termState, error) {
	return &termState{}, nil
}

func restore(_ uintptr, _ *termState) error {
	return nil
}
