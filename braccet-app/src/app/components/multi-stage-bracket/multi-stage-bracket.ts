import { Component, input, output, signal, computed, effect, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Tournament, TournamentStage, StageGroup } from '../../models/tournament.model';
import { Match, GroupBracketState, GroupStanding, BracketState, SwissBracketState, BracketStage, BracketType } from '../../models/bracket.model';
import { TournamentService } from '../../services/tournament.service';
import { BracketService } from '../../services/bracket.service';
import { TournamentUIService } from '../../services/tournament-ui.service';
import { BracketViewer } from '../bracket-viewer/bracket-viewer';
import { DoubleElimBracket } from '../double-elim-bracket/double-elim-bracket';
import { SwissBracket } from '../swiss-bracket/swiss-bracket';
import { GroupStandingsComponent } from '../group-standings/group-standings';

@Component({
  selector: 'app-multi-stage-bracket',
  standalone: true,
  imports: [
    CommonModule,
    BracketViewer,
    DoubleElimBracket,
    SwissBracket,
    GroupStandingsComponent
  ],
  templateUrl: './multi-stage-bracket.html',
  styleUrls: ['./multi-stage-bracket.css']
})
export class MultiStageBracketComponent {
  private tournamentService = inject(TournamentService);
  private bracketService = inject(BracketService);
  private tournamentUI = inject(TournamentUIService);

  // Inputs - still used for backwards compatibility with bracket-tab
  tournament = input.required<Tournament>();
  stages = input.required<TournamentStage[]>();
  isOrganizer = input(false);
  refreshKey = input(0);
  selectedGroupIdx = input<number | null>(null); // External group selection

  // Outputs
  matchClicked = output<Match>();
  matchEditClicked = output<Match>();
  matchReopened = output<Match>();
  stageClicked = output<{ round: number; stage: BracketStage; bracketType?: BracketType; stageId?: number; groupId?: number }>();
  swissStageClicked = output<{ round: number; stage: BracketStage; stageId?: number; groupId?: number }>();
  swissReseedClicked = output<{ round: number; stageId?: number; groupId?: number }>();
  finalsReseedClicked = output<{ stageId: number; format: string }>();
  stageReseedClicked = output<{ stageId: number; format: string }>();
  advanceStageClicked = output<void>();
  startStageClicked = output<void>();
  currentStagesChanged = output<BracketStage[]>();
  finalsCompleteChanged = output<boolean>();
  activeStageChanged = output<{ stageId: number; format: string; stageType: string }>();
  groupsChanged = output<{ groups: StageGroup[]; selectedIndex: number; stats: Map<number, { completed: number; total: number; isComplete: boolean }> }>();

  // State
  activeStage = signal<TournamentStage | null>(null);
  groups = signal<StageGroup[]>([]);
  selectedGroupIndex = signal(0);
  groupBrackets = signal<Map<number, GroupBracketState>>(new Map());
  groupSwissBrackets = signal<Map<number, SwissBracketState>>(new Map());
  groupStandings = signal<Map<number, GroupStanding[]>>(new Map());
  finalBracketState = signal<BracketState | null>(null);
  finalSwissBracketState = signal<SwissBracketState | null>(null);
  loading = signal(false);
  error = signal<string | null>(null);
  advancingGroupRound = signal(false);

  // Computed
  activeGroup = computed(() => {
    const groupList = this.groups();
    const index = this.selectedGroupIndex();
    return groupList[index] || null;
  });

  currentGroupBracket = computed(() => {
    const group = this.activeGroup();
    if (!group) return null;
    return this.groupBrackets().get(group.id) || null;
  });

  currentGroupStandings = computed(() => {
    const group = this.activeGroup();
    if (!group) return [];
    return this.groupStandings().get(group.id) || [];
  });

  currentGroupSwissBracket = computed(() => {
    const group = this.activeGroup();
    if (!group) return null;
    return this.groupSwissBrackets().get(group.id) || null;
  });

  // Computed: group completion stats for displaying progress indicators
  groupCompletionStats = computed(() => {
    const stage = this.activeStage();
    const groupList = this.groups();
    const brackets = this.groupBrackets();
    const swissBrackets = this.groupSwissBrackets();

    const stats = new Map<number, { completed: number; total: number; isComplete: boolean }>();

    for (const group of groupList) {
      let completed = 0;
      let total = 0;
      let isComplete = false;

      if (stage?.format === 'swiss') {
        const bracket = swissBrackets.get(group.id);
        if (bracket) {
          // Count matches for progress display
          for (const match of bracket.matches) {
            if (match.status !== 'bye') {
              total++;
              if (match.status === 'completed') {
                completed++;
              }
            }
          }

          // For Swiss, check multiple completion conditions:
          // 1. Backend has already marked it complete
          if (bracket.is_complete) {
            isComplete = true;
          } else {
            // 2. Check if current round is done and no more rounds can be generated
            const currentRound = bracket.current_round;
            const currentRoundMatches = bracket.matches.filter(m => m.round === currentRound);
            const allCurrentRoundDone = currentRoundMatches.length > 0 &&
              currentRoundMatches.every(m => m.status === 'completed' || m.status === 'bye');

            if (allCurrentRoundDone) {
              // Fixed rounds mode: complete when currentRound >= totalRounds
              if (bracket.total_rounds > 0 && currentRound >= bracket.total_rounds) {
                isComplete = true;
              }
              // Threshold mode: complete when fewer than 2 active participants
              if (bracket.total_rounds === 0) {
                const activeCount = bracket.standings.filter(s => s.status === 'active').length;
                if (activeCount < 2) {
                  isComplete = true;
                }
              }
            }
          }
        }
      } else {
        const bracket = brackets.get(group.id);
        if (bracket) {
          for (const match of bracket.matches) {
            if (match.status !== 'bye') {
              total++;
              if (match.status === 'completed') {
                completed++;
              }
            }
          }
          // For elimination brackets, complete when all matches are done
          isComplete = total > 0 && completed === total;
        }
      }

      stats.set(group.id, { completed, total, isComplete });
    }

    return stats;
  });

  finalBracket = computed(() => this.finalBracketState());
  finalSwissBracket = computed(() => this.finalSwissBracketState());

  // Check if finals bracket is complete
  finalsComplete = computed(() => {
    // Find the final stage
    const finalStage = this.stages().find(s => s.stage_type === 'final');
    if (!finalStage) return false;

    // Check if final bracket is complete based on format
    if (finalStage.format === 'swiss') {
      const swissBracket = this.finalSwissBracketState();
      if (!swissBracket) return false;
      // Swiss finals complete when is_complete flag is set
      return swissBracket.is_complete;
    } else {
      const bracket = this.finalBracketState();
      if (!bracket) return false;
      return bracket.is_complete;
    }
  });

  // Current bracket stages - computed from the active group/final bracket
  currentBracketStages = computed(() => {
    const stage = this.activeStage();
    if (!stage) return [];

    if (stage.stage_type === 'final') {
      // For finals, use final bracket stages
      if (stage.format === 'swiss') {
        return this.finalSwissBracketState()?.stages ?? [];
      }
      return this.finalBracketState()?.stages ?? [];
    }

    // For group stages, use the current group's bracket stages
    const group = this.activeGroup();
    if (!group) return [];

    if (stage.format === 'swiss') {
      return this.groupSwissBrackets().get(group.id)?.stages ?? [];
    }
    return this.groupBrackets().get(group.id)?.stages ?? [];
  });

  sortedStages = computed(() => {
    // Sort stages: group stages (1, 2, 3) first, then final stage (0)
    return [...this.stages()].sort((a, b) => {
      if (a.stage_order === 0) return 1;
      if (b.stage_order === 0) return -1;
      return a.stage_order - b.stage_order;
    });
  });

  // Check if all groups in the current stage are complete
  allGroupsComplete = computed(() => {
    const groupList = this.groups();
    const stats = this.groupCompletionStats();

    if (groupList.length === 0) return false;

    // All groups must have brackets loaded and be complete
    for (const group of groupList) {
      const groupStats = stats.get(group.id);
      if (!groupStats || !groupStats.isComplete) {
        return false;
      }
    }
    return true;
  });

  canAdvanceStage = computed(() => {
    const stage = this.activeStage();
    if (!stage || !this.isOrganizer()) return false;
    // Can advance if stage is active, has groups, and ALL groups are complete
    return stage.is_active && this.groups().length > 0 && this.allGroupsComplete();
  });

  canStartStage = computed(() => {
    const stage = this.activeStage();
    if (!stage || !this.isOrganizer()) return false;
    // Can start if stage is active but no groups exist yet
    return stage.is_active && this.groups().length === 0;
  });

  // Check if the current group's Swiss round can be advanced (all matches done, not yet complete)
  canAdvanceGroupSwissRound = computed(() => {
    const stage = this.activeStage();
    if (!stage || stage.format !== 'swiss' || !this.isOrganizer()) return false;

    const group = this.activeGroup();
    if (!group) return false;

    const bracket = this.groupSwissBrackets().get(group.id);
    if (!bracket || bracket.is_complete) return false;

    // Check if all matches in current round are completed
    const currentRound = bracket.current_round;
    const currentRoundMatches = bracket.matches.filter(m => m.round === currentRound);
    if (currentRoundMatches.length === 0) return false;

    const allMatchesComplete = currentRoundMatches.every(m => m.status === 'completed' || m.status === 'bye');
    if (!allMatchesComplete) return false;

    // For fixed rounds mode, check if this is the last round
    if (bracket.total_rounds > 0 && currentRound >= bracket.total_rounds) {
      return false; // No more rounds to generate
    }

    // For threshold mode, check if there are enough active participants for another round
    if (bracket.total_rounds === 0) {
      // Threshold mode: need at least 2 active participants
      const activeCount = bracket.standings.filter(s => s.status === 'active').length;
      if (activeCount < 2) {
        return false; // Not enough participants for another round
      }
    }

    return true;
  });

  constructor() {
    // Load active stage and groups when stages change
    effect(() => {
      const stageList = this.stages();
      const refreshKey = this.refreshKey();
      if (stageList.length > 0) {
        const active = stageList.find(s => s.is_active);
        this.activeStage.set(active || stageList[0]);
        if (active) {
          // Preserve current group index when refreshing (refreshKey > 0 means it's a refresh)
          const preserveIndex = refreshKey > 0;
          this.loadGroups(active, preserveIndex);
        }
      }
    });

    // Watch for stage selection from UI service (header cards)
    effect(() => {
      const selectedId = this.tournamentUI.selectedStageId();
      if (selectedId !== null) {
        const stage = this.stages().find(s => s.id === selectedId);
        if (stage && stage.id !== this.activeStage()?.id) {
          this.selectStage(stage);
        }
      }
    });

    // Emit current bracket stages whenever they change
    effect(() => {
      const bracketStages = this.currentBracketStages();
      this.currentStagesChanged.emit(bracketStages);
    });

    // Emit finals completion status whenever it changes
    effect(() => {
      const isComplete = this.finalsComplete();
      this.finalsCompleteChanged.emit(isComplete);
    });

    // Emit active stage info whenever it changes
    effect(() => {
      const stage = this.activeStage();
      if (stage) {
        this.activeStageChanged.emit({
          stageId: stage.id,
          format: stage.format,
          stageType: stage.stage_type
        });
      }
    });

    // Emit groups data whenever groups or selection changes
    effect(() => {
      const groupList = this.groups();
      const selectedIdx = this.selectedGroupIndex();
      const stats = this.groupCompletionStats();
      this.groupsChanged.emit({
        groups: groupList,
        selectedIndex: selectedIdx,
        stats: stats
      });
    });

    // Watch for external group selection
    effect(() => {
      const externalIdx = this.selectedGroupIdx();
      if (externalIdx !== null && externalIdx !== this.selectedGroupIndex()) {
        this.selectGroup(externalIdx);
      }
    });
  }

  selectStage(stage: TournamentStage): void {
    this.activeStage.set(stage);
    this.selectedGroupIndex.set(0);
    this.loadGroups(stage, false);
    // Sync to UI service so header cards show selection
    this.tournamentUI.selectStage(stage.id);
  }

  selectGroup(index: number): void {
    this.selectedGroupIndex.set(index);
    const group = this.groups()[index];
    const stage = this.activeStage();
    if (group && stage) {
      const hasExistingBracket = stage.format === 'swiss'
        ? this.groupSwissBrackets().has(group.id)
        : this.groupBrackets().has(group.id);
      if (!hasExistingBracket) {
        this.loadGroupBracket(group.id, stage.format);
      }
    }
  }

  private loadGroups(stage: TournamentStage, preserveIndex: boolean): void {
    // For final stage, load the final bracket directly
    if (stage.stage_type === 'final') {
      this.groups.set([]);
      // Only show loading indicator on initial load, not on refresh
      // This prevents destroying/recreating the bracket component which resets panzoom
      if (!preserveIndex) {
        this.loading.set(true);
      }
      this.loadFinalBracket(stage, preserveIndex);
      return;
    }

    // Only show loading indicator on initial load, not on refresh
    // This prevents destroying/recreating the bracket component which resets panzoom
    if (!preserveIndex) {
      this.loading.set(true);
    }
    this.tournamentService.getGroups(this.tournament().slug, stage.id).subscribe({
      next: (groups) => {
        this.groups.set(groups);
        // Only reset to first group if not preserving index (e.g., on initial load or stage change)
        if (!preserveIndex) {
          this.selectedGroupIndex.set(0);
        }
        // Load brackets for ALL groups so completion stats are visible
        for (const group of groups) {
          this.loadGroupBracket(group.id, stage.format);
        }
        this.loading.set(false);
      },
      error: (err) => {
        this.error.set(err.error?.error || 'Failed to load groups');
        this.loading.set(false);
      }
    });
  }

  private loadFinalBracket(stage: TournamentStage, isRefresh = false): void {
    if (stage.format === 'swiss') {
      // TODO: Add stage-specific Swiss bracket endpoint when needed
      this.bracketService.getSwissBracket(this.tournament().id).subscribe({
        next: (bracket) => {
          this.finalSwissBracketState.set(bracket);
          this.loading.set(false);
        },
        error: () => {
          // No bracket yet - finals not started
          this.finalSwissBracketState.set(null);
          this.loading.set(false);
        }
      });
    } else {
      // Use stage-specific endpoint to only get matches for this stage
      this.bracketService.getStageBracket(this.tournament().id, stage.id).subscribe({
        next: (bracket) => {
          this.finalBracketState.set(bracket);
          this.loading.set(false);
        },
        error: () => {
          // No bracket yet - finals not started
          this.finalBracketState.set(null);
          this.loading.set(false);
        }
      });
    }
  }

  private loadGroupBracket(groupId: number, format?: string): void {
    const stage = this.activeStage();
    if (!stage) {
      console.log('No active stage to load bracket for');
      return;
    }

    // For Swiss format groups, load Swiss bracket
    if (format === 'swiss') {
      this.bracketService.getGroupSwissBracket(this.tournament().id, stage.id, groupId).subscribe({
        next: (bracket) => {
          const brackets = new Map(this.groupSwissBrackets());
          brackets.set(groupId, bracket);
          this.groupSwissBrackets.set(brackets);
        },
        error: (err) => {
          console.log('No Swiss bracket for group', groupId, err);
        }
      });
      return;
    }

    this.bracketService.getGroupBracket(this.tournament().id, stage.id, groupId).subscribe({
      next: (bracket) => {
        const brackets = new Map(this.groupBrackets());
        brackets.set(groupId, bracket);
        this.groupBrackets.set(brackets);

        if (bracket.standings) {
          const standings = new Map(this.groupStandings());
          standings.set(groupId, bracket.standings);
          this.groupStandings.set(standings);
        }
      },
      error: (err) => {
        // Bracket may not exist yet
        console.log('No bracket for group', groupId, err);
      }
    });
  }

  onMatchClicked(match: Match): void {
    this.matchClicked.emit(match);
  }

  onMatchEditClicked(match: Match): void {
    this.matchEditClicked.emit(match);
  }

  onMatchReopened(match: Match): void {
    this.matchReopened.emit(match);
  }

  onStageClicked(event: { round: number; stage: BracketStage; bracketType?: BracketType }): void {
    // Include stageId and groupId from active context if in a group bracket
    const stage = this.activeStage();
    const group = this.activeGroup();
    this.stageClicked.emit({
      ...event,
      stageId: stage?.id,
      groupId: group?.id
    });
  }

  onSwissStageClicked(event: { round: number; stage: BracketStage }): void {
    // Include stageId and groupId from active context if in a group bracket
    const stage = this.activeStage();
    const group = this.activeGroup();
    this.swissStageClicked.emit({
      ...event,
      stageId: stage?.id,
      groupId: group?.id
    });
  }

  onSwissReseedClicked(event: { round: number }): void {
    // Include stageId and groupId from active context if in a group bracket
    const stage = this.activeStage();
    const group = this.activeGroup();
    this.swissReseedClicked.emit({
      ...event,
      stageId: stage?.id,
      groupId: group?.id
    });
  }

  onFinalsReseedClicked(): void {
    const stage = this.activeStage();
    if (stage?.stage_type === 'final') {
      this.finalsReseedClicked.emit({
        stageId: stage.id,
        format: stage.format
      });
    }
  }

  onStageReseedClicked(): void {
    const stage = this.activeStage();
    if (stage) {
      this.stageReseedClicked.emit({
        stageId: stage.id,
        format: stage.format
      });
    }
  }

  onAdvanceStage(): void {
    this.advanceStageClicked.emit();
  }

  onStartStage(): void {
    this.startStageClicked.emit();
  }

  advanceGroupSwissRound(): void {
    const stage = this.activeStage();
    const group = this.activeGroup();
    if (!stage || !group || !this.canAdvanceGroupSwissRound() || this.advancingGroupRound()) return;

    this.advancingGroupRound.set(true);
    this.error.set(null);

    this.bracketService.advanceGroupSwissRound(this.tournament().id, stage.id, group.id).subscribe({
      next: () => {
        this.advancingGroupRound.set(false);
        // Reload the group bracket to show new matches
        this.loadGroupBracket(group.id, stage.format);
      },
      error: (err) => {
        this.error.set(err.error?.error || 'Failed to advance round');
        this.advancingGroupRound.set(false);
      }
    });
  }

  getStageLabel(stage: TournamentStage): string {
    if (stage.stage_type === 'final') return 'Finals';
    return `Stage ${stage.stage_order}`;
  }

  getStageFormatLabel(stage: TournamentStage): string {
    switch (stage.format) {
      case 'single_elimination': return 'Single Elim';
      case 'double_elimination': return 'Double Elim';
      case 'swiss': return 'Swiss';
      default: return stage.format;
    }
  }

  getGroupStats(groupId: number): { completed: number; total: number; isComplete: boolean } | null {
    return this.groupCompletionStats().get(groupId) || null;
  }
}
