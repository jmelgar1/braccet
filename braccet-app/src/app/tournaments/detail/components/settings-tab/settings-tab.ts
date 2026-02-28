import { Component, output, inject, signal, effect, computed } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { DecimalPipe } from '@angular/common';
import { Router } from '@angular/router';
import { Tournament, UpdateTournamentRequest, TournamentStage, UpdateStageRequest, StageConfigRequest, RankingCriterion, StageFormat, TournamentClass, PlacementTier, PrizeDistributionMode } from '../../../../models/tournament.model';
import { TournamentService } from '../../../../services/tournament.service';
import { TournamentUIService } from '../../../../services/tournament-ui.service';

@Component({
  selector: 'app-settings-tab',
  imports: [FormsModule, DecimalPipe],
  templateUrl: './settings-tab.html'
})
export class SettingsTab {
  private tournamentService = inject(TournamentService);
  private tournamentUI = inject(TournamentUIService);
  private router = inject(Router);

  // Read from service
  tournament = computed(() => this.tournamentUI.tournament()!);
  stages = computed(() => this.tournamentUI.stages());

  tournamentUpdated = output<Tournament>();
  stageUpdated = output<TournamentStage>();

  // Form fields
  name = signal('');
  game = signal('');
  description = signal('');
  maxParticipants = signal<number | null>(null);
  startsAt = signal('');
  startsAtTentative = signal(false);
  registrationOpen = signal(false);
  tournamentClass = signal<TournamentClass | null>(null);
  prizePoolUsd = signal<number | null>(null);

  // Prize distribution state
  prizeDistributionMode = signal<PrizeDistributionMode>('percentage');
  prizeTiers = signal<PlacementTier[]>([]);
  showPrizeDistribution = signal(false);
  loadingTiers = signal(false);
  // Track if user manually opened prize distribution (to prevent effect from resetting)
  private userOpenedPrizeDistribution = false;

  // Stage editing state
  editingStageId = signal<number | null>(null);
  stageFormat = signal<StageFormat>('single_elimination');
  stageParticipantsPerGroup = signal<number>(4);
  stageAdvancingPerGroup = signal<number>(2);
  stageExpectedParticipants = signal<number | null>(null);
  stageSwissRounds = signal<number | null>(null);
  stageWinsToAdvance = signal<number | null>(null);
  stageLossesToEliminate = signal<number | null>(null);
  stageSkipFinals = signal(false);
  stageRankingCriteria = signal<RankingCriterion[]>([]);
  savingStage = signal(false);
  stageError = signal('');
  stageSuccess = signal('');

  // Add new stage state
  addingStage = signal(false);
  newStageFormat = signal<StageFormat>('swiss');
  newStageParticipantsPerGroup = signal<number>(4);
  newStageAdvancingPerGroup = signal<number>(2);
  newStageRankingCriteria = signal<RankingCriterion[]>(['match_wins', 'set_differential']);
  creatingStage = signal(false);
  deletingStageId = signal<number | null>(null);

  saving = signal(false);
  deleting = signal(false);
  error = signal('');
  success = signal('');

  // Computed
  isLocked = computed(() => this.tournament().status !== 'registration');
  isMultiStage = computed(() => this.tournament().format === 'multi_stage');
  hasCommunity = computed(() => !!this.tournament().community_id);

  // Prize distribution computed
  totalPrize = computed(() => this.prizePoolUsd() || 0);
  prizeDistributionSum = computed(() =>
    this.prizeTiers().reduce((acc, t) => acc + t.value, 0)
  );
  prizeDistributionValid = computed(() => {
    const tiers = this.prizeTiers();
    if (tiers.length === 0) return true;

    const sum = this.prizeDistributionSum();
    if (this.prizeDistributionMode() === 'percentage') {
      return Math.abs(sum - 100) < 0.01;
    } else {
      return sum <= this.totalPrize();
    }
  });

  // For bracket formats (single/double elim), rankings are determined by bracket placement
  stageUseBracketRanking = computed(() => {
    const format = this.stageFormat();
    return format === 'single_elimination' || format === 'double_elimination';
  });

  // Count group stages for add stage button visibility
  groupStageCount = computed(() => {
    const stageList = this.stages();
    if (!stageList) return 0;
    return stageList.filter(s => s.stage_type === 'group').length;
  });

  // For new stage form - check if using bracket ranking
  newStageUseBracketRanking = computed(() => {
    const format = this.newStageFormat();
    return format === 'single_elimination' || format === 'double_elimination';
  });

  availableFormats: { value: StageFormat; label: string }[] = [
    { value: 'single_elimination', label: 'Single Elimination' },
    { value: 'double_elimination', label: 'Double Elimination' },
    { value: 'swiss', label: 'Swiss' }
  ];

  availableCriteria: { value: RankingCriterion; label: string }[] = [
    { value: 'match_wins', label: 'Match Wins' },
    { value: 'set_wins', label: 'Set Wins' },
    { value: 'set_win_pct', label: 'Set Win %' },
    { value: 'set_differential', label: 'Set Differential' },
    { value: 'points_scored', label: 'Points Scored' },
    { value: 'points_differential', label: 'Points Differential' },
    { value: 'seed', label: 'Seed' }
  ];

  // Settings sub-tabs
  activeSettingsTab = signal<'general' | 'stages'>('general');

  constructor() {
    // Initialize form with tournament data
    effect(() => {
      const t = this.tournament();
      this.name.set(t.name);
      this.game.set(t.game || '');
      this.description.set(t.description || '');
      this.maxParticipants.set(t.max_participants || null);
      this.startsAtTentative.set(t.starts_at_tentative);
      this.registrationOpen.set(t.registration_open);
      this.tournamentClass.set(t.tournament_class || null);
      this.prizePoolUsd.set(t.prize_pool_usd || null);

      // Initialize prize distribution from tournament
      // Don't reset if user manually opened the configuration
      if (t.prize_distribution) {
        this.showPrizeDistribution.set(true);
        this.prizeDistributionMode.set(t.prize_distribution.mode);
        this.prizeTiers.set(t.prize_distribution.tiers);
        this.userOpenedPrizeDistribution = false; // Reset flag when loading saved distribution
      } else if (!this.userOpenedPrizeDistribution) {
        // Only reset if user hasn't manually opened the configuration
        this.showPrizeDistribution.set(false);
        this.prizeTiers.set([]);
      }

      // Convert ISO date to datetime-local format
      if (t.starts_at) {
        const date = new Date(t.starts_at);
        this.startsAt.set(this.toLocalDateTimeString(date));
      } else {
        this.startsAt.set('');
      }
    });
  }

  private toLocalDateTimeString(date: Date): string {
    const pad = (n: number) => n.toString().padStart(2, '0');
    return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
  }

  onSubmit(): void {
    const t = this.tournament();
    this.saving.set(true);
    this.error.set('');
    this.success.set('');

    const request: UpdateTournamentRequest = {
      name: this.name(),
      game: this.game() || undefined,
      description: this.description() || undefined,
      max_participants: this.maxParticipants() || undefined,
      starts_at: this.startsAt() ? new Date(this.startsAt()).toISOString() : undefined,
      starts_at_tentative: this.startsAtTentative(),
      registration_open: this.registrationOpen(),
      tournament_class: this.tournamentClass() || undefined,
      prize_pool_usd: this.prizePoolUsd() || undefined,
      prize_distribution: this.showPrizeDistribution() && this.prizeTiers().length > 0 ? {
        mode: this.prizeDistributionMode(),
        tiers: this.prizeTiers()
      } : undefined
    };

    this.tournamentService.updateTournament(t.slug, request).subscribe({
      next: (updated) => {
        this.tournamentUpdated.emit(updated);
        this.saving.set(false);
        this.success.set('Tournament updated successfully');
        setTimeout(() => this.success.set(''), 3000);
      },
      error: (err) => {
        this.error.set(err.error?.error || 'Failed to update tournament');
        this.saving.set(false);
      }
    });
  }

  deleteTournament(): void {
    const t = this.tournament();

    if (!confirm(`Are you sure you want to delete "${t.name}"? This cannot be undone.`)) {
      return;
    }

    this.deleting.set(true);
    this.error.set('');

    this.tournamentService.deleteTournament(t.slug).subscribe({
      next: () => {
        this.router.navigate(['/tournaments']);
      },
      error: (err) => {
        this.error.set(err.error?.error || 'Failed to delete tournament');
        this.deleting.set(false);
      }
    });
  }

  // Stage editing methods

  getSortedStages(): TournamentStage[] {
    const stageList = this.stages();
    if (!stageList) return [];
    // Group stages (order > 0) first, then final stage (order = 0)
    return [...stageList].sort((a, b) => {
      if (a.stage_order === 0) return 1;
      if (b.stage_order === 0) return -1;
      return a.stage_order - b.stage_order;
    });
  }

  getStageLabel(stage: TournamentStage): string {
    if (stage.stage_type === 'final') {
      return 'Final Stage';
    }
    return `Group Stage ${stage.stage_order}`;
  }

  getFormatLabel(format: StageFormat): string {
    const found = this.availableFormats.find(f => f.value === format);
    return found?.label || format;
  }

  editStage(stage: TournamentStage): void {
    this.editingStageId.set(stage.id);
    this.stageFormat.set(stage.format);
    this.stageParticipantsPerGroup.set(stage.participants_per_group || 4);
    this.stageAdvancingPerGroup.set(stage.advancing_per_group || 2);
    this.stageExpectedParticipants.set(stage.expected_participants ?? null);
    this.stageSwissRounds.set(stage.swiss_rounds || null);
    this.stageWinsToAdvance.set(stage.wins_to_advance ?? null);
    this.stageLossesToEliminate.set(stage.losses_to_eliminate ?? null);
    this.stageSkipFinals.set(stage.skip_finals || false);
    this.stageRankingCriteria.set(stage.ranking_criteria || []);
    this.stageError.set('');
    this.stageSuccess.set('');
  }

  cancelStageEdit(): void {
    this.editingStageId.set(null);
    this.stageError.set('');
    this.stageSuccess.set('');
  }

  onStageFormatChange(format: StageFormat): void {
    this.stageFormat.set(format);

    // Update ranking criteria defaults based on format
    if (format === 'single_elimination' || format === 'double_elimination') {
      // For bracket formats, default to empty (bracket determines ranking)
      this.stageRankingCriteria.set([]);
    } else if (format === 'swiss') {
      // For Swiss, set sensible defaults if empty
      if (this.stageRankingCriteria().length === 0) {
        this.stageRankingCriteria.set(['match_wins', 'set_differential']);
      }
    }
  }

  toggleCriterion(criterion: RankingCriterion): void {
    const current = this.stageRankingCriteria();
    if (current.includes(criterion)) {
      this.stageRankingCriteria.set(current.filter(c => c !== criterion));
    } else {
      this.stageRankingCriteria.set([...current, criterion]);
    }
  }

  moveCriterionUp(index: number): void {
    if (index === 0) return;
    const current = [...this.stageRankingCriteria()];
    [current[index - 1], current[index]] = [current[index], current[index - 1]];
    this.stageRankingCriteria.set(current);
  }

  moveCriterionDown(index: number): void {
    const current = [...this.stageRankingCriteria()];
    if (index >= current.length - 1) return;
    [current[index], current[index + 1]] = [current[index + 1], current[index]];
    this.stageRankingCriteria.set(current);
  }

  saveStage(): void {
    const stageId = this.editingStageId();
    if (!stageId) return;

    const stage = this.stages().find(s => s.id === stageId);
    if (!stage) return;

    this.savingStage.set(true);
    this.stageError.set('');
    this.stageSuccess.set('');

    const request: UpdateStageRequest = {
      format: this.stageFormat(),
      skip_finals: this.stageSkipFinals(),
      ranking_criteria: this.stageRankingCriteria()
    };

    // Only include group-specific fields for group stages
    if (stage.stage_type === 'group') {
      request.participants_per_group = this.stageParticipantsPerGroup();
      request.advancing_per_group = this.stageAdvancingPerGroup();
    }

    // Include expected participants if set (for prize tier calculation)
    if (this.stageExpectedParticipants() !== null) {
      request.expected_participants = this.stageExpectedParticipants()!;
    }

    // Only include Swiss-specific fields for Swiss format
    if (this.stageFormat() === 'swiss') {
      if (this.stageSwissRounds()) {
        request.swiss_rounds = this.stageSwissRounds()!;
      }
      if (this.stageWinsToAdvance() !== null) {
        request.wins_to_advance = this.stageWinsToAdvance()!;
      }
      if (this.stageLossesToEliminate() !== null) {
        request.losses_to_eliminate = this.stageLossesToEliminate()!;
      }
    }

    this.tournamentService.updateStage(this.tournament().slug, stageId, request).subscribe({
      next: (updated) => {
        this.stageUpdated.emit(updated);
        this.savingStage.set(false);
        this.stageSuccess.set('Stage updated successfully');
        this.editingStageId.set(null);
        setTimeout(() => this.stageSuccess.set(''), 3000);
      },
      error: (err) => {
        this.stageError.set(err.error?.error || 'Failed to update stage');
        this.savingStage.set(false);
      }
    });
  }

  // Add/delete stage methods

  showAddStageForm(): void {
    this.addingStage.set(true);
    this.newStageFormat.set('swiss');
    this.newStageParticipantsPerGroup.set(4);
    this.newStageAdvancingPerGroup.set(2);
    this.newStageRankingCriteria.set(['match_wins', 'set_differential']);
    this.stageError.set('');
  }

  cancelAddStage(): void {
    this.addingStage.set(false);
    this.stageError.set('');
  }

  onNewStageFormatChange(format: StageFormat): void {
    this.newStageFormat.set(format);
    if (format === 'single_elimination' || format === 'double_elimination') {
      this.newStageRankingCriteria.set([]);
    } else if (format === 'swiss' && this.newStageRankingCriteria().length === 0) {
      this.newStageRankingCriteria.set(['match_wins', 'set_differential']);
    }
  }

  toggleNewStageCriterion(criterion: RankingCriterion): void {
    const current = this.newStageRankingCriteria();
    if (current.includes(criterion)) {
      this.newStageRankingCriteria.set(current.filter(c => c !== criterion));
    } else {
      this.newStageRankingCriteria.set([...current, criterion]);
    }
  }

  moveNewStageCriterionUp(index: number): void {
    if (index === 0) return;
    const current = [...this.newStageRankingCriteria()];
    [current[index - 1], current[index]] = [current[index], current[index - 1]];
    this.newStageRankingCriteria.set(current);
  }

  moveNewStageCriterionDown(index: number): void {
    const current = [...this.newStageRankingCriteria()];
    if (index >= current.length - 1) return;
    [current[index], current[index + 1]] = [current[index + 1], current[index]];
    this.newStageRankingCriteria.set(current);
  }

  createStage(): void {
    this.creatingStage.set(true);
    this.stageError.set('');

    const request: StageConfigRequest = {
      stage_order: 0, // Backend will set the correct order
      format: this.newStageFormat(),
      participants_per_group: this.newStageParticipantsPerGroup(),
      advancing_per_group: this.newStageAdvancingPerGroup(),
      ranking_criteria: this.newStageRankingCriteria()
    };

    this.tournamentService.addStage(this.tournament().slug, request).subscribe({
      next: (created) => {
        this.stageUpdated.emit(created);
        this.creatingStage.set(false);
        this.addingStage.set(false);
        this.stageSuccess.set('Stage added successfully');
        setTimeout(() => this.stageSuccess.set(''), 3000);
      },
      error: (err) => {
        this.stageError.set(err.error?.error || 'Failed to add stage');
        this.creatingStage.set(false);
      }
    });
  }

  deleteStage(stage: TournamentStage): void {
    if (!confirm(`Are you sure you want to delete "${this.getStageLabel(stage)}"? This cannot be undone.`)) {
      return;
    }

    this.deletingStageId.set(stage.id);
    this.stageError.set('');

    this.tournamentService.deleteStage(this.tournament().slug, stage.id).subscribe({
      next: () => {
        this.deletingStageId.set(null);
        this.stageSuccess.set('Stage deleted successfully');
        // Emit a fake updated stage to trigger parent refresh
        this.stageUpdated.emit({ ...stage, id: -1 } as TournamentStage);
        setTimeout(() => this.stageSuccess.set(''), 3000);
      },
      error: (err) => {
        this.stageError.set(err.error?.error || 'Failed to delete stage');
        this.deletingStageId.set(null);
      }
    });
  }

  // Prize distribution methods

  generateSuggestedTiers(): void {
    this.userOpenedPrizeDistribution = true; // Mark that user manually opened this
    this.loadingTiers.set(true);
    const tournament = this.tournament();
    // Use participant_count (actual) or max_participants, let backend figure it out if neither
    const participants = tournament.participant_count || tournament.max_participants || undefined;

    this.tournamentService.getSuggestedPrizeTiers(tournament.slug, participants)
      .subscribe({
        next: (response) => {
          // Handle null/empty tiers from API
          const responseTiers = response.tiers || [];
          if (responseTiers.length === 0) {
            this.prizeTiers.set([]);
            this.loadingTiers.set(false);
            this.error.set('No prize tiers available for the current participant count');
            return;
          }
          // Initialize with default percentage values (halving pattern: 50, 25, 12.5...)
          const tiers = responseTiers.map((t, i) => ({
            ...t,
            value: this.getDefaultPercentage(i, responseTiers.length)
          }));
          this.prizeTiers.set(tiers);
          this.loadingTiers.set(false);
        },
        error: (err) => {
          console.error('Failed to load prize tiers:', err);
          this.loadingTiers.set(false);
          this.error.set('Failed to load prize tiers');
        }
      });
  }

  private getDefaultPercentage(index: number, total: number): number {
    // Standard prize distribution: halving pattern
    const defaultDistribution = [50, 25, 12.5, 6.25, 3.125, 1.5625, 0.78125];
    if (index < defaultDistribution.length) {
      return defaultDistribution[index];
    }
    return 0;
  }

  updateTierValue(index: number, value: number): void {
    const tiers = [...this.prizeTiers()];
    tiers[index] = { ...tiers[index], value };
    this.prizeTiers.set(tiers);
  }

  removeTier(index: number): void {
    this.prizeTiers.set(this.prizeTiers().filter((_, i) => i !== index));
  }

  clearPrizeDistribution(): void {
    this.prizeTiers.set([]);
    this.showPrizeDistribution.set(false);
    this.userOpenedPrizeDistribution = false;
  }

  computeTierAmount(tier: PlacementTier): number {
    if (this.prizeDistributionMode() === 'percentage') {
      return (this.totalPrize() * tier.value) / 100;
    }
    return tier.value;
  }
}
