package internalapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/example/ssh-riders/internal/domain"
)

type CreateRoomRequest struct {
	ManifestPath string `json:"manifest_path"`
}
type JoinRoomRequest struct {
	RoomID, PlayerName string `json:"room_id","player_name"`
}
type JoinRoomResponse struct {
	Room    domain.Room    `json:"room"`
	Player  domain.Player  `json:"player"`
	Session domain.Session `json:"session"`
}

type Registry interface {
	ListRooms() []domain.Room
	RegisterRoom(room domain.Room)
}

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{baseURL: baseURL, httpClient: &http.Client{Timeout: 3 * time.Second}}
}
func (c *Client) ListRooms() ([]domain.Room, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/rooms")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var rooms []domain.Room
	return rooms, json.NewDecoder(resp.Body).Decode(&rooms)
}
func (c *Client) CreateRoom(req CreateRoomRequest) error {
	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(req); err != nil {
		return err
	}
	resp, err := c.httpClient.Post(c.baseURL+"/rooms", "application/json", buf)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
