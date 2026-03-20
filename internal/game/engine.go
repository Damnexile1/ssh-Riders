package game

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/example/ssh-riders/internal/config"
	"github.com/example/ssh-riders/internal/domain"
)

type Engine struct {
	mu           sync.RWMutex
	cfg          config.RoomConfig
	state        domain.RoomState
	players      map[string]domain.Player
	inputs       map[string]domain.Direction
	lastActivity time.Time
}

func NewEngine(cfg config.RoomConfig) *Engine {
	occupied := make(map[domain.Position]string)
	state := domain.RoomState{
		Arena:    domain.Arena{Width: cfg.ArenaWidth, Height: cfg.ArenaHeight},
		Round:    domain.RoundState{Number: 1, Phase: domain.RoundWaiting, CountdownRemaining: cfg.CountdownSeconds},
		Riders:   make(map[string]*domain.RiderState),
		Occupied: occupied,
	}
	return &Engine{cfg: cfg, state: state, players: map[string]domain.Player{}, inputs: map[string]domain.Direction{}, lastActivity: time.Now()}
}

func (e *Engine) AddPlayer(p domain.Player) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, ok := e.players[p.ID]; ok {
		return
	}
	p.Alive = true
	if p.ColorANSI == "" {
		p.ColorANSI = "36"
	}
	e.players[p.ID] = p
	e.spawnLocked(p.ID)
	e.rebuildScoreboardLocked()
	e.lastActivity = time.Now()
	if len(e.players) >= 2 && e.state.Round.Phase == domain.RoundWaiting {
		e.state.Round.Phase = domain.RoundCountdown
	}
}

func (e *Engine) RemovePlayer(playerID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.players, playerID)
	delete(e.inputs, playerID)
	delete(e.state.Riders, playerID)
	for pos, owner := range e.state.Occupied {
		if owner == playerID {
			delete(e.state.Occupied, pos)
		}
	}
	e.rebuildScoreboardLocked()
	e.lastActivity = time.Now()
}

func (e *Engine) ApplyInput(in domain.InputEvent) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if rider, ok := e.state.Riders[in.PlayerID]; ok && !isReverse(rider.Direction, in.Direction) {
		e.inputs[in.PlayerID] = in.Direction
	}
	e.lastActivity = time.Now()
}

func (e *Engine) Snapshot() domain.RoomState {
	e.mu.RLock()
	defer e.mu.RUnlock()
	cp := e.state
	cp.Riders = make(map[string]*domain.RiderState, len(e.state.Riders))
	for id, rider := range e.state.Riders {
		r := *rider
		r.Trail = append([]domain.Position(nil), rider.Trail...)
		cp.Riders[id] = &r
	}
	cp.ScoreBoard.Entries = append([]domain.ScoreEntry(nil), e.state.ScoreBoard.Entries...)
	cp.Occupied = make(map[domain.Position]string, len(e.state.Occupied))
	for pos, owner := range e.state.Occupied {
		cp.Occupied[pos] = owner
	}
	return cp
}

func (e *Engine) IdleFor() time.Duration {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return time.Since(e.lastActivity)
}

func (e *Engine) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Second / time.Duration(e.cfg.TickRate))
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.tick()
		}
	}
}

func (e *Engine) tick() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.state.Tick++
	switch e.state.Round.Phase {
	case domain.RoundWaiting:
		if len(e.players) >= 2 {
			e.state.Round.Phase = domain.RoundCountdown
			e.state.Round.CountdownRemaining = e.cfg.CountdownSeconds
		}
		return
	case domain.RoundCountdown:
		if e.state.Tick%uint64(e.cfg.TickRate) == 0 {
			e.state.Round.CountdownRemaining--
			if e.state.Round.CountdownRemaining <= 0 {
				e.state.Round.Phase = domain.RoundRunning
				e.state.Round.StartedAt = time.Now()
			}
		}
		return
	case domain.RoundFinished:
		if e.state.Tick%uint64(e.cfg.TickRate) == 0 {
			e.resetRoundLocked()
		}
		return
	}
	aliveCount := 0
	aliveID := ""
	for playerID, rider := range e.state.Riders {
		if rider.Eliminated {
			continue
		}
		if next, ok := e.inputs[playerID]; ok {
			rider.Direction = next
		}
		nextPos := step(rider.Head, rider.Direction)
		if e.hitLocked(nextPos) {
			rider.Eliminated = true
			if p := e.players[playerID]; p.ID != "" {
				p.Alive = false
				e.players[playerID] = p
			}
			continue
		}
		rider.Head = nextPos
		rider.Trail = append(rider.Trail, nextPos)
		e.state.Occupied[nextPos] = playerID
		aliveCount++
		aliveID = playerID
	}
	e.rebuildScoreboardLocked()
	if aliveCount <= 1 && len(e.state.Riders) > 1 {
		e.state.Round.Phase = domain.RoundFinished
		e.state.Round.FinishedAt = time.Now()
		e.state.Round.WinnerPlayerID = aliveID
		if aliveID != "" {
			p := e.players[aliveID]
			p.Score++
			e.players[aliveID] = p
		}
		e.rebuildScoreboardLocked()
	}
}

func (e *Engine) resetRoundLocked() {
	e.state.Tick = 0
	e.state.Round.Number++
	e.state.Round.Phase = domain.RoundCountdown
	e.state.Round.CountdownRemaining = e.cfg.CountdownSeconds
	e.state.Round.WinnerPlayerID = ""
	e.state.Riders = map[string]*domain.RiderState{}
	e.state.Occupied = map[domain.Position]string{}
	for id, p := range e.players {
		p.Alive = true
		e.players[id] = p
		e.spawnLocked(id)
	}
	e.rebuildScoreboardLocked()
}

func (e *Engine) spawnLocked(playerID string) {
	idx := len(e.state.Riders)
	spawns := []struct {
		pos domain.Position
		dir domain.Direction
	}{
		{domain.Position{X: 3, Y: 3}, domain.DirectionRight},
		{domain.Position{X: e.cfg.ArenaWidth - 4, Y: e.cfg.ArenaHeight - 4}, domain.DirectionLeft},
		{domain.Position{X: e.cfg.ArenaWidth - 4, Y: 3}, domain.DirectionDown},
		{domain.Position{X: 3, Y: e.cfg.ArenaHeight - 4}, domain.DirectionUp},
	}
	spawn := spawns[idx%len(spawns)]
	e.state.Riders[playerID] = &domain.RiderState{PlayerID: playerID, Head: spawn.pos, Direction: spawn.dir, Trail: []domain.Position{spawn.pos}}
	e.state.Occupied[spawn.pos] = playerID
}

func (e *Engine) hitLocked(pos domain.Position) bool {
	if pos.X < 0 || pos.Y < 0 || pos.X >= e.cfg.ArenaWidth || pos.Y >= e.cfg.ArenaHeight {
		return true
	}
	_, occupied := e.state.Occupied[pos]
	return occupied
}

func (e *Engine) rebuildScoreboardLocked() {
	entries := make([]domain.ScoreEntry, 0, len(e.players))
	for _, p := range e.players {
		entries = append(entries, domain.ScoreEntry{PlayerID: p.ID, Name: p.Name, Score: p.Score, Alive: p.Alive})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Score == entries[j].Score {
			return entries[i].Name < entries[j].Name
		}
		return entries[i].Score > entries[j].Score
	})
	e.state.ScoreBoard = domain.ScoreBoard{Entries: entries}
}

func step(pos domain.Position, dir domain.Direction) domain.Position {
	switch dir {
	case domain.DirectionUp:
		pos.Y--
	case domain.DirectionDown:
		pos.Y++
	case domain.DirectionLeft:
		pos.X--
	case domain.DirectionRight:
		pos.X++
	}
	return pos
}

func isReverse(prev, next domain.Direction) bool {
	return (prev == domain.DirectionUp && next == domain.DirectionDown) ||
		(prev == domain.DirectionDown && next == domain.DirectionUp) ||
		(prev == domain.DirectionLeft && next == domain.DirectionRight) ||
		(prev == domain.DirectionRight && next == domain.DirectionLeft)
}

func DebugFrame(state domain.RoomState) string {
	return fmt.Sprintf("tick=%d riders=%d phase=%s", state.Tick, len(state.Riders), state.Round.Phase)
}
