import { Component, computed, inject, signal, effect, ViewChild, output } from '@angular/core';
import { Tournament, TournamentStage, Participant, StageGroup } from '../../../../models/tournament.model';
import { BracketPreview } from '../../../../services/bracket-generator.service';
import { BracketService } from '../../../../services/bracket.service';
import { TournamentService } from '../../../../services/tournament.service';
import { TournamentUIService } from '../../../../services/tournament-ui.service';
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
  tournamentUI = inject(TournamentUIService);

  // Computed accessors from service
  tournament = computed(() => this.tournamentUI.tournament()!);
  participants = computed(() => this.tournamentUI.participants());
  isOrganizer = computed(() => this.tournamentUI.isOrganizer());

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
  multiStageBracketStages = signal<BracketStage[]>([]); // Track current group/final bracket stages
  multiStageFinalsComplete = signal(false); // Track if multi-stage finals are complete

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
  selectedStageId = signal<number | null>(null); // For multi-stage group brackets
  selectedGroupId = signal<number | null>(null); // For multi-stage group brackets
  showStageModal = signal(false);
  hideStageNameField = signal(false); // True for Swiss brackets

  // Reseed modal state
  showReseedConfirm = signal(false);
  reseedRound = signal(0);
  reseedingRound = signal(false);
  reseedStageId = signal<number | null>(null); // For multi-stage group brackets
  reseedGroupId = signal<number | null>(null); // For multi-stage group brackets

  // Finals reseed modal state
  showFinalsReseedConfirm = signal(false);
  reseedingFinals = signal(false);
  finalsReseedStageId = signal<number | null>(null);
  finalsReseedFormat = signal<string>('');

  // Stage reseed modal state (reseeds all groups in a stage)
  showStageReseedConfirm = signal(false);
  reseedingStage = signal(false);
  stageReseedStageId = signal<number | null>(null);
  stageReseedFormat = signal<string>('');

  // Active stage info for header reseed button (from multi-stage bracket)
  activeStageId = signal<number | null>(null);
  activeStageFormat = signal<string>('');
  activeStageType = signal<string>('');

  // Group tabs state (from multi-stage bracket)
  multiStageGroups = signal<StageGroup[]>([]);
  multiStageSelectedGroupIndex = signal(0);
  multiStageGroupStats = signal<Map<number, { completed: number; total: number; isComplete: boolean }>>(new Map());

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
    if (!match) return 1;

    // For multi-stage tournaments, use the multi-stage bracket stages
    if (this.isMultiStage()) {
      const multiStageStages = this.multiStageBracketStages();
      if (multiStageStages.length > 0) {
        const stage = multiStageStages.find(s =>
          s.round === match.round &&
          (!match.bracket_type || s.bracket_type === match.bracket_type)
        );
        return stage?.best_of ?? 1;
      }
      return 1;
    }

    // Use Swiss stages if it's a Swiss bracket
    const swissState = this.swissBracketState();
    if (swissState && swissState.stages?.length > 0) {
      const swissStage = swissState.stages.find(s => s.round === match.round);
      return swissStage?.best_of ?? 1;
    }

    // Use regular bracket stages
    const stagesData = this.stages();
    if (stagesData.length === 0) return 1;

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
    // For multi-stage tournaments: can end when finals bracket is complete
    if (this.isMultiStage()) {
      return this.multiStageFinalsComplete();
    }

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

  // Show reseed stage button for multi-stage tournaments with elimination format group stages
  showReseedStageButton = computed(() => {
    const t = this.tournament();
    if (!this.isOrganizer() || t.status !== 'in_progress' || !this.isMultiStage()) return false;

    // Only show for elimination formats (single/double), not Swiss
    const format = this.activeStageFormat();
    const stageType = this.activeStageType();
    return stageType === 'group' && (format === 'single_elimination' || format === 'double_elimination');
  });

  // Show group tabs for multi-stage tournaments with group stages
  showGroupTabs = computed(() => {
    return this.isMultiStage() &&
           this.activeStageType() === 'group' &&
           this.multiStageGroups().length > 0;
  });

  constructor() {
    // Load actual bracket when tournament is in progress or completed
    effect(() => {
      const t = this.tournamentUI.tournament();
      if (!t) return;
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
      const key = this.tournamentUI.bracketRefreshKey();
      const t = this.tournamentUI.tournament();
      if (!t) return;
      // Only reload if key > 0 (not initial) and bracket is active
      if (key > 0 && (t.status === 'in_progress' || t.status === 'completed')) {
        if (t.format === 'multi_stage') {
          this.multiStageRefreshKey.update(k => k + 1);
        } else if (t.format === 'swiss') {
          // Silent reload to preserve panzoom state
          this.loadSwissBracket(t.id, true);
        } else {
          // Silent reload to preserve panzoom state
          this.loadBracket(t.id, true);
        }
      }
    });

    // Load preview from backend when tournament is in draft/registration mode
    effect(() => {
      const t = this.tournamentUI.tournament();
      const p = this.tournamentUI.participants();
      if (!t) return;

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

  private loadBracket(tournamentId: number, silent = false): void {
    // Only show loading indicator on initial load, not on refresh
    // This prevents destroying/recreating the bracket component which resets panzoom
    if (!silent) {
      this.loadingBracket.set(true);
    }
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

  private loadSwissBracket(tournamentId: number, silent = false): void {
    // Only show loading indicator on initial load, not on refresh
    // This prevents destroying/recreating the bracket component which resets panzoom
    if (!silent) {
      this.loadingBracket.set(true);
    }
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

  onMultiStageStageClicked(event: { round: number; stage: BracketStage; bracketType?: BracketType; stageId?: number; groupId?: number }): void {
    // Store stageId and groupId for multi-stage group brackets
    this.selectedStageId.set(event.stageId ?? null);
    this.selectedGroupId.set(event.groupId ?? null);
    this.onStageClicked(event);
  }

  onMultiStageSwissStageClicked(event: { round: number; stage: BracketStage; stageId?: number; groupId?: number }): void {
    // Store stageId and groupId for multi-stage group brackets
    this.selectedStageId.set(event.stageId ?? null);
    this.selectedGroupId.set(event.groupId ?? null);
    this.onSwissStageClicked(event);
  }

  onMultiStageStagesChanged(stages: BracketStage[]): void {
    this.multiStageBracketStages.set(stages);
  }

  onMultiStageFinalsCompleteChanged(isComplete: boolean): void {
    this.multiStageFinalsComplete.set(isComplete);
  }

  onActiveStageChanged(event: { stageId: number; format: string; stageType: string }): void {
    this.activeStageId.set(event.stageId);
    this.activeStageFormat.set(event.format);
    this.activeStageType.set(event.stageType);
  }

  onGroupsChanged(event: { groups: StageGroup[]; selectedIndex: number; stats: Map<number, { completed: number; total: number; isComplete: boolean }> }): void {
    this.multiStageGroups.set(event.groups);
    this.multiStageSelectedGroupIndex.set(event.selectedIndex);
    this.multiStageGroupStats.set(event.stats);
  }

  selectMultiStageGroup(index: number): void {
    this.multiStageSelectedGroupIndex.set(index);
  }

  getGroupStats(groupId: number): { completed: number; total: number; isComplete: boolean } | null {
    return this.multiStageGroupStats().get(groupId) || null;
  }

  triggerStageReseed(): void {
    const stageId = this.activeStageId();
    const format = this.activeStageFormat();
    if (stageId !== null) {
      this.onStageReseedClicked({ stageId, format });
    }
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
        // Silent reload to preserve panzoom state
        this.loadSwissBracket(t.id, true);
      } else {
        // Silent reload to preserve panzoom state
        this.loadBracket(t.id, true);
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
          // Silent reload to preserve panzoom state
          this.loadSwissBracket(t.id, true);
        } else {
          // Silent reload to preserve panzoom state
          this.loadBracket(t.id, true);
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
    if (t.format === 'multi_stage') {
      this.multiStageRefreshKey.update(k => k + 1);
    } else if (t.format === 'swiss') {
      this.loadSwissBracket(t.id);
    } else {
      this.loadBracket(t.id);
    }
  }

  closeStageModal(): void {
    this.showStageModal.set(false);
    this.selectedStage.set(null);
    this.selectedStageId.set(null);
    this.selectedGroupId.set(null);
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

  // Reseed Swiss round methods
  onSwissReseedClicked(event: { round: number }): void {
    this.reseedRound.set(event.round);
    this.reseedStageId.set(null);
    this.reseedGroupId.set(null);
    this.showReseedConfirm.set(true);
  }

  onMultiStageSwissReseedClicked(event: { round: number; stageId?: number; groupId?: number }): void {
    this.reseedRound.set(event.round);
    this.reseedStageId.set(event.stageId ?? null);
    this.reseedGroupId.set(event.groupId ?? null);
    this.showReseedConfirm.set(true);
  }

  cancelReseed(): void {
    this.showReseedConfirm.set(false);
    this.reseedRound.set(0);
    this.reseedStageId.set(null);
    this.reseedGroupId.set(null);
  }

  confirmReseedRound(): void {
    const t = this.tournament();
    const round = this.reseedRound();
    const stageId = this.reseedStageId();
    const groupId = this.reseedGroupId();

    if (!this.isOrganizer() || round < 1) return;

    this.showReseedConfirm.set(false);
    this.reseedingRound.set(true);
    this.bracketError.set('');

    // Determine if this is a group reseed or regular Swiss reseed
    if (stageId !== null && groupId !== null) {
      // Multi-stage group Swiss reseed
      this.bracketService.reseedGroupSwissRound(t.id, stageId, groupId, round).subscribe({
        next: () => {
          this.reseedingRound.set(false);
          this.reseedRound.set(0);
          this.reseedStageId.set(null);
          this.reseedGroupId.set(null);
          this.multiStageRefreshKey.update(k => k + 1);
        },
        error: (err) => {
          this.bracketError.set(err.error?.error || 'Failed to reseed round');
          this.reseedingRound.set(false);
        }
      });
    } else {
      // Regular Swiss reseed
      this.bracketService.reseedSwissRound(t.id, round).subscribe({
        next: () => {
          this.reseedingRound.set(false);
          this.reseedRound.set(0);
          this.loadSwissBracket(t.id);
        },
        error: (err) => {
          this.bracketError.set(err.error?.error || 'Failed to reseed round');
          this.reseedingRound.set(false);
        }
      });
    }
  }

  // Helper computed for reseed warning messages
  isReseedCascade(): boolean {
    const swissState = this.swissBracketState();
    if (!swissState) return false;
    return this.reseedRound() < swissState.current_round;
  }

  reseedRoundHasResults(): boolean {
    const swissState = this.swissBracketState();
    if (!swissState) return false;
    const round = this.reseedRound();
    const roundMatches = swissState.matches.filter(m => m.round === round);
    return roundMatches.some(m => m.status === 'completed');
  }

  // Finals reseed methods
  onFinalsReseedClicked(event: { stageId: number; format: string }): void {
    this.finalsReseedStageId.set(event.stageId);
    this.finalsReseedFormat.set(event.format);
    this.showFinalsReseedConfirm.set(true);
  }

  cancelFinalsReseed(): void {
    this.showFinalsReseedConfirm.set(false);
    this.finalsReseedStageId.set(null);
    this.finalsReseedFormat.set('');
  }

  confirmFinalsReseed(): void {
    const t = this.tournament();
    const stageId = this.finalsReseedStageId();
    const format = this.finalsReseedFormat();

    if (!this.isOrganizer() || stageId === null) return;

    this.showFinalsReseedConfirm.set(false);
    this.reseedingFinals.set(true);
    this.bracketError.set('');

    this.bracketService.reseedFinalBracket(t.id, stageId, format).subscribe({
      next: () => {
        this.reseedingFinals.set(false);
        this.finalsReseedStageId.set(null);
        this.finalsReseedFormat.set('');
        this.multiStageRefreshKey.update(k => k + 1);
      },
      error: (err) => {
        this.bracketError.set(err.error?.error || 'Failed to reseed finals bracket');
        this.reseedingFinals.set(false);
      }
    });
  }

  // Stage reseed methods (reseeds all groups in a stage with proper cross-group distribution)
  onStageReseedClicked(event: { stageId: number; format: string }): void {
    this.stageReseedStageId.set(event.stageId);
    this.stageReseedFormat.set(event.format);
    this.showStageReseedConfirm.set(true);
  }

  cancelStageReseed(): void {
    this.showStageReseedConfirm.set(false);
    this.stageReseedStageId.set(null);
    this.stageReseedFormat.set('');
  }

  confirmStageReseed(): void {
    const t = this.tournament();
    const stageId = this.stageReseedStageId();
    const format = this.stageReseedFormat();

    if (!this.isOrganizer() || stageId === null) return;

    this.showStageReseedConfirm.set(false);
    this.reseedingStage.set(true);
    this.bracketError.set('');

    this.bracketService.reseedStage(t.id, stageId, format).subscribe({
      next: () => {
        this.reseedingStage.set(false);
        this.stageReseedStageId.set(null);
        this.stageReseedFormat.set('');
        this.multiStageRefreshKey.update(k => k + 1);
      },
      error: (err) => {
        this.bracketError.set(err.error?.error || 'Failed to reseed stage');
        this.reseedingStage.set(false);
      }
    });
  }
}
