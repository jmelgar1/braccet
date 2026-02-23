import { Component, input, computed, inject, signal, effect, ViewChild, output } from '@angular/core';
import { Tournament, Participant } from '../../../../models/tournament.model';
import { BracketGeneratorService, BracketPreview } from '../../../../services/bracket-generator.service';
import { BracketService } from '../../../../services/bracket.service';
import { TournamentService } from '../../../../services/tournament.service';
import { BracketState, BracketStage, Match, BracketType } from '../../../../models/bracket.model';
import { BracketViewer } from '../../../../components/bracket-viewer/bracket-viewer';
import { DoubleElimBracket } from '../../../../components/double-elim-bracket/double-elim-bracket';
import { MatchResultModal, MatchResultEvent } from '../../../../components/match-result-modal/match-result-modal';
import { EditStageModal } from '../../../../components/edit-stage-modal/edit-stage-modal';

@Component({
  selector: 'app-bracket-tab',
  imports: [BracketViewer, DoubleElimBracket, MatchResultModal, EditStageModal],
  templateUrl: './bracket-tab.html'
})
export class BracketTab {
  private bracketGenerator = inject(BracketGeneratorService);
  private bracketService = inject(BracketService);
  private tournamentService = inject(TournamentService);

  tournament = input.required<Tournament>();
  participants = input.required<Participant[]>();
  refreshKey = input(0);
  isOrganizer = input(false);

  // Output for tournament ended event
  tournamentEnded = output<Tournament>();

  bracketState = signal<BracketState | null>(null);
  loadingBracket = signal(false);
  bracketError = signal('');
  endingTournament = signal(false);

  // Modal state
  selectedMatch = signal<Match | null>(null);
  showModal = signal(false);
  isEditMode = signal(false);

  // Stage modal state
  selectedStage = signal<BracketStage | null>(null);
  showStageModal = signal(false);

  @ViewChild(MatchResultModal) matchModal?: MatchResultModal;

  // Stages computed property
  stages = computed(() => this.bracketState()?.stages ?? []);

  // Get bestOf for the selected match based on its round and bracket type
  selectedMatchBestOf = computed(() => {
    const match = this.selectedMatch();
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

  // Preview is generated client-side from participants
  preview = computed<BracketPreview | null>(() => {
    const t = this.tournament();
    const p = this.participants();

    // Only show preview if tournament is not in_progress/completed (no real bracket yet)
    if (t.status === 'in_progress' || t.status === 'completed') {
      return null;
    }

    if (p.length < 2) {
      return null;
    }

    // Generate appropriate preview based on format
    if (t.format === 'double_elimination') {
      return this.bracketGenerator.generateDoubleElimPreview(p);
    }
    return this.bracketGenerator.generatePreview(p);
  });

  isPreviewMode = computed(() => {
    const t = this.tournament();
    return t.status !== 'in_progress' && t.status !== 'completed';
  });

  canEndTournament = computed(() => {
    const bracket = this.bracketState();
    return bracket?.is_complete ?? false;
  });

  showEndButton = computed(() => {
    const t = this.tournament();
    return this.isOrganizer() && t.status === 'in_progress';
  });

  constructor() {
    // Load actual bracket when tournament is in progress or completed
    effect(() => {
      const t = this.tournament();
      if (t.status === 'in_progress' || t.status === 'completed') {
        this.loadBracket(t.id);
      }
    });

    // Reload bracket when refreshKey changes (e.g., after withdraw)
    effect(() => {
      const key = this.refreshKey();
      const t = this.tournament();
      // Only reload if key > 0 (not initial) and bracket is active
      if (key > 0 && (t.status === 'in_progress' || t.status === 'completed')) {
        this.loadBracket(t.id);
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

    const handleSuccess = () => {
      this.closeModal();
      this.loadBracket(this.tournament().id);
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
    this.bracketService.reopenMatch(match.id).subscribe({
      next: () => {
        this.loadBracket(this.tournament().id);
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

    this.tournamentService.updateTournament(t.slug, { status: 'completed' }).subscribe({
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
    this.showStageModal.set(true);
  }

  onStageUpdated(stage: BracketStage): void {
    this.showStageModal.set(false);
    this.selectedStage.set(null);
    this.loadBracket(this.tournament().id);
  }

  closeStageModal(): void {
    this.showStageModal.set(false);
    this.selectedStage.set(null);
  }
}
