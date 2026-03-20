package room

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/example/ssh-riders/internal/config"
	"github.com/example/ssh-riders/internal/domain"
	"github.com/example/ssh-riders/internal/game"
	"github.com/example/ssh-riders/internal/render"
	"github.com/example/ssh-riders/pkg/logx"
)

type Server struct {
	cfg    config.RoomConfig
	engine *game.Engine
	logger *logx.Logger
}

func New(cfg config.RoomConfig, logger *logx.Logger) *Server {
	return &Server{cfg: cfg, engine: game.NewEngine(cfg), logger: logger}
}

func (s *Server) Run(ctx context.Context) error {
	go s.engine.Run(ctx)
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })
	mux.HandleFunc("/join", s.handleJoin)
	mux.HandleFunc("/input", s.handleInput)
	mux.HandleFunc("/state", s.handleState)
	mux.HandleFunc("/frame", s.handleFrame)
	server := &http.Server{Addr: s.cfg.ListenAddr, Handler: mux}
	go func() { <-ctx.Done(); _ = server.Shutdown(context.Background()) }()
	s.logger.Info("room_started", map[string]any{"room_id": s.cfg.RoomID, "addr": s.cfg.ListenAddr})
	return server.ListenAndServe()
}

func (s *Server) handleJoin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var p domain.Player
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if p.JoinedAt.IsZero() {
		p.JoinedAt = time.Now()
	}
	s.engine.AddPlayer(p)
	_ = json.NewEncoder(w).Encode(domain.Session{ID: p.ID + "-session", PlayerID: p.ID, RoomID: s.cfg.RoomID, ConnectedAt: time.Now(), LastSeenAt: time.Now(), RemoteAddr: r.RemoteAddr})
}

func (s *Server) handleInput(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var in domain.InputEvent
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if in.At.IsZero() {
		in.At = time.Now()
	}
	s.engine.ApplyInput(in)
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleState(w http.ResponseWriter, _ *http.Request) {
	_ = json.NewEncoder(w).Encode(s.engine.Snapshot())
}
func (s *Server) handleFrame(w http.ResponseWriter, r *http.Request) {
	playerID := r.URL.Query().Get("player_id")
	_ = json.NewEncoder(w).Encode(render.BuildFrame(s.cfg.RoomID, playerID, s.engine.Snapshot()))
}
