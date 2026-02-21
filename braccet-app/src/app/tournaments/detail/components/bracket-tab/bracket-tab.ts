import { Component, input, computed, inject, signal, effect, ViewChild } from '@angular/core';
import { Tournament, Participant } from '../../../../models/tournament.model';
import { BracketGeneratorService, BracketPreview } from '../../../../services/bracket-generator.service';
import { BracketService } from '../../../../services/bracket.service';
import { BracketState, Match } from '../../../../models/bracket.model';
import { BracketViewer } from '../../../../components/bracket-viewer/bracket-viewer';
import { MatchResultModal, MatchResultEvent } from '../../../../components/match-result-modal/match-result-modal';

@Component({
  selector: 'app-bracket-tab',
  imports: [BracketViewer, MatchResultModal],
  templateUrl: './bracket-tab.html'
})
export class BracketTab {
  private bracketGenerator = inject(BracketGeneratorService);
  private bracketService = inject(BracketService);

  tournament = input.required<Tournament>();
  participants = input.required<Participant[]>();
  refreshKey = input(0);
  isOrganizer = input(false);

  bracketState = signal<BracketState | null>(null);
  loadingBracket = signal(false);
  bracketError = signal('');

  // Modal state
  selectedMatch = signal<Match | null>(null);
  showModal = signal(false);

  @ViewChild(MatchResultModal) matchModal?: MatchResultModal;

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

    return this.bracketGenerator.generatePreview(p);
  });

  isPreviewMode = computed(() => {
    const t = this.tournament();
    return t.status !== 'in_progress' && t.status !== 'completed';
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
    this.showModal.set(true);
  }

  closeModal(): void {
    this.selectedMatch.set(null);
    this.showModal.set(false);
  }

  onResultSubmitted(event: MatchResultEvent): void {
    this.bracketService.reportResult(event.matchId, {
      sets: event.sets
    }).subscribe({
      next: () => {
        this.closeModal();
        this.loadBracket(this.tournament().id);
      },
      error: (err) => {
        const errorMsg = err.error?.error || 'Failed to save result';
        this.matchModal?.setError(errorMsg);
      }
    });
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
}
