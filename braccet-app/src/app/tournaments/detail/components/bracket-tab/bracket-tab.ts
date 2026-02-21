import { Component, input, computed, inject, signal, effect } from '@angular/core';
import { Tournament, Participant } from '../../../../models/tournament.model';
import { BracketGeneratorService, BracketPreview } from '../../../../services/bracket-generator.service';
import { BracketService } from '../../../../services/bracket.service';
import { BracketState } from '../../../../models/bracket.model';
import { BracketViewer } from '../../../../components/bracket-viewer/bracket-viewer';

@Component({
  selector: 'app-bracket-tab',
  imports: [BracketViewer],
  templateUrl: './bracket-tab.html'
})
export class BracketTab {
  private bracketGenerator = inject(BracketGeneratorService);
  private bracketService = inject(BracketService);

  tournament = input.required<Tournament>();
  participants = input.required<Participant[]>();
  refreshKey = input(0);

  bracketState = signal<BracketState | null>(null);
  loadingBracket = signal(false);
  bracketError = signal('');

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
}
