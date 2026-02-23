package domain

// Participant represents a tournament participant.
// IDs reference external data in the Tournament Service.
type Participant struct {
	ID      uint64 `json:"id"`
	Name    string `json:"name"`
	Seed    int    `json:"seed"`
	IconURL string `json:"icon_url"`
}
