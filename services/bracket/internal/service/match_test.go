package service

import (
	"context"
	"testing"

	"github.com/braccet/bracket/internal/domain"
	"github.com/braccet/bracket/internal/repository"
)

// mockMatchRepository implements repository.MatchRepository for testing
type mockMatchRepository struct {
	matches map[uint64]*domain.Match
	nextID  uint64
}

func newMockRepo() *mockMatchRepository {
	return &mockMatchRepository{
		matches: make(map[uint64]*domain.Match),
		nextID:  1,
	}
}

func (r *mockMatchRepository) CreateBatch(ctx context.Context, matches []*domain.Match) error {
	for _, m := range matches {
		m.ID = r.nextID
		r.nextID++
		r.matches[m.ID] = m
	}
	return nil
}

func (r *mockMatchRepository) GetByID(ctx context.Context, id uint64) (*domain.Match, error) {
	m, ok := r.matches[id]
	if !ok {
		return nil, repository.ErrMatchNotFound
	}
	// Return a copy to simulate DB behavior
	copy := *m
	return &copy, nil
}

func (r *mockMatchRepository) GetByTournament(ctx context.Context, tournamentID uint64) ([]*domain.Match, error) {
	var matches []*domain.Match
	for _, m := range r.matches {
		if m.TournamentID == tournamentID {
			matches = append(matches, m)
		}
	}
	return matches, nil
}

func (r *mockMatchRepository) UpdateResult(ctx context.Context, matchID uint64, result domain.MatchResult) error {
	m, ok := r.matches[matchID]
	if !ok {
		return repository.ErrMatchNotFound
	}
	m.WinnerID = &result.WinnerID
	m.Participant1Score = &result.Participant1Score
	m.Participant2Score = &result.Participant2Score
	m.Status = domain.MatchCompleted
	return nil
}

func (r *mockMatchRepository) UpdateStatus(ctx context.Context, matchID uint64, status domain.MatchStatus) error {
	m, ok := r.matches[matchID]
	if !ok {
		return repository.ErrMatchNotFound
	}
	m.Status = status
	return nil
}

func (r *mockMatchRepository) SetParticipant(ctx context.Context, matchID uint64, slot int, participantID uint64, name string) error {
	m, ok := r.matches[matchID]
	if !ok {
		return repository.ErrMatchNotFound
	}
	if slot == 1 {
		m.Participant1ID = &participantID
		m.Participant1Name = &name
	} else {
		m.Participant2ID = &participantID
		m.Participant2Name = &name
	}
	return nil
}

func (r *mockMatchRepository) UpdateNextMatchLinks(ctx context.Context, matches []*domain.Match) error {
	for _, m := range matches {
		if stored, ok := r.matches[m.ID]; ok {
			stored.NextMatchID = m.NextMatchID
			stored.LoserMatchID = m.LoserMatchID
		}
	}
	return nil
}

// Helper to create a simple 4-player bracket for testing
func createTestBracket(repo *mockMatchRepository) []*domain.Match {
	p1, p2, p3, p4 := uint64(1), uint64(2), uint64(3), uint64(4)
	n1, n2, n3, n4 := "Player1", "Player2", "Player3", "Player4"

	matches := []*domain.Match{
		{
			TournamentID:     1,
			Round:            1,
			Position:         1,
			Participant1ID:   &p1,
			Participant2ID:   &p4,
			Participant1Name: &n1,
			Participant2Name: &n4,
			Status:           domain.MatchReady,
			BracketType:      domain.BracketWinners,
		},
		{
			TournamentID:     1,
			Round:            1,
			Position:         2,
			Participant1ID:   &p2,
			Participant2ID:   &p3,
			Participant1Name: &n2,
			Participant2Name: &n3,
			Status:           domain.MatchReady,
			BracketType:      domain.BracketWinners,
		},
		{
			TournamentID:     1,
			Round:            2,
			Position:         1,
			Status:           domain.MatchPending,
			BracketType:      domain.BracketWinners,
		},
	}

	repo.CreateBatch(context.Background(), matches)

	// Link matches
	matches[0].NextMatchID = &matches[2].ID
	matches[1].NextMatchID = &matches[2].ID
	repo.UpdateNextMatchLinks(context.Background(), matches)

	return matches
}

func TestReportResult_Success(t *testing.T) {
	repo := newMockRepo()
	createTestBracket(repo)
	svc := NewMatchService(repo)
	ctx := context.Background()

	// Report result for match 1 (seed 1 vs seed 4)
	result := domain.MatchResult{
		WinnerID:          1,
		Participant1Score: 2,
		Participant2Score: 0,
	}

	err := svc.ReportResult(ctx, 1, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check match 1 is completed
	match1, _ := repo.GetByID(ctx, 1)
	if match1.Status != domain.MatchCompleted {
		t.Errorf("expected match 1 to be completed, got %s", match1.Status)
	}
	if match1.WinnerID == nil || *match1.WinnerID != 1 {
		t.Error("expected winner to be participant 1")
	}

	// Check winner advanced to final
	final, _ := repo.GetByID(ctx, 3)
	if final.Participant1ID == nil || *final.Participant1ID != 1 {
		t.Error("expected winner to be placed in final match slot 1")
	}
}

func TestReportResult_BothMatchesComplete_FinalReady(t *testing.T) {
	repo := newMockRepo()
	createTestBracket(repo)
	svc := NewMatchService(repo)
	ctx := context.Background()

	// Report result for match 1
	svc.ReportResult(ctx, 1, domain.MatchResult{WinnerID: 1})

	// Report result for match 2
	err := svc.ReportResult(ctx, 2, domain.MatchResult{WinnerID: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check final is now ready
	final, _ := repo.GetByID(ctx, 3)
	if final.Status != domain.MatchReady {
		t.Errorf("expected final to be ready, got %s", final.Status)
	}
	if final.Participant1ID == nil || *final.Participant1ID != 1 {
		t.Error("expected participant 1 in slot 1")
	}
	if final.Participant2ID == nil || *final.Participant2ID != 2 {
		t.Error("expected participant 2 in slot 2")
	}
}

func TestReportResult_InvalidWinner(t *testing.T) {
	repo := newMockRepo()
	createTestBracket(repo)
	svc := NewMatchService(repo)
	ctx := context.Background()

	// Try to report with invalid winner
	result := domain.MatchResult{WinnerID: 999}
	err := svc.ReportResult(ctx, 1, result)

	if err != ErrInvalidWinner {
		t.Errorf("expected ErrInvalidWinner, got %v", err)
	}
}

func TestReportResult_MatchNotReady(t *testing.T) {
	repo := newMockRepo()
	createTestBracket(repo)
	svc := NewMatchService(repo)
	ctx := context.Background()

	// Try to report result for pending match (the final)
	result := domain.MatchResult{WinnerID: 1}
	err := svc.ReportResult(ctx, 3, result)

	if err != ErrMatchNotReady {
		t.Errorf("expected ErrMatchNotReady, got %v", err)
	}
}

func TestReportResult_AlreadyComplete(t *testing.T) {
	repo := newMockRepo()
	createTestBracket(repo)
	svc := NewMatchService(repo)
	ctx := context.Background()

	// Report result once
	svc.ReportResult(ctx, 1, domain.MatchResult{WinnerID: 1})

	// Try to report again
	err := svc.ReportResult(ctx, 1, domain.MatchResult{WinnerID: 4})

	if err != ErrMatchAlreadyComplete {
		t.Errorf("expected ErrMatchAlreadyComplete, got %v", err)
	}
}

func TestStartMatch(t *testing.T) {
	repo := newMockRepo()
	createTestBracket(repo)
	svc := NewMatchService(repo)
	ctx := context.Background()

	// Start match 1
	err := svc.StartMatch(ctx, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	match, _ := repo.GetByID(ctx, 1)
	if match.Status != domain.MatchInProgress {
		t.Errorf("expected in_progress, got %s", match.Status)
	}
}

func TestStartMatch_NotReady(t *testing.T) {
	repo := newMockRepo()
	createTestBracket(repo)
	svc := NewMatchService(repo)
	ctx := context.Background()

	// Try to start the final (which is pending)
	err := svc.StartMatch(ctx, 3)

	if err != ErrMatchNotReady {
		t.Errorf("expected ErrMatchNotReady, got %v", err)
	}
}

func TestGetBracketState_Initial(t *testing.T) {
	repo := newMockRepo()
	createTestBracket(repo)
	svc := NewMatchService(repo)
	ctx := context.Background()

	state, err := svc.GetBracketState(ctx, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if state.TournamentID != 1 {
		t.Errorf("expected tournament ID 1, got %d", state.TournamentID)
	}
	if state.TotalRounds != 2 {
		t.Errorf("expected 2 rounds, got %d", state.TotalRounds)
	}
	if state.CurrentRound != 1 {
		t.Errorf("expected current round 1, got %d", state.CurrentRound)
	}
	if state.IsComplete {
		t.Error("bracket should not be complete")
	}
}

func TestGetBracketState_Complete(t *testing.T) {
	repo := newMockRepo()
	createTestBracket(repo)
	svc := NewMatchService(repo)
	ctx := context.Background()

	// Complete all matches
	svc.ReportResult(ctx, 1, domain.MatchResult{WinnerID: 1})
	svc.ReportResult(ctx, 2, domain.MatchResult{WinnerID: 2})
	svc.ReportResult(ctx, 3, domain.MatchResult{WinnerID: 1})

	state, _ := svc.GetBracketState(ctx, 1)

	if !state.IsComplete {
		t.Error("bracket should be complete")
	}
	if state.ChampionID == nil || *state.ChampionID != 1 {
		t.Error("expected champion to be participant 1")
	}
}

func TestReportResult_InProgressMatch(t *testing.T) {
	repo := newMockRepo()
	createTestBracket(repo)
	svc := NewMatchService(repo)
	ctx := context.Background()

	// Start match first
	svc.StartMatch(ctx, 1)

	// Now report result
	err := svc.ReportResult(ctx, 1, domain.MatchResult{WinnerID: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	match, _ := repo.GetByID(ctx, 1)
	if match.Status != domain.MatchCompleted {
		t.Errorf("expected completed, got %s", match.Status)
	}
}
