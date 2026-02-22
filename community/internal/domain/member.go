package domain

import "time"

type MemberRole string

const (
	RoleOwner  MemberRole = "owner"
	RoleAdmin  MemberRole = "admin"
	RoleMember MemberRole = "member"
)

type CommunityMember struct {
	ID          uint64
	CommunityID uint64
	UserID      *uint64 // NULL for ghost members
	DisplayName string
	Role        MemberRole

	// ELO/ranking fields (future use)
	EloRating     *int
	RankingPoints *int
	MatchesPlayed int
	MatchesWon    int

	JoinedAt  time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// IsGhost returns true if this is a ghost member (no linked user account)
func (m *CommunityMember) IsGhost() bool {
	return m.UserID == nil
}
