package sshsession

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/example/ssh-riders/internal/domain"
)

type Session interface {
	ID() string
	RemoteAddr() string
	ReadLine(ctx context.Context) (string, error)
	WriteFrame(frame domain.RenderFrame) error
	Close() error
}

type LocalTerminalSession struct {
	id         string
	remoteAddr string
	r          *bufio.Reader
	w          io.Writer
}

func NewLocalTerminalSession(id string, r io.Reader, w io.Writer, remoteAddr string) *LocalTerminalSession {
	return &LocalTerminalSession{id: id, remoteAddr: remoteAddr, r: bufio.NewReader(r), w: w}
}

func (s *LocalTerminalSession) ID() string         { return s.id }
func (s *LocalTerminalSession) RemoteAddr() string { return s.remoteAddr }
func (s *LocalTerminalSession) ReadLine(ctx context.Context) (string, error) {
	type result struct {
		line string
		err  error
	}
	ch := make(chan result, 1)
	go func() { line, err := s.r.ReadString('\n'); ch <- result{line: strings.TrimSpace(line), err: err} }()
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case res := <-ch:
		return res.line, res.err
	}
}
func (s *LocalTerminalSession) WriteFrame(frame domain.RenderFrame) error {
	_, err := fmt.Fprintf(s.w, "\x1b[2J\x1b[H%s\n", strings.Join(frame.Lines, "\n"))
	return err
}
func (s *LocalTerminalSession) Close() error {
	_, err := fmt.Fprintf(s.w, "\nSession closed at %s\n", time.Now().Format(time.RFC3339))
	return err
}
