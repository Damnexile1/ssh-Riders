package domain

import "time"

type Direction string

const (
	DirectionUp    Direction = "up"
	DirectionDown  Direction = "down"
	DirectionLeft  Direction = "left"
	DirectionRight Direction = "right"
)

type Position struct {
	X int `json:"x" yaml:"x"`
	Y int `json:"y" yaml:"y"`
}

type Player struct {
	ID        string    `json:"id" yaml:"id"`
	Name      string    `json:"name" yaml:"name"`
	ColorANSI string    `json:"color_ansi" yaml:"color_ansi"`
	Alive     bool      `json:"alive" yaml:"alive"`
	Score     int       `json:"score" yaml:"score"`
	JoinedAt  time.Time `json:"joined_at" yaml:"joined_at"`
}

type Session struct {
	ID          string    `json:"id" yaml:"id"`
	PlayerID    string    `json:"player_id" yaml:"player_id"`
	RoomID      string    `json:"room_id" yaml:"room_id"`
	ConnectedAt time.Time `json:"connected_at" yaml:"connected_at"`
	LastSeenAt  time.Time `json:"last_seen_at" yaml:"last_seen_at"`
	RemoteAddr  string    `json:"remote_addr" yaml:"remote_addr"`
}

type Arena struct {
	Width     int        `json:"width" yaml:"width"`
	Height    int        `json:"height" yaml:"height"`
	Obstacles []Position `json:"obstacles" yaml:"obstacles"`
	Wrap      bool       `json:"wrap" yaml:"wrap"`
}

type OccupiedCell struct {
	Position Position `json:"position" yaml:"position"`
	OwnerID  string   `json:"owner_id" yaml:"owner_id"`
	Tick     uint64   `json:"tick" yaml:"tick"`
}

type RoundPhase string

const (
	RoundWaiting   RoundPhase = "waiting"
	RoundCountdown RoundPhase = "countdown"
	RoundRunning   RoundPhase = "running"
	RoundFinished  RoundPhase = "finished"
)

type RoundState struct {
	Number             int        `json:"number" yaml:"number"`
	Phase              RoundPhase `json:"phase" yaml:"phase"`
	WinnerPlayerID     string     `json:"winner_player_id" yaml:"winner_player_id"`
	CountdownRemaining int        `json:"countdown_remaining" yaml:"countdown_remaining"`
	StartedAt          time.Time  `json:"started_at" yaml:"started_at"`
	FinishedAt         time.Time  `json:"finished_at" yaml:"finished_at"`
}

type ScoreEntry struct {
	PlayerID string `json:"player_id" yaml:"player_id"`
	Name     string `json:"name" yaml:"name"`
	Score    int    `json:"score" yaml:"score"`
	Alive    bool   `json:"alive" yaml:"alive"`
}

type ScoreBoard struct {
	Entries []ScoreEntry `json:"entries" yaml:"entries"`
}

type RiderState struct {
	PlayerID   string     `json:"player_id" yaml:"player_id"`
	Head       Position   `json:"head" yaml:"head"`
	Direction  Direction  `json:"direction" yaml:"direction"`
	Eliminated bool       `json:"eliminated" yaml:"eliminated"`
	Trail      []Position `json:"trail" yaml:"trail"`
}

type RoomState struct {
	Tick       uint64                 `json:"tick" yaml:"tick"`
	Arena      Arena                  `json:"arena" yaml:"arena"`
	Round      RoundState             `json:"round" yaml:"round"`
	Riders     map[string]*RiderState `json:"riders" yaml:"riders"`
	Occupied   map[Position]string    `json:"occupied" yaml:"occupied"`
	ScoreBoard ScoreBoard             `json:"scoreboard" yaml:"scoreboard"`
}

type Room struct {
	ID             string    `json:"id" yaml:"id"`
	Name           string    `json:"name" yaml:"name"`
	Address        string    `json:"address" yaml:"address"`
	MaxPlayers     int       `json:"max_players" yaml:"max_players"`
	CurrentPlayers int       `json:"current_players" yaml:"current_players"`
	Status         string    `json:"status" yaml:"status"`
	CreatedAt      time.Time `json:"created_at" yaml:"created_at"`
	ManifestPath   string    `json:"manifest_path" yaml:"manifest_path"`
}

type RoomManifest struct {
	ID             string `yaml:"id"`
	Name           string `yaml:"name"`
	Image          string `yaml:"image"`
	ListenAddress  string `yaml:"listen_address"`
	MaxPlayers     int    `yaml:"max_players"`
	TickRate       int    `yaml:"tick_rate"`
	ArenaWidth     int    `yaml:"arena_width"`
	ArenaHeight    int    `yaml:"arena_height"`
	CountdownSec   int    `yaml:"countdown_sec"`
	IdleTTLSeconds int    `yaml:"idle_ttl_seconds"`
}

type InputEvent struct {
	SessionID string    `json:"session_id" yaml:"session_id"`
	PlayerID  string    `json:"player_id" yaml:"player_id"`
	Direction Direction `json:"direction" yaml:"direction"`
	TickHint  uint64    `json:"tick_hint" yaml:"tick_hint"`
	At        time.Time `json:"at" yaml:"at"`
}

type RenderFrame struct {
	Tick        uint64    `json:"tick" yaml:"tick"`
	RoomID      string    `json:"room_id" yaml:"room_id"`
	PlayerID    string    `json:"player_id" yaml:"player_id"`
	Lines       []string  `json:"lines" yaml:"lines"`
	GeneratedAt time.Time `json:"generated_at" yaml:"generated_at"`
	Full        bool      `json:"full" yaml:"full"`
}

type RegistryRecord struct {
	Room       Room      `json:"room" yaml:"room"`
	LastPingAt time.Time `json:"last_ping_at" yaml:"last_ping_at"`
}
