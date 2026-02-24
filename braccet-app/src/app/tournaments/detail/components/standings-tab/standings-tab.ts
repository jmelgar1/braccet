import { Component, input, computed, inject, signal, effect } from '@angular/core';
import { NgClass } from '@angular/common';
import { Tournament } from '../../../../models/tournament.model';
import { SwissBracketState, SwissStanding, EliminationStandingsResponse, EliminationStanding } from '../../../../models/bracket.model';
import { BracketService } from '../../../../services/bracket.service';
import { AuthService } from '../../../../services/auth.service';

@Component({
  selector: 'app-standings-tab',
  imports: [NgClass],
  templateUrl: './standings-tab.html',
  styleUrl: './standings-tab.css'
})
export class StandingsTab {
  private bracketService = inject(BracketService);
  private authService = inject(AuthService);

  tournament = input.required<Tournament>();
  refreshKey = input(0);

  // State
  swissBracketState = signal<SwissBracketState | null>(null);
  eliminationStandings = signal<EliminationStandingsResponse | null>(null);
  loading = signal(false);
  error = signal('');

  // Computed: determine format
  isSwissFormat = computed(() => this.tournament().format === 'swiss');
  isEliminationFormat = computed(() => {
    const format = this.tournament().format;
    return format === 'single_elimination' || format === 'double_elimination';
  });

  // Swiss standings
  swissStandings = computed(() => this.swissBracketState()?.standings ?? []);

  // Elimination standings
  elimStandings = computed(() => this.eliminationStandings()?.standings ?? []);

  // Tournament info for Swiss
  totalRounds = computed(() => this.swissBracketState()?.total_rounds ?? 0);
  currentRound = computed(() => this.swissBracketState()?.current_round ?? 0);
  isComplete = computed(() => {
    if (this.isSwissFormat()) {
      return this.swissBracketState()?.is_complete ?? false;
    }
    return this.eliminationStandings()?.is_complete ?? false;
  });

  // Format display name
  formatDisplayName = computed(() => {
    const format = this.tournament().format;
    switch (format) {
      case 'swiss': return 'Swiss';
      case 'single_elimination': return 'Single Elimination';
      case 'double_elimination': return 'Double Elimination';
      default: return format;
    }
  });

  // Get current user for highlighting
  currentUser = computed(() => this.authService.user());

  constructor() {
    // Load standings when tournament changes
    effect(() => {
      const t = this.tournament();
      if (t.status === 'in_progress' || t.status === 'completed') {
        this.loadStandings(t);
      }
    });

    // Reload when refresh key changes
    effect(() => {
      const key = this.refreshKey();
      const t = this.tournament();
      if (key > 0 && (t.status === 'in_progress' || t.status === 'completed')) {
        this.loadStandings(t);
      }
    });
  }

  private loadStandings(tournament: Tournament): void {
    this.loading.set(true);
    this.error.set('');

    if (tournament.format === 'swiss') {
      this.bracketService.getSwissBracket(tournament.id).subscribe({
        next: (state) => {
          this.swissBracketState.set(state);
          this.loading.set(false);
        },
        error: (err) => {
          this.error.set(err.error?.error || 'Failed to load standings');
          this.loading.set(false);
        }
      });
    } else {
      // Single or double elimination
      this.bracketService.getEliminationStandings(tournament.id).subscribe({
        next: (standings) => {
          this.eliminationStandings.set(standings);
          this.loading.set(false);
        },
        error: (err) => {
          this.error.set(err.error?.error || 'Failed to load standings');
          this.loading.set(false);
        }
      });
    }
  }

  // Calculate game differential (game wins - game losses)
  getGameDiff(standing: SwissStanding | EliminationStanding): string {
    const diff = standing.game_wins - standing.game_losses;
    if (diff > 0) return `+${diff}`;
    return diff.toString();
  }

  // Format W-L record
  getRecord(standing: SwissStanding | EliminationStanding): string {
    return `${standing.wins}-${standing.losses}`;
  }

  // Check if this is the current user's row
  isCurrentUser(standing: SwissStanding | EliminationStanding): boolean {
    // This would require knowing the user's participant ID
    // For now we don't have that mapping directly
    return false;
  }

  // Get rank badge class based on position
  getRankClass(rank: number): string {
    if (rank === 1) return 'rank-gold';
    if (rank === 2) return 'rank-silver';
    if (rank === 3) return 'rank-bronze';
    return '';
  }

  // Get placement badge class
  getPlacementClass(placement: string): string {
    if (placement === 'Champion') return 'placement-champion';
    if (placement === '2nd') return 'placement-second';
    if (placement === '3rd' || placement === '3rd-4th') return 'placement-third';
    return '';
  }
}
