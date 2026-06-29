package macoshelper

import (
	"encoding/json"
	"io"
)

const ProtocolVersion = 1

type Message struct {
	Type            string   `json:"type"`
	Protocol        int      `json:"protocol,omitempty"`
	Room            string   `json:"room,omitempty"`
	State           string   `json:"state,omitempty"`
	Title           string   `json:"title,omitempty"`
	Artist          string   `json:"artist,omitempty"`
	Album           string   `json:"album,omitempty"`
	AlbumArtURL     string   `json:"albumArtURL,omitempty"`
	PositionSeconds *float64 `json:"positionSeconds,omitempty"`
	DurationSeconds *float64 `json:"durationSeconds,omitempty"`
	Volume          *int     `json:"volume,omitempty"`
	Muted           *bool    `json:"muted,omitempty"`
	HUDEnabled      *bool    `json:"hudEnabled,omitempty"`
	HUDPosition     *string  `json:"hudPosition,omitempty"`
	Command         string   `json:"command,omitempty"`
	Text            string   `json:"message,omitempty"`
}

func HelloMessage() Message {
	return Message{Type: "hello", Protocol: ProtocolVersion}
}

func Encode(w io.Writer, msg Message) error {
	return json.NewEncoder(w).Encode(msg)
}

func Decode(r io.Reader, msg *Message) error {
	return json.NewDecoder(r).Decode(msg)
}
