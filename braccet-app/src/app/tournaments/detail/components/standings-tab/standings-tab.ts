import { Component, computed, inject, signal, effect } from '@angular/core';
import { NgClass } from '@angular/common';
import { Tournament } from '../../../../models/tournament.model';
import { SwissBracketState, SwissStanding, SwissParticipantStatus, EliminationStandingsResponse, EliminationStanding } from '../../../../models/bracket.model';
import { BracketService } from '../../../../services/bracket.service';
import { TournamentUIService } from '../../../../services/tournament-ui.service';
import { AuthService } from '../../../../services/auth.service';

@Component({
  selector: 'app-standings-tab',
  imports: [NgClass],
  templateUrl: './standings-tab.html',
  styleUrl: './standings-tab.css'
})
export class StandingsTab {
  private bracketService = inject(BracketService);
  private tournamentUI = inject(TournamentUIService);
  private authService = inject(AuthService);

  // Read from service
  tournament = computed(() => this.tournamentUI.tournament()!);

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
  isMultiStageFormat = computed(() => this.tournament().format === 'multi_stage');

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

  // Swiss threshold info
  winsToAdvance = computed(() => this.swissBracketState()?.wins_to_advance);
  lossesToEliminate = computed(() => this.swissBracketState()?.losses_to_eliminate);
  isThresholdMode = computed(() =>
    this.winsToAdvance() !== undefined || this.lossesToEliminate() !== undefined
  );

  // Format display name
  formatDisplayName = computed(() => {
    const format = this.tournament().format;
    switch (format) {
      case 'swiss': return 'Swiss';
      case 'single_elimination': return 'Single Elimination';
      case 'double_elimination': return 'Double Elimination';
      case 'multi_stage': return 'Multi-Stage';
      default: return format;
    }
  });

  // Get current user for highlighting
  currentUser = computed(() => this.authService.user());

  constructor() {
    // Load standings when tournament changes
    effect(() => {
      const t = this.tournamentUI.tournament();
      if (t && (t.status === 'in_progress' || t.status === 'completed')) {
        this.loadStandings(t);
      }
    });

    // Reload when refresh key changes
    effect(() => {
      const key = this.tournamentUI.bracketRefreshKey();
      const t = this.tournamentUI.tournament();
      if (t && key > 0 && (t.status === 'in_progress' || t.status === 'completed')) {
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

  // Swiss threshold mode helpers
  getStatusClass(status?: SwissParticipantStatus): string {
    switch (status) {
      case 'advanced': return 'status-advanced';
      case 'eliminated': return 'status-eliminated';
      default: return 'status-active';
    }
  }

  getStatusText(status?: SwissParticipantStatus): string {
    switch (status) {
      case 'advanced': return 'Advanced';
      case 'eliminated': return 'Eliminated';
      default: return 'Active';
    }
  }

  getThresholdProgress(standing: SwissStanding): { winsProgress?: string; lossesProgress?: string } {
    const result: { winsProgress?: string; lossesProgress?: string } = {};

    const winsThreshold = this.winsToAdvance();
    const lossesThreshold = this.lossesToEliminate();

    if (winsThreshold !== undefined) {
      const matchWins = standing.match_wins ?? standing.wins;
      result.winsProgress = `${matchWins}/${winsThreshold}`;
    }

    if (lossesThreshold !== undefined) {
      result.lossesProgress = `${standing.losses}/${lossesThreshold}`;
    }

    return result;
  }
}
