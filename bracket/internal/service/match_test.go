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

func (r *mockMatchRepository) UpdateResult(ctx context.Context, matchID uint64, winnerID uint64) error {
	m, ok := r.matches[matchID]
	if !ok {
		return repository.ErrMatchNotFound
	}
	m.WinnerID = &winnerID
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

func (r *mockMatchRepository) SetParticipant(ctx context.Context, matchID uint64, slot int, participantID uint64, name string, iconURL string, seed int) error {
	m, ok := r.matches[matchID]
	if !ok {
		return repository.ErrMatchNotFound
	}
	if slot == 1 {
		m.Participant1ID = &participantID
		m.Participant1Name = &name
		if iconURL != "" {
			m.Participant1IconURL = &iconURL
		}
		m.Seed1 = &seed
	} else {
		m.Participant2ID = &participantID
		m.Participant2Name = &name
		if iconURL != "" {
			m.Participant2IconURL = &iconURL
		}
		m.Seed2 = &seed
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

func (r *mockMatchRepository) GetPendingByParticipant(ctx context.Context, tournamentID, participantID uint64) ([]*domain.Match, error) {
	var matches []*domain.Match
	for _, m := range r.matches {
		if m.TournamentID == tournamentID &&
			(m.Status == domain.MatchPending || m.Status == domain.MatchReady || m.Status == domain.MatchInProgress) &&
			((m.Participant1ID != nil && *m.Participant1ID == participantID) ||
				(m.Participant2ID != nil && *m.Participant2ID == participantID)) {
			matches = append(matches, m)
		}
	}
	return matches, nil
}

func (r *mockMatchRepository) UpdateForfeit(ctx context.Context, matchID uint64, winnerID uint64) error {
	m, ok := r.matches[matchID]
	if !ok {
		return repository.ErrMatchNotFound
	}
	m.WinnerID = &winnerID
	m.ForfeitWinnerID = &winnerID
	m.Status = domain.MatchCompleted
	return nil
}

func (r *mockMatchRepository) ReopenMatch(ctx context.Context, matchID uint64) error {
	m, ok := r.matches[matchID]
	if !ok {
		return repository.ErrMatchNotFound
	}
	m.WinnerID = nil
	m.ForfeitWinnerID = nil
	m.Status = domain.MatchReady
	return nil
}

func (r *mockMatchRepository) ClearParticipant(ctx context.Context, matchID uint64, slot int) error {
	m, ok := r.matches[matchID]
	if !ok {
		return repository.ErrMatchNotFound
	}
	if slot == 1 {
		m.Participant1ID = nil
		m.Participant1Name = nil
		m.Participant1IconURL = nil
		m.Seed1 = nil
	} else {
		m.Participant2ID = nil
		m.Participant2Name = nil
		m.Participant2IconURL = nil
		m.Seed2 = nil
	}
	return nil
}

func (r *mockMatchRepository) DeleteByTournament(ctx context.Context, tournamentID uint64) error {
	for id, m := range r.matches {
		if m.TournamentID == tournamentID {
			delete(r.matches, id)
		}
	}
	return nil
}

// mockSetRepository implements repository.SetRepository for testing
type mockSetRepository struct {
	sets map[uint64][]domain.Set
}

func newMockSetRepo() *mockSetRepository {
	return &mockSetRepository{
		sets: make(map[uint64][]domain.Set),
	}
}

func (r *mockSetRepository) GetByMatchID(ctx context.Context, matchID uint64) ([]domain.Set, error) {
	return r.sets[matchID], nil
}

func (r *mockSetRepository) GetByMatchIDs(ctx context.Context, matchIDs []uint64) (map[uint64][]domain.Set, error) {
	result := make(map[uint64][]domain.Set)
	for _, id := range matchIDs {
		if sets, ok := r.sets[id]; ok {
			result[id] = sets
		}
	}
	return result, nil
}

func (r *mockSetRepository) CreateBatch(ctx context.Context, matchID uint64, sets []domain.SetScore) error {
	domainSets := make([]domain.Set, len(sets))
	for i, s := range sets {
		domainSets[i] = domain.Set{
			MatchID:           matchID,
			SetNumber:         s.SetNumber,
			Participant1Score: s.Participant1Score,
			Participant2Score: s.Participant2Score,
		}
	}
	r.sets[matchID] = domainSets
	return nil
}

func (r *mockSetRepository) DeleteByMatchID(ctx context.Context, matchID uint64) error {
	delete(r.sets, matchID)
	return nil
}

// newTestService creates a MatchService with mock dependencies for testing.
// TournamentClient and CommunityClient are nil (not needed for basic match tests).
func newTestService(repo *mockMatchRepository) (MatchService, *mockSetRepository) {
	setRepo := newMockSetRepo()
	return NewMatchService(repo, setRepo, nil, nil), setRepo
}

// resultForP1 creates a MatchResult where participant 1 wins (2-0 in sets)
func resultForP1() domain.MatchResult {
	return domain.MatchResult{
		Sets: []domain.SetScore{
			{SetNumber: 1, Participant1Score: 2, Participant2Score: 0},
		},
	}
}

// resultForP2 creates a MatchResult where participant 2 wins (0-2 in sets)
func resultForP2() domain.MatchResult {
	return domain.MatchResult{
		Sets: []domain.SetScore{
			{SetNumber: 1, Participant1Score: 0, Participant2Score: 2},
		},
	}
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
	svc, _ := newTestService(repo)
	ctx := context.Background()

	// Report result for match 1 (seed 1 vs seed 4) - P1 wins
	err := svc.ReportResult(ctx, 1, resultForP1())
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
	svc, _ := newTestService(repo)
	ctx := context.Background()

	// Report result for match 1 - P1 wins
	svc.ReportResult(ctx, 1, resultForP1())

	// Report result for match 2 - P1 wins (which is participant 2 in this match)
	err := svc.ReportResult(ctx, 2, resultForP1())
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
	svc, _ := newTestService(repo)
	ctx := context.Background()

	// Try to report with tied sets (no winner)
	result := domain.MatchResult{
		Sets: []domain.SetScore{
			{SetNumber: 1, Participant1Score: 1, Participant2Score: 1},
		},
	}
	err := svc.ReportResult(ctx, 1, result)

	if err != ErrSetsTied {
		t.Errorf("expected ErrSetsTied, got %v", err)
	}
}

func TestReportResult_MatchNotReady(t *testing.T) {
	repo := newMockRepo()
	createTestBracket(repo)
	svc, _ := newTestService(repo)
	ctx := context.Background()

	// Try to report result for pending match (the final)
	err := svc.ReportResult(ctx, 3, resultForP1())

	if err != ErrMatchNotReady {
		t.Errorf("expected ErrMatchNotReady, got %v", err)
	}
}

func TestReportResult_AlreadyComplete(t *testing.T) {
	repo := newMockRepo()
	createTestBracket(repo)
	svc, _ := newTestService(repo)
	ctx := context.Background()

	// Report result once
	svc.ReportResult(ctx, 1, resultForP1())

	// Try to report again
	err := svc.ReportResult(ctx, 1, resultForP2())

	if err != ErrMatchAlreadyComplete {
		t.Errorf("expected ErrMatchAlreadyComplete, got %v", err)
	}
}

func TestStartMatch(t *testing.T) {
	repo := newMockRepo()
	createTestBracket(repo)
	svc, _ := newTestService(repo)
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
	svc, _ := newTestService(repo)
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
	svc, _ := newTestService(repo)
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
	svc, _ := newTestService(repo)
	ctx := context.Background()

	// Complete all matches - P1 wins match 1, P1 (which is player 2) wins match 2, P1 wins final
	svc.ReportResult(ctx, 1, resultForP1())
	svc.ReportResult(ctx, 2, resultForP1())
	svc.ReportResult(ctx, 3, resultForP1())

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
	svc, _ := newTestService(repo)
	ctx := context.Background()

	// Start match first
	svc.StartMatch(ctx, 1)

	// Now report result - P1 wins
	err := svc.ReportResult(ctx, 1, resultForP1())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	match, _ := repo.GetByID(ctx, 1)
	if match.Status != domain.MatchCompleted {
		t.Errorf("expected completed, got %s", match.Status)
	}
}
