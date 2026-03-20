package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/example/ssh-riders/internal/config"
	"github.com/example/ssh-riders/internal/domain"
	"github.com/example/ssh-riders/internal/transport/internalapi"
	"github.com/example/ssh-riders/internal/transport/sshsession"
	"github.com/example/ssh-riders/pkg/logx"
)

type App struct {
	cfg    config.GatewayConfig
	logger *logx.Logger
	client *internalapi.Client
}

func New(cfg config.GatewayConfig, logger *logx.Logger) *App {
	return &App{cfg: cfg, logger: logger, client: internalapi.NewClient(cfg.OrchestratorAddr)}
}

func (a *App) RunCLI(ctx context.Context, r io.Reader, w io.Writer) error {
	sess := sshsession.NewLocalTerminalSession("local-1", r, w, "127.0.0.1")
	defer sess.Close()
	rooms, err := a.client.ListRooms()
	if err != nil || len(rooms) == 0 {
		rooms = []domain.Room{{ID: a.cfg.DefaultRoomID, Name: "Alpha Room", Address: "http://127.0.0.1:9090", MaxPlayers: 6, Status: "ready"}}
	}
	welcome := domain.RenderFrame{Lines: []string{
		"SSH Riders Gateway",
		"Local terminal mode enabled.",
		"Enter your rider name and press ENTER:",
	}}
	if err := sess.WriteFrame(welcome); err != nil {
		return err
	}
	name, err := sess.ReadLine(ctx)
	if err != nil {
		return err
	}
	if strings.TrimSpace(name) == "" {
		name = "rider"
	}
	if err := sess.EnableGameMode(); err != nil {
		a.logger.Error("enable_game_mode_failed", map[string]any{"err": err.Error()})
	}
	room := rooms[0]
	player := domain.Player{ID: sanitize(name), Name: name, JoinedAt: time.Now(), ColorANSI: "36"}
	if err := a.joinRoom(room, player); err != nil {
		return err
	}
	frameTicker := time.NewTicker(a.cfg.LobbyRefresh)
	defer frameTicker.Stop()
	inputErrCh := make(chan error, 1)
	go a.captureInput(ctx, sess, room, player, inputErrCh)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-inputErrCh:
			if err != nil && !errors.Is(err, context.Canceled) {
				return err
			}
			return nil
		case <-frameTicker.C:
			frame, err := a.fetchFrame(room.Address, player.ID)
			if err != nil {
				return err
			}
			if err := sess.WriteFrame(frame); err != nil {
				return err
			}
		}
	}
}

func (a *App) joinRoom(room domain.Room, player domain.Player) error {
	body, _ := json.Marshal(player)
	resp, err := http.Post(room.Address+"/join", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("join room failed: %s", resp.Status)
	}
	return nil
}

func (a *App) captureInput(ctx context.Context, sess sshsession.Session, room domain.Room, player domain.Player, errCh chan<- error) {
	for {
		key, err := sess.ReadKey(ctx)
		if err != nil {
			errCh <- err
			return
		}
		if key == 'q' || key == 'Q' {
			errCh <- nil
			return
		}
		dir, ok := parseDirectionKey(key)
		if !ok {
			continue
		}
		payload, _ := json.Marshal(domain.InputEvent{SessionID: sess.ID(), PlayerID: player.ID, Direction: dir, At: time.Now()})
		_, _ = http.Post(room.Address+"/input", "application/json", bytes.NewReader(payload))
	}
}

func (a *App) fetchFrame(roomAddr, playerID string) (domain.RenderFrame, error) {
	resp, err := http.Get(fmt.Sprintf("%s/frame?player_id=%s", roomAddr, playerID))
	if err != nil {
		return domain.RenderFrame{}, err
	}
	defer resp.Body.Close()
	var frame domain.RenderFrame
	return frame, json.NewDecoder(resp.Body).Decode(&frame)
}

func sanitize(v string) string {
	return strings.NewReplacer(" ", "-", "/", "-", "\\", "-").Replace(strings.ToLower(v))
}

func parseDirectionKey(key byte) (domain.Direction, bool) {
	switch key {
	case 'w', 'W':
		return domain.DirectionUp, true
	case 's', 'S':
		return domain.DirectionDown, true
	case 'a', 'A':
		return domain.DirectionLeft, true
	case 'd', 'D':
		return domain.DirectionRight, true
	default:
		return "", false
	}
}

func RunMain() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return New(config.LoadGateway(), logx.New("gateway")).RunCLI(ctx, os.Stdin, os.Stdout)
}
