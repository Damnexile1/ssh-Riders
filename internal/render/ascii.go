package render

import (
	"fmt"
	"strings"
	"time"

	"github.com/example/ssh-riders/internal/domain"
)

func BuildFrame(roomID, playerID string, state domain.RoomState) domain.RenderFrame {
	grid := make([][]rune, state.Arena.Height)
	for y := range grid {
		grid[y] = make([]rune, state.Arena.Width)
		for x := range grid[y] {
			grid[y][x] = ' '
		}
	}
	for pos := range state.Occupied {
		if inside(pos, state.Arena) {
			grid[pos.Y][pos.X] = '·'
		}
	}
	for _, rider := range state.Riders {
		if inside(rider.Head, state.Arena) {
			grid[rider.Head.Y][rider.Head.X] = '@'
		}
	}
	lines := []string{fmt.Sprintf("SSH Riders | room=%s | tick=%d | phase=%s", roomID, state.Tick, state.Round.Phase)}
	lines = append(lines, "+"+strings.Repeat("-", state.Arena.Width)+"+")
	for y := 0; y < state.Arena.Height; y++ {
		lines = append(lines, "|"+string(grid[y])+"|")
	}
	lines = append(lines, "+"+strings.Repeat("-", state.Arena.Width)+"+")
	lines = append(lines, "Scoreboard:")
	for _, entry := range state.ScoreBoard.Entries {
		status := "DEAD"
		if entry.Alive {
			status = "ALIVE"
		}
		lines = append(lines, fmt.Sprintf(" - %-12s %2d pts [%s]", entry.Name, entry.Score, status))
	}
	if state.Round.WinnerPlayerID != "" {
		lines = append(lines, "Winner: "+state.Round.WinnerPlayerID)
	}
	return domain.RenderFrame{Tick: state.Tick, RoomID: roomID, PlayerID: playerID, Lines: lines, GeneratedAt: time.Now(), Full: true}
}

func inside(pos domain.Position, arena domain.Arena) bool {
	return pos.X >= 0 && pos.Y >= 0 && pos.X < arena.Width && pos.Y < arena.Height
}
