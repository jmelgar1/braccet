import { Component, signal, computed, inject, OnInit, effect } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { Breadcrumb, BreadcrumbItem } from '../../components/breadcrumb/breadcrumb';
import { TournamentService } from '../../services/tournament.service';
import { CommunityService } from '../../services/community.service';
import { EloService } from '../../services/elo.service';
import { CreateTournamentRequest, CreateMultiStageTournamentRequest, StageConfigRequest, RankingCriterion, StageFormat, TournamentFormat } from '../../models/tournament.model';
import { Community } from '../../models/community.model';
import { EloSystem } from '../../models/elo.model';

@Component({
  selector: 'app-tournament-new',
  imports: [Breadcrumb, FormsModule],
  templateUrl: './tournament-new.html',
  styleUrl: './tournament-new.css'
})
export class TournamentNew implements OnInit {
  private router = inject(Router);
  private tournamentService = inject(TournamentService);
  private communityService = inject(CommunityService);
  private eloService = inject(EloService);

  breadcrumbs: BreadcrumbItem[] = [
    { label: 'Tournaments', route: '/tournaments' },
    { label: 'New Tournament' }
  ];

  // Communities
  communities = signal<Community[]>([]);
  loadingCommunities = signal(false);

  // ELO Systems (for selected community)
  eloSystems = signal<EloSystem[]>([]);
  loadingEloSystems = signal(false);

  // Form fields
  name = signal('');
  game = signal('');
  description = signal('');
  format = signal<StageFormat>('single_elimination');
  multiStageEnabled = signal(false);
  maxParticipants = signal<number | null>(null);
  startsAt = signal('');
  startsAtTentative = signal(false);
  communityId = signal<number | null>(null);
  eloSystemId = signal<number | null>(null);

  // Swiss format settings
  swissRounds = signal<number>(3);
  showSwissModal = signal(false);
  previousFormat = signal<StageFormat>('single_elimination');

  // Multi-stage tournament settings
  showMultiStageModal = signal(false);
  showCancelConfirmation = signal(false);
  groupStages = signal<StageConfigRequest[]>([]);
  finalStage = signal<StageConfigRequest | null>(null);
  // Backup for cancel/restore
  private groupStagesBackup: StageConfigRequest[] = [];
  private finalStageBackup: StageConfigRequest | null = null;

  // Current stage being edited in multi-stage wizard
  // -1 means final stage is selected, 0+ means group stage index
  currentStageIndex = signal(0);
  editingFinalStage = signal(false);
  stageParticipantsPerGroup = signal(4);
  stageAdvancingPerGroup = signal(2);
  stageFormat = signal<StageFormat>('swiss');
  stageSwissRounds = signal(3);
  stageRankingCriteria = signal<RankingCriterion[]>([]);
  stagePlacementMatches = signal(false);
  stagePlacementDepth = signal(1);
  stageSkipFinals = signal(false);

  // For bracket formats (single/double elim), rankings are determined by bracket placement
  // Ranking criteria only applies to Swiss format or as an optional override for elim formats
  stageUseBracketRanking = computed(() => {
    const format = this.stageFormat();
    return format === 'single_elimination' || format === 'double_elimination';
  });

  availableRankingCriteria: { value: RankingCriterion; label: string }[] = [
    { value: 'seed', label: 'Original Seed' },
    { value: 'match_wins', label: 'Match Wins' },
    { value: 'set_wins', label: 'Set Wins' },
    { value: 'set_win_pct', label: 'Set Win %' },
    { value: 'set_differential', label: 'Set Differential' },
    { value: 'points_scored', label: 'Points Scored' },
    { value: 'points_differential', label: 'Point Differential' }
  ];

  constructor() {
    // Load ELO systems when community changes
    effect(() => {
      const cid = this.communityId();
      if (cid) {
        this.loadEloSystems(cid);
      } else {
        this.eloSystems.set([]);
        this.eloSystemId.set(null);
      }
    });
  }

  // Touched state for validation
  nameTouched = signal(false);

  // Form state
  loading = signal(false);
  error = signal('');

  // Validation
  nameError = computed(() => {
    if (!this.nameTouched()) return '';
    if (!this.name().trim()) return 'Tournament name is required';
    if (this.name().length > 200) return 'Name must be 200 characters or less';
    return '';
  });

  isValid = computed(() => {
    return this.name().trim().length > 0 && this.name().length <= 200;
  });

  ngOnInit(): void {
    this.loadCommunities();
  }

  loadCommunities(): void {
    this.loadingCommunities.set(true);
    this.communityService.getCommunities().subscribe({
      next: (communities) => {
        this.communities.set(communities || []);
        this.loadingCommunities.set(false);
      },
      error: () => {
        this.communities.set([]);
        this.loadingCommunities.set(false);
      }
    });
  }

  loadEloSystems(communityId: number): void {
    const community = this.communities().find(c => c.id === communityId);
    if (!community) return;

    this.loadingEloSystems.set(true);
    this.eloSystemId.set(null);

    this.eloService.getEloSystems(community.slug).subscribe({
      next: (systems) => {
        this.eloSystems.set(systems || []);
        // Auto-select the default system if one exists
        const defaultSystem = systems?.find(s => s.is_default);
        if (defaultSystem) {
          this.eloSystemId.set(defaultSystem.id);
        }
        this.loadingEloSystems.set(false);
      },
      error: () => {
        this.eloSystems.set([]);
        this.loadingEloSystems.set(false);
      }
    });
  }

  onFormatChange(newFormat: StageFormat): void {
    if (newFormat === 'swiss') {
      this.previousFormat.set(this.format());
      this.showSwissModal.set(true);
    }
    this.format.set(newFormat);
  }

  onMultiStageToggle(enabled: boolean): void {
    if (enabled) {
      // Initialize with one group stage if none exist
      if (this.groupStages().length === 0) {
        this.addGroupStage();
      }
      // Final stage is always required
      if (!this.finalStage()) {
        this.setFinalStage();
      }
      // Create backup before opening modal
      this.groupStagesBackup = JSON.parse(JSON.stringify(this.groupStages()));
      this.finalStageBackup = this.finalStage() ? JSON.parse(JSON.stringify(this.finalStage())) : null;
      this.showMultiStageModal.set(true);
    }
    this.multiStageEnabled.set(enabled);
  }

  openMultiStageModal(): void {
    // Create backup before opening modal
    this.groupStagesBackup = JSON.parse(JSON.stringify(this.groupStages()));
    this.finalStageBackup = this.finalStage() ? JSON.parse(JSON.stringify(this.finalStage())) : null;
    this.showMultiStageModal.set(true);
  }

  confirmSwissFormat(): void {
    this.showSwissModal.set(false);
  }

  cancelSwissFormat(): void {
    this.showSwissModal.set(false);
    this.format.set(this.previousFormat());
  }

  // Multi-stage methods
  addGroupStage(): void {
    if (this.groupStages().length >= 2) return; // Max 2 group stages

    const newStage: StageConfigRequest = {
      stage_order: this.groupStages().length + 1,
      format: 'swiss',
      participants_per_group: 4,
      advancing_per_group: 2,
      // For Swiss format, default to match_wins + set_differential
      // For elim formats, empty array means use bracket placement
      ranking_criteria: ['match_wins', 'set_differential']
    };
    this.groupStages.update(stages => [...stages, newStage]);
    this.currentStageIndex.set(this.groupStages().length - 1);
    this.loadStageToForm(newStage);
  }

  removeGroupStage(index: number): void {
    this.groupStages.update(stages => {
      const newStages = stages.filter((_, i) => i !== index);
      // Renumber stages
      return newStages.map((s, i) => ({ ...s, stage_order: i + 1 }));
    });
    if (this.currentStageIndex() >= this.groupStages().length) {
      this.currentStageIndex.set(Math.max(0, this.groupStages().length - 1));
    }
    if (this.groupStages().length > 0) {
      this.loadStageToForm(this.groupStages()[this.currentStageIndex()]);
    }
  }

  selectStage(index: number): void {
    // Save current stage before switching
    if (!this.editingFinalStage()) {
      this.saveCurrentStage();
    }
    this.editingFinalStage.set(false);
    this.currentStageIndex.set(index);
    this.loadStageToForm(this.groupStages()[index]);
  }

  selectFinalStage(): void {
    // Save current group stage before switching
    if (!this.editingFinalStage()) {
      this.saveCurrentStage();
    }
    this.editingFinalStage.set(true);
  }

  private loadStageToForm(stage: StageConfigRequest): void {
    this.stageParticipantsPerGroup.set(stage.participants_per_group || 4);
    this.stageAdvancingPerGroup.set(stage.advancing_per_group || 2);
    this.stageFormat.set(stage.format);
    this.stageSwissRounds.set(stage.swiss_rounds || 3);
    this.stageSkipFinals.set(stage.skip_finals || false);
    // For elim formats, empty criteria means use bracket ranking (which is the default)
    // For swiss, if no criteria set, use sensible defaults
    const defaultCriteria = (stage.format === 'swiss' && (!stage.ranking_criteria || stage.ranking_criteria.length === 0))
      ? ['match_wins', 'set_differential'] as RankingCriterion[]
      : (stage.ranking_criteria || []);
    this.stageRankingCriteria.set(defaultCriteria);
    this.stagePlacementMatches.set(stage.placement_matches || false);
    this.stagePlacementDepth.set(stage.placement_depth || 1);
  }

  private saveCurrentStage(): void {
    const index = this.currentStageIndex();
    if (index < 0 || index >= this.groupStages().length) return;

    const updatedStage: StageConfigRequest = {
      stage_order: index + 1,
      format: this.stageFormat(),
      participants_per_group: this.stageParticipantsPerGroup(),
      advancing_per_group: this.stageAdvancingPerGroup(),
      skip_finals: this.stageSkipFinals(),
      ranking_criteria: this.stageRankingCriteria(),
      placement_matches: this.stagePlacementMatches(),
      placement_depth: this.stagePlacementDepth()
    };

    if (this.stageFormat() === 'swiss') {
      updatedStage.swiss_rounds = this.stageSwissRounds();
    }

    this.groupStages.update(stages =>
      stages.map((s, i) => i === index ? updatedStage : s)
    );
  }

  private setFinalStage(): void {
    this.finalStage.set({
      stage_order: 0,
      format: 'single_elimination'
    });
  }

  updateFinalStageFormat(format: StageFormat): void {
    this.finalStage.update(f => f ? { ...f, format } : null);
  }

  onStageFormatChange(format: StageFormat): void {
    this.stageFormat.set(format);

    // Update ranking criteria defaults based on format
    if (format === 'single_elimination' || format === 'double_elimination') {
      // For bracket formats, default to empty (bracket determines ranking)
      // but preserve user's choice if they've already set criteria
      if (this.stageRankingCriteria().length > 0) {
        // Ask user if they want to keep criteria or use bracket ranking
        // For now, just clear it since we're setting a new format
        this.stageRankingCriteria.set([]);
      }
    } else if (format === 'swiss') {
      // For Swiss, set sensible defaults if empty
      if (this.stageRankingCriteria().length === 0) {
        this.stageRankingCriteria.set(['match_wins', 'set_differential']);
      }
    }
  }

  toggleRankingCriterion(criterion: RankingCriterion): void {
    const current = this.stageRankingCriteria();
    if (current.includes(criterion)) {
      this.stageRankingCriteria.set(current.filter(c => c !== criterion));
    } else {
      this.stageRankingCriteria.set([...current, criterion]);
    }
  }

  // Drag and drop for ranking criteria
  draggedCriterionIndex: number | null = null;

  onDragStart(index: number): void {
    this.draggedCriterionIndex = index;
  }

  onDragOver(event: DragEvent, index: number): void {
    event.preventDefault();
  }

  onDrop(event: DragEvent, dropIndex: number): void {
    event.preventDefault();
    if (this.draggedCriterionIndex === null || this.draggedCriterionIndex === dropIndex) return;

    const criteria = [...this.stageRankingCriteria()];
    const [removed] = criteria.splice(this.draggedCriterionIndex, 1);
    criteria.splice(dropIndex, 0, removed);
    this.stageRankingCriteria.set(criteria);
    this.draggedCriterionIndex = null;
  }

  onDragEnd(): void {
    this.draggedCriterionIndex = null;
  }

  getCriterionLabel(criterion: RankingCriterion): string {
    const found = this.availableRankingCriteria.find(c => c.value === criterion);
    return found?.label ?? criterion;
  }

  confirmMultiStage(): void {
    if (!this.editingFinalStage()) {
      this.saveCurrentStage();
    }
    this.showMultiStageModal.set(false);
  }

  requestCancelMultiStage(): void {
    this.showCancelConfirmation.set(true);
  }

  confirmCancelMultiStage(): void {
    this.showCancelConfirmation.set(false);
    this.showMultiStageModal.set(false);
    // Restore from backup
    this.groupStages.set(this.groupStagesBackup);
    this.finalStage.set(this.finalStageBackup);
    // Always uncheck multi-stage when canceling
    this.multiStageEnabled.set(false);
    // Reset form to first stage if there are stages
    if (this.groupStages().length > 0) {
      this.currentStageIndex.set(0);
      this.loadStageToForm(this.groupStages()[0]);
    }
  }

  dismissCancelConfirmation(): void {
    this.showCancelConfirmation.set(false);
  }

  onSubmit() {
    this.nameTouched.set(true);

    if (!this.isValid()) {
      return;
    }

    this.loading.set(true);
    this.error.set('');

    // Handle multi-stage tournaments separately
    if (this.multiStageEnabled()) {
      this.submitMultiStage();
      return;
    }

    const request: CreateTournamentRequest = {
      name: this.name().trim(),
      format: this.format(),
    };

    if (this.description().trim()) {
      request.description = this.description().trim();
    }
    if (this.game().trim()) {
      request.game = this.game().trim();
    }
    if (this.maxParticipants()) {
      request.max_participants = this.maxParticipants()!;
    }
    if (this.format() === 'swiss' && this.swissRounds()) {
      request.swiss_rounds = this.swissRounds();
    }
    if (this.startsAt()) {
      request.starts_at = new Date(this.startsAt()).toISOString();
      request.starts_at_tentative = this.startsAtTentative();
    }
    if (this.communityId()) {
      request.community_id = this.communityId()!;
    }
    if (this.eloSystemId()) {
      request.elo_system_id = this.eloSystemId()!;
    }

    this.tournamentService.createTournament(request).subscribe({
      next: (tournament) => {
        this.router.navigate(['/tournaments', tournament.slug]);
      },
      error: (err) => {
        this.loading.set(false);
        this.error.set(err.error?.message || 'Failed to create tournament');
      }
    });
  }

  private submitMultiStage(): void {
    // Make sure current stage is saved
    this.saveCurrentStage();

    const request: CreateMultiStageTournamentRequest = {
      name: this.name().trim(),
      group_stages: this.groupStages()
    };

    if (this.description().trim()) {
      request.description = this.description().trim();
    }
    if (this.game().trim()) {
      request.game = this.game().trim();
    }
    if (this.maxParticipants()) {
      request.max_participants = this.maxParticipants()!;
    }
    if (this.startsAt()) {
      request.starts_at = new Date(this.startsAt()).toISOString();
      request.starts_at_tentative = this.startsAtTentative();
    }
    if (this.communityId()) {
      request.community_id = this.communityId()!;
    }
    if (this.eloSystemId()) {
      request.elo_system_id = this.eloSystemId()!;
    }
    if (this.finalStage()) {
      request.final_stage = this.finalStage()!;
    }

    this.tournamentService.createMultiStageTournament(request).subscribe({
      next: (tournament) => {
        this.router.navigate(['/tournaments', tournament.slug]);
      },
      error: (err) => {
        this.loading.set(false);
        this.error.set(err.error?.message || 'Failed to create tournament');
      }
    });
  }
}
