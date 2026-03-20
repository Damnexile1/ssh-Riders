package orchestrator

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/example/ssh-riders/internal/config"
	"github.com/example/ssh-riders/internal/domain"
	"github.com/example/ssh-riders/internal/transport/internalapi"
	"github.com/example/ssh-riders/pkg/logx"
)

type Service struct {
	cfg      config.OrchestratorConfig
	logger   *logx.Logger
	mu       sync.RWMutex
	registry map[string]domain.RegistryRecord
}

func New(cfg config.OrchestratorConfig, logger *logx.Logger) *Service {
	return &Service{cfg: cfg, logger: logger, registry: map[string]domain.RegistryRecord{}}
}

func (s *Service) Run(ctx context.Context) error {
	if err := os.MkdirAll(filepath.Dir(s.cfg.RegistryPath), 0o755); err != nil {
		return err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })
	mux.HandleFunc("/rooms", s.handleRooms)
	mux.HandleFunc("/register", s.handleRegister)
	mux.HandleFunc("/manifests/", s.handleManifest)
	server := &http.Server{Addr: s.cfg.ListenAddr, Handler: mux}
	go func() {
		<-ctx.Done()
		_ = server.Shutdown(context.Background())
	}()
	s.logger.Info("orchestrator_started", map[string]any{"addr": s.cfg.ListenAddr})
	return server.ListenAndServe()
}

func (s *Service) handleRooms(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		_ = json.NewEncoder(w).Encode(s.ListRooms())
	case http.MethodPost:
		var req internalapi.CreateRoomRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := s.CreateRoomFromManifest(req.ManifestPath); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Service) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var room domain.Room
	if err := json.NewDecoder(r.Body).Decode(&room); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.RegisterRoom(room)
	w.WriteHeader(http.StatusAccepted)
}

func (s *Service) handleManifest(w http.ResponseWriter, r *http.Request) {
	path := filepath.Join(s.cfg.ManifestDir, filepath.Base(r.URL.Path))
	http.ServeFile(w, r, path)
}

func (s *Service) ListRooms() []domain.Room {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rooms := make([]domain.Room, 0, len(s.registry))
	for _, rec := range s.registry {
		rooms = append(rooms, rec.Room)
	}
	return rooms
}

func (s *Service) RegisterRoom(room domain.Room) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.registry[room.ID] = domain.RegistryRecord{Room: room, LastPingAt: time.Now()}
	s.persistLocked()
}

func (s *Service) CreateRoomFromManifest(manifestPath string) error {
	body, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	manifest, err := parseManifest(body)
	if err != nil {
		return err
	}
	room := domain.Room{
		ID:           manifest.ID,
		Name:         manifest.Name,
		Address:      manifest.ListenAddress,
		MaxPlayers:   manifest.MaxPlayers,
		Status:       "booting",
		CreatedAt:    time.Now(),
		ManifestPath: manifestPath,
	}
	s.RegisterRoom(room)
	s.logger.Info("room_manifest_loaded", map[string]any{"room_id": manifest.ID, "image": manifest.Image, "listen": manifest.ListenAddress})
	return nil
}

func (s *Service) persistLocked() {
	f, err := os.Create(s.cfg.RegistryPath)
	if err != nil {
		s.logger.Error("registry_persist_failed", map[string]any{"err": err.Error()})
		return
	}
	defer f.Close()
	_ = json.NewEncoder(f).Encode(s.registry)
}

func parseManifest(body []byte) (domain.RoomManifest, error) {
	var manifest domain.RoomManifest
	lines := bytes.Split(body, []byte("\n"))
	for _, raw := range lines {
		line := strings.TrimSpace(string(raw))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), "\"")
		switch key {
		case "id":
			manifest.ID = value
		case "name":
			manifest.Name = value
		case "image":
			manifest.Image = value
		case "listen_address":
			manifest.ListenAddress = value
		case "max_players":
			manifest.MaxPlayers, _ = strconv.Atoi(value)
		case "tick_rate":
			manifest.TickRate, _ = strconv.Atoi(value)
		case "arena_width":
			manifest.ArenaWidth, _ = strconv.Atoi(value)
		case "arena_height":
			manifest.ArenaHeight, _ = strconv.Atoi(value)
		case "countdown_sec":
			manifest.CountdownSec, _ = strconv.Atoi(value)
		case "idle_ttl_seconds":
			manifest.IdleTTLSeconds, _ = strconv.Atoi(value)
		}
	}
	return manifest, nil
}
