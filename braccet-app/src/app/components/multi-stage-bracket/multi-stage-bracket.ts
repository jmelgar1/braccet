import { Component, input, output, signal, computed, effect, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Tournament, TournamentStage, StageGroup } from '../../models/tournament.model';
import { Match, GroupBracketState, GroupStanding, BracketState, SwissBracketState } from '../../models/bracket.model';
import { TournamentService } from '../../services/tournament.service';
import { BracketService } from '../../services/bracket.service';
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

  // Inputs
  tournament = input.required<Tournament>();
  stages = input.required<TournamentStage[]>();
  isOrganizer = input(false);
  refreshKey = input(0);

  // Outputs
  matchClicked = output<Match>();
  matchEditClicked = output<Match>();
  matchReopened = output<Match>();
  stageClicked = output<{ stageId: number; round: number; bracketType: string }>();
  advanceStageClicked = output<void>();
  startStageClicked = output<void>();

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

  finalBracket = computed(() => this.finalBracketState());
  finalSwissBracket = computed(() => this.finalSwissBracketState());

  sortedStages = computed(() => {
    // Sort stages: group stages (1, 2, 3) first, then final stage (0)
    return [...this.stages()].sort((a, b) => {
      if (a.stage_order === 0) return 1;
      if (b.stage_order === 0) return -1;
      return a.stage_order - b.stage_order;
    });
  });

  canAdvanceStage = computed(() => {
    const stage = this.activeStage();
    if (!stage || !this.isOrganizer()) return false;
    // Can advance if stage is active and has groups (groups are complete check would need bracket data)
    return stage.is_active && this.groups().length > 0;
  });

  canStartStage = computed(() => {
    const stage = this.activeStage();
    if (!stage || !this.isOrganizer()) return false;
    // Can start if stage is active but no groups exist yet
    return stage.is_active && this.groups().length === 0;
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
          this.loadGroups(active);
        }
      }
    });
  }

  selectStage(stage: TournamentStage): void {
    this.activeStage.set(stage);
    this.selectedGroupIndex.set(0);
    this.loadGroups(stage);
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

  private loadGroups(stage: TournamentStage): void {
    // For final stage, load the final bracket directly
    if (stage.stage_type === 'final') {
      this.groups.set([]);
      this.loading.set(true);
      this.loadFinalBracket(stage);
      return;
    }

    this.loading.set(true);
    this.tournamentService.getGroups(this.tournament().slug, stage.id).subscribe({
      next: (groups) => {
        this.groups.set(groups);
        this.selectedGroupIndex.set(0);
        if (groups.length > 0) {
          this.loadGroupBracket(groups[0].id, stage.format);
        }
        this.loading.set(false);
      },
      error: (err) => {
        this.error.set(err.error?.error || 'Failed to load groups');
        this.loading.set(false);
      }
    });
  }

  private loadFinalBracket(stage: TournamentStage): void {
    if (stage.format === 'swiss') {
      this.bracketService.getSwissBracket(this.tournament().id).subscribe({
        next: (bracket) => {
          this.finalSwissBracketState.set(bracket);
          this.loading.set(false);
        },
        error: () => {
          this.loading.set(false);
        }
      });
    } else {
      this.bracketService.getBracket(this.tournament().id).subscribe({
        next: (bracket) => {
          this.finalBracketState.set(bracket);
          this.loading.set(false);
        },
        error: () => {
          this.loading.set(false);
        }
      });
    }
  }

  private loadGroupBracket(groupId: number, format?: string): void {
    // For Swiss format groups, load Swiss bracket
    if (format === 'swiss') {
      this.bracketService.getGroupSwissBracket(this.tournament().id, groupId).subscribe({
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

    this.bracketService.getGroupBracket(this.tournament().id, groupId).subscribe({
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

  onStageClicked(event: { stageId: number; round: number; bracketType: string }): void {
    this.stageClicked.emit(event);
  }

  onAdvanceStage(): void {
    this.advanceStageClicked.emit();
  }

  onStartStage(): void {
    this.startStageClicked.emit();
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
}
