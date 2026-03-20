package gateway

import (
	"bytes"
	"context"
	"encoding/json"
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
		"This binary contains SSH session handling skeleton for local terminal emulation.",
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
	room := rooms[0]
	player := domain.Player{ID: sanitize(name), Name: name, JoinedAt: time.Now(), ColorANSI: "36"}
	body, _ := json.Marshal(player)
	resp, err := http.Post(room.Address+"/join", "application/json", bytes.NewReader(body))
	if err == nil {
		resp.Body.Close()
	}
	frameTicker := time.NewTicker(a.cfg.LobbyRefresh)
	defer frameTicker.Stop()
	inputCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go a.captureInput(inputCtx, sess, room, player)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
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

func (a *App) captureInput(ctx context.Context, sess sshsession.Session, room domain.Room, player domain.Player) {
	for {
		line, err := sess.ReadLine(ctx)
		if err != nil {
			return
		}
		dir, ok := parseDirection(line)
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
func parseDirection(v string) (domain.Direction, bool) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "w", "up":
		return domain.DirectionUp, true
	case "s", "down":
		return domain.DirectionDown, true
	case "a", "left":
		return domain.DirectionLeft, true
	case "d", "right":
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
