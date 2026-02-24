import { Component, input, output, inject, signal, effect, computed } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { Tournament, UpdateTournamentRequest, TournamentStage, UpdateStageRequest, RankingCriterion, StageFormat } from '../../../../models/tournament.model';
import { TournamentService } from '../../../../services/tournament.service';

@Component({
  selector: 'app-settings-tab',
  imports: [FormsModule],
  templateUrl: './settings-tab.html'
})
export class SettingsTab {
  private tournamentService = inject(TournamentService);
  private router = inject(Router);

  tournament = input.required<Tournament>();
  stages = input<TournamentStage[]>([]);
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

  // Stage editing state
  editingStageId = signal<number | null>(null);
  stageFormat = signal<StageFormat>('single_elimination');
  stageParticipantsPerGroup = signal<number>(4);
  stageAdvancingPerGroup = signal<number>(2);
  stageSwissRounds = signal<number | null>(null);
  stageSkipFinals = signal(false);
  stageRankingCriteria = signal<RankingCriterion[]>([]);
  savingStage = signal(false);
  stageError = signal('');
  stageSuccess = signal('');

  saving = signal(false);
  deleting = signal(false);
  error = signal('');
  success = signal('');

  // Computed
  isLocked = computed(() => this.tournament().status !== 'registration');
  isMultiStage = computed(() => this.tournament().format === 'multi_stage');

  // For bracket formats (single/double elim), rankings are determined by bracket placement
  stageUseBracketRanking = computed(() => {
    const format = this.stageFormat();
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
      registration_open: this.registrationOpen()
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
    this.stageSwissRounds.set(stage.swiss_rounds || null);
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

    // Only include swiss_rounds for Swiss format
    if (this.stageFormat() === 'swiss' && this.stageSwissRounds()) {
      request.swiss_rounds = this.stageSwissRounds()!;
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
}
