package domain

import (
	"encoding/json"
	"time"
)

type Community struct {
	ID          uint64
	Slug        string
	OwnerID     uint64 // References auth service user
	Name        string
	Description *string
	Game        *string
	AvatarURL   *string
	Settings    json.RawMessage
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
