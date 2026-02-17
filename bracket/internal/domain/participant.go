package domain

// Participant represents a tournament participant.
// IDs reference external data in the Tournament Service.
type Participant struct {
	ID   uint64
	Name string
	Seed int
}
