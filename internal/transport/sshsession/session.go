package sshsession

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/example/ssh-riders/internal/domain"
)

type Session interface {
	ID() string
	RemoteAddr() string
	ReadLine(ctx context.Context) (string, error)
	ReadKey(ctx context.Context) (byte, error)
	EnableGameMode() error
	WriteFrame(frame domain.RenderFrame) error
	Close() error
}

type LocalTerminalSession struct {
	id           string
	remoteAddr   string
	r            *bufio.Reader
	w            io.Writer
	file         *os.File
	mu           sync.Mutex
	rawEnabled   bool
	restoreState *termState
}

func NewLocalTerminalSession(id string, r io.Reader, w io.Writer, remoteAddr string) *LocalTerminalSession {
	sess := &LocalTerminalSession{id: id, remoteAddr: remoteAddr, r: bufio.NewReader(r), w: w}
	if file, ok := r.(*os.File); ok {
		sess.file = file
	}
	return sess
}

func (s *LocalTerminalSession) ID() string         { return s.id }
func (s *LocalTerminalSession) RemoteAddr() string { return s.remoteAddr }

func (s *LocalTerminalSession) ReadLine(ctx context.Context) (string, error) {
	type result struct {
		line string
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		line, err := s.r.ReadString('\n')
		ch <- result{line: strings.TrimSpace(line), err: err}
	}()
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case res := <-ch:
		return res.line, res.err
	}
}

func (s *LocalTerminalSession) ReadKey(ctx context.Context) (byte, error) {
	type result struct {
		b   byte
		err error
	}
	ch := make(chan result, 1)
	go func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		b, err := s.r.ReadByte()
		ch <- result{b: b, err: err}
	}()
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case res := <-ch:
		return res.b, res.err
	}
}

func (s *LocalTerminalSession) EnableGameMode() error {
	if s.file == nil {
		return nil
	}
	state, err := makeRaw(s.file.Fd())
	if err != nil {
		return err
	}
	s.restoreState = state
	s.rawEnabled = true
	_, err = fmt.Fprint(s.w, "\x1b[?25l")
	return err
}

func (s *LocalTerminalSession) WriteFrame(frame domain.RenderFrame) error {
	lines := append([]string{}, frame.Lines...)
	lines = append(lines, "", "Controls: W/A/S/D • q quits round view")
	_, err := fmt.Fprintf(s.w, "\x1b[2J\x1b[H%s\n", strings.Join(lines, "\n"))
	return err
}

func (s *LocalTerminalSession) Close() error {
	if s.rawEnabled && s.file != nil && s.restoreState != nil {
		_ = restore(s.file.Fd(), s.restoreState)
	}
	_, err := fmt.Fprintf(s.w, "\x1b[?25h\nSession closed at %s\n", time.Now().Format(time.RFC3339))
	return err
}
