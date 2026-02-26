import { Component, computed, inject, signal, effect } from '@angular/core';
import { NgClass } from '@angular/common';
import { Tournament, TournamentStage, StageGroup } from '../../../../models/tournament.model';
import { SwissBracketState, SwissStanding, SwissParticipantStatus, EliminationStandingsResponse, EliminationStanding } from '../../../../models/bracket.model';
import { BracketService } from '../../../../services/bracket.service';
import { TournamentService } from '../../../../services/tournament.service';
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
  private tournamentService = inject(TournamentService);
  private tournamentUI = inject(TournamentUIService);
  private authService = inject(AuthService);

  // Read from service
  tournament = computed(() => this.tournamentUI.tournament()!);

  // State
  swissBracketState = signal<SwissBracketState | null>(null);
  eliminationStandings = signal<EliminationStandingsResponse | null>(null);
  loading = signal(false);
  error = signal('');

  // Multi-stage state
  stages = signal<TournamentStage[]>([]);
  groups = signal<StageGroup[]>([]);
  activeStageIndex = signal(0);
  activeGroupIndex = signal(0);

  // Computed: determine format
  isSwissFormat = computed(() => this.tournament().format === 'swiss');
  isEliminationFormat = computed(() => {
    const format = this.tournament().format;
    return format === 'single_elimination' || format === 'double_elimination';
  });
  isMultiStageFormat = computed(() => this.tournament().format === 'multi_stage');

  // Multi-stage computed
  activeStage = computed(() => {
    const stageList = this.stages();
    const index = this.activeStageIndex();
    return stageList[index] || null;
  });

  activeGroup = computed(() => {
    const groupList = this.groups();
    const index = this.activeGroupIndex();
    return groupList[index] || null;
  });

  isGroupStage = computed(() => this.activeStage()?.stage_type === 'group');
  isFinalStage = computed(() => this.activeStage()?.stage_type === 'final');

  // Get display name for stage
  getStageName(stage: TournamentStage): string {
    if (stage.stage_type === 'final') {
      return 'Finals';
    }
    // For group stages, use stage_order
    const groupStages = this.stages().filter(s => s.stage_type === 'group');
    if (groupStages.length === 1) {
      return 'Group Stage';
    }
    return `Group Stage ${stage.stage_order}`;
  }

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
    if (this.isMultiStageFormat()) {
      const stage = this.activeStage();
      if (stage?.stage_type === 'group') {
        return this.swissBracketState()?.is_complete ?? false;
      }
      return this.eliminationStandings()?.is_complete ?? false;
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
    } else if (tournament.format === 'multi_stage') {
      // Load stages first
      this.tournamentService.getStages(tournament.slug).subscribe({
        next: (stages) => {
          // Sort stages: group stages first (by stage_order desc), then finals
          const sorted = [...stages].sort((a, b) => {
            if (a.stage_type === 'final' && b.stage_type !== 'final') return 1;
            if (a.stage_type !== 'final' && b.stage_type === 'final') return -1;
            return b.stage_order - a.stage_order; // Higher stage_order first for groups
          });
          this.stages.set(sorted);
          this.activeStageIndex.set(0);
          this.loadStageStandings(tournament);
        },
        error: (err) => {
          this.error.set(err.error?.error || 'Failed to load stages');
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

  private loadStageStandings(tournament: Tournament): void {
    const stage = this.activeStage();
    if (!stage) {
      this.loading.set(false);
      return;
    }

    if (stage.stage_type === 'group') {
      // Load groups for this stage
      this.tournamentService.getGroups(tournament.slug, stage.id).subscribe({
        next: (groups) => {
          // Sort by group_order
          const sorted = [...groups].sort((a, b) => a.group_order - b.group_order);
          this.groups.set(sorted);
          this.activeGroupIndex.set(0);
          this.loadGroupStandings(tournament, stage);
        },
        error: (err) => {
          this.error.set(err.error?.error || 'Failed to load groups');
          this.loading.set(false);
        }
      });
    } else {
      // Finals stage - load elimination standings
      this.groups.set([]);
      this.swissBracketState.set(null);

      // For finals, we need to get elimination standings filtered by stage
      // The getStageBracket returns bracket state, we can use that for now
      const elimFormat = stage.format === 'swiss' ? 'single_elimination' : stage.format;
      this.bracketService.getStageBracket(tournament.id, stage.id).subscribe({
        next: (bracketState) => {
          // Convert bracket state to elimination standings format
          // For now, just show the bracket state matches as standings
          // This might need a dedicated endpoint
          this.eliminationStandings.set({
            tournament_id: tournament.id,
            format: elimFormat as 'single_elimination' | 'double_elimination',
            is_complete: stage.is_complete,
            standings: [] // We'll need to compute this from matches or add an endpoint
          });
          this.loading.set(false);
        },
        error: (err) => {
          // If no bracket yet, show empty
          this.eliminationStandings.set({
            tournament_id: tournament.id,
            format: elimFormat as 'single_elimination' | 'double_elimination',
            is_complete: false,
            standings: []
          });
          this.loading.set(false);
        }
      });
    }
  }

  private loadGroupStandings(tournament: Tournament, stage: TournamentStage): void {
    const group = this.activeGroup();
    if (!group) {
      this.loading.set(false);
      return;
    }

    this.eliminationStandings.set(null);

    // Load Swiss bracket for this group (Swiss groups have standings)
    if (stage.format === 'swiss') {
      this.bracketService.getGroupSwissBracket(tournament.id, stage.id, group.id).subscribe({
        next: (state) => {
          this.swissBracketState.set(state);
          this.loading.set(false);
        },
        error: (err) => {
          // If no bracket yet for this group
          this.swissBracketState.set(null);
          this.loading.set(false);
        }
      });
    } else {
      // Elimination format groups - show elimination standings
      this.swissBracketState.set(null);
      this.bracketService.getGroupBracket(tournament.id, stage.id, group.id).subscribe({
        next: (state) => {
          // For elimination groups, we'd need to compute standings
          this.loading.set(false);
        },
        error: () => {
          this.loading.set(false);
        }
      });
    }
  }

  // Tab selection methods
  selectStage(index: number): void {
    if (index === this.activeStageIndex()) return;

    this.activeStageIndex.set(index);
    this.activeGroupIndex.set(0);
    this.swissBracketState.set(null);
    this.eliminationStandings.set(null);
    this.loading.set(true);

    const tournament = this.tournament();
    if (tournament) {
      this.loadStageStandings(tournament);
    }
  }

  selectGroup(index: number): void {
    if (index === this.activeGroupIndex()) return;

    this.activeGroupIndex.set(index);
    this.swissBracketState.set(null);
    this.loading.set(true);

    const tournament = this.tournament();
    const stage = this.activeStage();
    if (tournament && stage) {
      this.loadGroupStandings(tournament, stage);
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
