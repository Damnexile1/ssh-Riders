package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type GatewayConfig struct {
	ListenAddr       string
	LobbyRefresh     time.Duration
	OrchestratorAddr string
	DefaultRoomID    string
}

type OrchestratorConfig struct {
	ListenAddr     string
	RegistryPath   string
	ManifestDir    string
	RoomBinaryPath string
}

type RoomConfig struct {
	RoomID           string
	Name             string
	ListenAddr       string
	TickRate         int
	ArenaWidth       int
	ArenaHeight      int
	MaxPlayers       int
	CountdownSeconds int
	IdleTTL          time.Duration
}

func LoadGateway() GatewayConfig {
	return GatewayConfig{
		ListenAddr:       getenv("GATEWAY_LISTEN_ADDR", ":2222"),
		LobbyRefresh:     getDuration("GATEWAY_LOBBY_REFRESH", 2*time.Second),
		OrchestratorAddr: getenv("ORCHESTRATOR_ADDR", "http://127.0.0.1:8081"),
		DefaultRoomID:    getenv("DEFAULT_ROOM_ID", "alpha"),
	}
}

func LoadOrchestrator() OrchestratorConfig {
	return OrchestratorConfig{
		ListenAddr:     getenv("ORCHESTRATOR_LISTEN_ADDR", ":8081"),
		RegistryPath:   getenv("REGISTRY_PATH", "./var/registry.json"),
		ManifestDir:    getenv("MANIFEST_DIR", "./manifests"),
		RoomBinaryPath: getenv("ROOM_BINARY_PATH", "./room"),
	}
}

func LoadRoom() RoomConfig {
	return RoomConfig{
		RoomID:           getenv("ROOM_ID", "alpha"),
		Name:             getenv("ROOM_NAME", "Alpha Room"),
		ListenAddr:       getenv("ROOM_LISTEN_ADDR", ":9090"),
		TickRate:         getInt("ROOM_TICK_RATE", 12),
		ArenaWidth:       getInt("ROOM_ARENA_WIDTH", 48),
		ArenaHeight:      getInt("ROOM_ARENA_HEIGHT", 20),
		MaxPlayers:       getInt("ROOM_MAX_PLAYERS", 6),
		CountdownSeconds: getInt("ROOM_COUNTDOWN_SECONDS", 3),
		IdleTTL:          getDuration("ROOM_IDLE_TTL", 30*time.Second),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		i, err := strconv.Atoi(v)
		if err == nil {
			return i
		}
	}
	return fallback
}

func getDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil {
			return d
		}
	}
	return fallback
}

func (c RoomConfig) Validate() error {
	if c.TickRate <= 0 {
		return fmt.Errorf("tick rate must be positive")
	}
	if c.ArenaWidth < 16 || c.ArenaHeight < 10 {
		return fmt.Errorf("arena too small")
	}
	return nil
}
