import { Component, input, computed, inject, signal, effect, ViewChild, output } from '@angular/core';
import { Tournament, Participant, TournamentStage } from '../../../../models/tournament.model';
import { BracketPreview } from '../../../../services/bracket-generator.service';
import { BracketService } from '../../../../services/bracket.service';
import { TournamentService } from '../../../../services/tournament.service';
import { BracketState, BracketStage, Match, BracketType, SwissBracketState } from '../../../../models/bracket.model';
import { BracketViewer } from '../../../../components/bracket-viewer/bracket-viewer';
import { DoubleElimBracket } from '../../../../components/double-elim-bracket/double-elim-bracket';
import { SwissBracket } from '../../../../components/swiss-bracket/swiss-bracket';
import { MultiStageBracketComponent } from '../../../../components/multi-stage-bracket/multi-stage-bracket';
import { MatchResultModal, MatchResultEvent } from '../../../../components/match-result-modal/match-result-modal';
import { EditStageModal } from '../../../../components/edit-stage-modal/edit-stage-modal';

@Component({
  selector: 'app-bracket-tab',
  imports: [BracketViewer, DoubleElimBracket, SwissBracket, MultiStageBracketComponent, MatchResultModal, EditStageModal],
  templateUrl: './bracket-tab.html'
})
export class BracketTab {
  private bracketService = inject(BracketService);
  private tournamentService = inject(TournamentService);

  tournament = input.required<Tournament>();
  participants = input.required<Participant[]>();
  refreshKey = input(0);
  isOrganizer = input(false);

  // Output for tournament ended event
  tournamentEnded = output<Tournament>();
  // Output for tournament reset event
  tournamentReset = output<Tournament>();

  bracketState = signal<BracketState | null>(null);
  swissBracketState = signal<SwissBracketState | null>(null);
  loadingBracket = signal(false);
  bracketError = signal('');
  advancingRound = signal(false);

  // Multi-stage state
  tournamentStages = signal<TournamentStage[]>([]);
  loadingStages = signal(false);
  multiStageRefreshKey = signal(0);

  // Preview state (loaded from backend API)
  previewState = signal<BracketPreview | null>(null);
  loadingPreview = signal(false);
  previewError = signal('');
  endingTournament = signal(false);
  resettingTournament = signal(false);
  showResetConfirm = signal(false);

  // Modal state
  selectedMatch = signal<Match | null>(null);
  showModal = signal(false);
  isEditMode = signal(false);

  // Stage modal state
  selectedStage = signal<BracketStage | null>(null);
  showStageModal = signal(false);
  hideStageNameField = signal(false); // True for Swiss brackets

  @ViewChild(MatchResultModal) matchModal?: MatchResultModal;

  // Stages computed property - use bracket state stages for actual bracket, preview stages for preview mode
  stages = computed(() => this.bracketState()?.stages ?? []);

  // Preview stages computed property - convert to BracketStage format for compatibility
  previewStages = computed((): BracketStage[] => {
    const preview = this.previewState();
    if (!preview?.stages) return [];
    return preview.stages.map(s => ({
      tournament_id: s.tournament_id,
      bracket_type: s.bracket_type,
      round: s.round,
      stage_name: s.stage_name,
      best_of: s.best_of
    }));
  });

  // Get bestOf for the selected match based on its round and bracket type
  selectedMatchBestOf = computed(() => {
    const match = this.selectedMatch();

    // Use Swiss stages if it's a Swiss bracket
    const swissState = this.swissBracketState();
    if (swissState && swissState.stages?.length > 0) {
      const swissStage = swissState.stages.find(s => s.round === match?.round);
      return swissStage?.best_of ?? 1;
    }

    // Use regular bracket stages
    const stagesData = this.stages();
    if (!match || stagesData.length === 0) return 1;

    // For double elimination, also match on bracket_type
    const stage = stagesData.find(s =>
      s.round === match.round &&
      (!match.bracket_type || s.bracket_type === match.bracket_type)
    );
    return stage?.best_of ?? 1;
  });

  // Check if tournament uses double elimination
  isDoubleElimination = computed(() => {
    const t = this.tournament();
    const bracket = this.bracketState();

    // Check bracket state format first (for in-progress tournaments)
    if (bracket?.format === 'double_elimination') {
      return true;
    }

    // Fall back to tournament format (for preview)
    return t.format === 'double_elimination';
  });

  // Check if tournament uses Swiss format
  isSwiss = computed(() => {
    const t = this.tournament();
    return t.format === 'swiss';
  });

  // Check if tournament uses multi-stage format
  isMultiStage = computed(() => {
    const t = this.tournament();
    return t.format === 'multi_stage';
  });

  // Check if we're on the final round of Swiss
  isFinalSwissRound = computed(() => {
    const swissState = this.swissBracketState();
    if (!swissState) return false;
    return swissState.current_round >= swissState.total_rounds;
  });

  // Check if Swiss round can be advanced (not on final round)
  canAdvanceSwissRound = computed(() => {
    const swissState = this.swissBracketState();
    if (!swissState || swissState.is_complete) return false;

    // Can't advance if we're on the final round - use End Tournament instead
    if (this.isFinalSwissRound()) return false;

    // Check if all matches in current round are completed
    const currentRound = swissState.current_round;
    const currentRoundMatches = swissState.matches.filter(m => m.round === currentRound);

    if (currentRoundMatches.length === 0) return false;

    return currentRoundMatches.every(m => m.status === 'completed');
  });

  // Show advance round button for Swiss tournaments (not on final round)
  showAdvanceRoundButton = computed(() => {
    const t = this.tournament();
    const swissState = this.swissBracketState();
    return this.isOrganizer() &&
           this.isSwiss() &&
           t.status === 'in_progress' &&
           !swissState?.is_complete &&
           !this.isFinalSwissRound();
  });

  // Preview loaded from backend API (same logic as actual bracket generation)
  preview = computed<BracketPreview | null>(() => {
    const t = this.tournament();

    // Only show preview if tournament is not in_progress/completed (no real bracket yet)
    if (t.status === 'in_progress' || t.status === 'completed') {
      return null;
    }

    return this.previewState();
  });

  isPreviewMode = computed(() => {
    const t = this.tournament();
    return t.status !== 'in_progress' && t.status !== 'completed';
  });

  canEndTournament = computed(() => {
    // For regular brackets (single/double elimination)
    const bracket = this.bracketState();
    if (bracket?.is_complete) return true;

    // For Swiss brackets: can end on final round when all matches are complete
    const swissState = this.swissBracketState();
    if (swissState && this.isFinalSwissRound()) {
      const currentRound = swissState.current_round;
      const currentRoundMatches = swissState.matches.filter(m => m.round === currentRound);
      if (currentRoundMatches.length > 0 && currentRoundMatches.every(m => m.status === 'completed')) {
        return true;
      }
    }

    return false;
  });

  showEndButton = computed(() => {
    const t = this.tournament();
    return this.isOrganizer() && t.status === 'in_progress';
  });

  showResetButton = computed(() => {
    const t = this.tournament();
    return this.isOrganizer() && t.status === 'in_progress';
  });

  constructor() {
    // Load actual bracket when tournament is in progress or completed
    effect(() => {
      const t = this.tournament();
      if (t.status === 'in_progress' || t.status === 'completed') {
        if (t.format === 'multi_stage') {
          this.loadStages(t.slug);
        } else if (t.format === 'swiss') {
          this.loadSwissBracket(t.id);
        } else {
          this.loadBracket(t.id);
        }
      }
    });

    // Reload bracket when refreshKey changes (e.g., after withdraw)
    effect(() => {
      const key = this.refreshKey();
      const t = this.tournament();
      // Only reload if key > 0 (not initial) and bracket is active
      if (key > 0 && (t.status === 'in_progress' || t.status === 'completed')) {
        if (t.format === 'multi_stage') {
          this.multiStageRefreshKey.update(k => k + 1);
        } else if (t.format === 'swiss') {
          this.loadSwissBracket(t.id);
        } else {
          this.loadBracket(t.id);
        }
      }
    });

    // Load preview from backend when tournament is in draft/registration mode
    effect(() => {
      const t = this.tournament();
      const p = this.participants();

      // Only load preview if tournament is not in_progress/completed
      if (t.status === 'in_progress' || t.status === 'completed') {
        this.previewState.set(null);
        return;
      }

      // Need at least 2 participants
      if (p.length < 2) {
        this.previewState.set(null);
        return;
      }

      this.loadPreview(t.id, t.format, p);
    });
  }

  private loadPreview(tournamentId: number, format: string, participants: Participant[]): void {
    this.loadingPreview.set(true);
    this.previewError.set('');

    const bracketFormat = format === 'double_elimination' ? 'double_elimination' : 'single_elimination';

    this.bracketService.getPreview(tournamentId, bracketFormat, participants).subscribe({
      next: (preview) => {
        this.previewState.set(preview);
        this.loadingPreview.set(false);
      },
      error: (err) => {
        this.previewError.set(err.error?.error || 'Failed to load preview');
        this.loadingPreview.set(false);
      }
    });
  }

  private loadBracket(tournamentId: number): void {
    this.loadingBracket.set(true);
    this.bracketError.set('');

    this.bracketService.getBracket(tournamentId).subscribe({
      next: (state) => {
        this.bracketState.set(state);
        this.loadingBracket.set(false);
      },
      error: (err) => {
        this.bracketError.set(err.error?.error || 'Failed to load bracket');
        this.loadingBracket.set(false);
      }
    });
  }

  private loadSwissBracket(tournamentId: number): void {
    this.loadingBracket.set(true);
    this.bracketError.set('');

    this.bracketService.getSwissBracket(tournamentId).subscribe({
      next: (state) => {
        this.swissBracketState.set(state);
        this.loadingBracket.set(false);
      },
      error: (err) => {
        this.bracketError.set(err.error?.error || 'Failed to load Swiss bracket');
        this.loadingBracket.set(false);
      }
    });
  }

  private loadStages(slug: string): void {
    this.loadingStages.set(true);
    this.bracketError.set('');

    this.tournamentService.getStages(slug).subscribe({
      next: (stages) => {
        this.tournamentStages.set(stages);
        this.loadingStages.set(false);
      },
      error: (err) => {
        this.bracketError.set(err.error?.error || 'Failed to load stages');
        this.loadingStages.set(false);
      }
    });
  }

  onMultiStageMatchClicked(match: Match): void {
    this.onMatchClicked(match);
  }

  onMultiStageMatchEditClicked(match: Match): void {
    this.onMatchEditClicked(match);
  }

  onMultiStageMatchReopened(match: Match): void {
    this.onMatchReopened(match);
  }

  onMultiStageAdvance(): void {
    const t = this.tournament();
    this.tournamentService.advanceStage(t.slug).subscribe({
      next: () => {
        this.loadStages(t.slug);
        this.multiStageRefreshKey.update(k => k + 1);
      },
      error: (err) => {
        this.bracketError.set(err.error?.error || 'Failed to advance stage');
      }
    });
  }

  onMultiStageStart(): void {
    const t = this.tournament();
    this.tournamentService.startStage(t.slug).subscribe({
      next: () => {
        this.loadStages(t.slug);
        this.multiStageRefreshKey.update(k => k + 1);
      },
      error: (err) => {
        this.bracketError.set(err.error?.error || 'Failed to start stage');
      }
    });
  }

  advanceSwissRound(): void {
    const t = this.tournament();
    if (!this.canAdvanceSwissRound() || this.advancingRound()) return;

    this.advancingRound.set(true);
    this.bracketError.set('');

    this.bracketService.advanceSwissRound(t.id).subscribe({
      next: () => {
        this.advancingRound.set(false);
        this.loadSwissBracket(t.id);
      },
      error: (err) => {
        this.bracketError.set(err.error?.error || 'Failed to advance round');
        this.advancingRound.set(false);
      }
    });
  }

  onMatchClicked(match: Match): void {
    this.selectedMatch.set(match);
    this.isEditMode.set(false);
    this.showModal.set(true);
  }

  onMatchEditClicked(match: Match): void {
    this.selectedMatch.set(match);
    this.isEditMode.set(true);
    this.showModal.set(true);
  }

  closeModal(): void {
    this.selectedMatch.set(null);
    this.showModal.set(false);
    this.isEditMode.set(false);
  }

  onResultSubmitted(event: MatchResultEvent): void {
    const request = { sets: event.sets };
    const t = this.tournament();

    const handleSuccess = () => {
      this.closeModal();
      if (t.format === 'multi_stage') {
        this.multiStageRefreshKey.update(k => k + 1);
      } else if (t.format === 'swiss') {
        this.loadSwissBracket(t.id);
      } else {
        this.loadBracket(t.id);
      }
    };

    const handleError = (err: { error?: { error?: string } }) => {
      const errorMsg = err.error?.error || 'Failed to save result';
      this.matchModal?.setError(errorMsg);
    };

    if (this.isEditMode()) {
      this.bracketService.editResult(event.matchId, request).subscribe({
        next: handleSuccess,
        error: handleError
      });
    } else {
      this.bracketService.reportResult(event.matchId, request).subscribe({
        next: handleSuccess,
        error: handleError
      });
    }
  }

  onMatchReopened(match: Match): void {
    const t = this.tournament();
    this.bracketService.reopenMatch(match.id).subscribe({
      next: () => {
        if (t.format === 'multi_stage') {
          this.multiStageRefreshKey.update(k => k + 1);
        } else if (t.format === 'swiss') {
          this.loadSwissBracket(t.id);
        } else {
          this.loadBracket(t.id);
        }
      },
      error: (err) => {
        this.bracketError.set(err.error?.error || 'Failed to reopen match');
      }
    });
  }

  endTournament(): void {
    const t = this.tournament();
    if (!this.canEndTournament()) return;

    this.endingTournament.set(true);
    this.bracketError.set('');

    // For Swiss brackets on final round, first call advanceRound to mark is_complete
    // then update the tournament status
    if (this.isSwiss() && this.isFinalSwissRound()) {
      this.bracketService.advanceSwissRound(t.id).subscribe({
        next: () => {
          // Now update tournament status to completed
          this.updateTournamentStatus(t.slug);
        },
        error: (err) => {
          this.bracketError.set(err.error?.error || 'Failed to complete Swiss bracket');
          this.endingTournament.set(false);
        }
      });
    } else {
      // Regular brackets - just update tournament status
      this.updateTournamentStatus(t.slug);
    }
  }

  private updateTournamentStatus(slug: string): void {
    this.tournamentService.updateTournament(slug, { status: 'completed' }).subscribe({
      next: (updatedTournament) => {
        this.endingTournament.set(false);
        this.tournamentEnded.emit(updatedTournament);
      },
      error: (err) => {
        this.bracketError.set(err.error?.error || 'Failed to end tournament');
        this.endingTournament.set(false);
      }
    });
  }

  onStageClicked(event: { round: number; stage: BracketStage; bracketType?: BracketType }): void {
    this.selectedStage.set(event.stage);
    this.hideStageNameField.set(false); // Show name field for regular brackets
    this.showStageModal.set(true);
  }

  onSwissStageClicked(event: { round: number; stage: BracketStage }): void {
    this.selectedStage.set(event.stage);
    this.hideStageNameField.set(true); // Hide name field for Swiss brackets
    this.showStageModal.set(true);
  }

  onStageUpdated(stage: BracketStage): void {
    this.showStageModal.set(false);
    this.selectedStage.set(null);
    this.hideStageNameField.set(false);

    // Reload the appropriate bracket type
    const t = this.tournament();
    if (t.format === 'swiss') {
      this.loadSwissBracket(t.id);
    } else {
      this.loadBracket(t.id);
    }
  }

  closeStageModal(): void {
    this.showStageModal.set(false);
    this.selectedStage.set(null);
    this.hideStageNameField.set(false);
  }

  showResetConfirmDialog(): void {
    this.showResetConfirm.set(true);
  }

  cancelReset(): void {
    this.showResetConfirm.set(false);
  }

  confirmResetTournament(): void {
    const t = this.tournament();
    if (!this.isOrganizer()) return;

    this.showResetConfirm.set(false);
    this.resettingTournament.set(true);
    this.bracketError.set('');

    this.tournamentService.updateTournament(t.slug, { status: 'registration' }).subscribe({
      next: (updatedTournament) => {
        this.resettingTournament.set(false);
        this.bracketState.set(null); // Clear bracket state
        this.tournamentReset.emit(updatedTournament);
      },
      error: (err) => {
        this.bracketError.set(err.error?.error || 'Failed to reset tournament');
        this.resettingTournament.set(false);
      }
    });
  }
}
