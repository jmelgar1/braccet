import { Component, inject, input, output, signal, computed, OnInit, OnDestroy } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { TournamentService } from '../../services/tournament.service';
import { TournamentStage, Participant, StageSeedAssignment, StageSeedInput } from '../../models/tournament.model';
import { Subject } from 'rxjs';
import { takeUntil } from 'rxjs/operators';

@Component({
  selector: 'app-stage-seed-popover',
  imports: [FormsModule],
  templateUrl: './stage-seed-popover.html',
  styleUrl: './stage-seed-popover.css'
})
export class StageSeedPopover implements OnInit, OnDestroy {
  private tournamentService = inject(TournamentService);
  private destroy$ = new Subject<void>();

  // Inputs
  stage = input.required<TournamentStage>();
  participants = input.required<Participant[]>();
  tournamentSlug = input.required<string>();
  initialSeeds = input<StageSeedAssignment[]>([]);

  // Outputs
  close = output<void>();
  seedsUpdated = output<StageSeedAssignment[]>();

  // State
  currentSeeds = signal<StageSeedInput[]>([]);
  saving = signal(false);
  error = signal('');
  searchQuery = signal('');
  activeGroupDropdown = signal<number | null>(null);

  // Expected number of groups based on participants and stage config
  // For final stage, we use 1 "group" (the finals bracket itself)
  expectedGroups = computed(() => {
    const stg = this.stage();
    if (stg.stage_type === 'final') {
      return 1; // Finals has just one "group" - the bracket itself
    }
    const participantCount = this.participants().length;
    const perGroup = stg.participants_per_group ?? 4;
    return Math.ceil(participantCount / perGroup);
  });

  // Generate group names (Group A, Group B, etc.) or "Finals" for final stage
  groupNames = computed(() => {
    const stg = this.stage();
    if (stg.stage_type === 'final') {
      return ['Finals'];
    }
    return Array.from({ length: this.expectedGroups() }, (_, i) =>
      String.fromCharCode(65 + i)
    );
  });

  // Get group letter from order
  getGroupLetter(groupOrder: number): string {
    if (this.stage().stage_type === 'final') {
      return 'Finals';
    }
    return String.fromCharCode(65 + groupOrder);
  }

  // Get seeds for a specific group
  getSeedsForGroup(groupOrder: number): StageSeedInput[] {
    return this.currentSeeds().filter(s => s.target_group_order === groupOrder);
  }

  // Get participant by ID
  getParticipant(participantId: number): Participant | undefined {
    return this.participants().find(p => p.id === participantId);
  }

  // Get unseeded participants (filtered by search)
  getUnseededParticipants(): Participant[] {
    const seededIds = new Set(this.currentSeeds().map(s => s.participant_id));
    const query = this.searchQuery().toLowerCase();
    return this.participants()
      .filter(p => !seededIds.has(p.id))
      .filter(p => !query || p.display_name.toLowerCase().includes(query));
  }

  ngOnInit(): void {
    // Initialize from existing seeds
    const initial = this.initialSeeds();
    if (initial.length > 0) {
      this.currentSeeds.set(initial.map(s => ({
        participant_id: s.participant_id,
        target_group_order: s.target_group_order
      })));
    }
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }

  onSearchInput(event: Event): void {
    const value = (event.target as HTMLInputElement).value;
    this.searchQuery.set(value);
  }

  toggleGroupDropdown(groupOrder: number): void {
    if (this.activeGroupDropdown() === groupOrder) {
      this.activeGroupDropdown.set(null);
    } else {
      this.activeGroupDropdown.set(groupOrder);
      this.searchQuery.set('');
    }
  }

  addSeed(participantId: number, groupOrder: number): void {
    this.currentSeeds.update(seeds => [
      ...seeds,
      { participant_id: participantId, target_group_order: groupOrder }
    ]);
    this.activeGroupDropdown.set(null);
    this.searchQuery.set('');
  }

  removeSeed(participantId: number): void {
    this.currentSeeds.update(seeds =>
      seeds.filter(s => s.participant_id !== participantId)
    );
  }

  save(): void {
    this.saving.set(true);
    this.error.set('');

    this.tournamentService.updateStageSeeds(
      this.tournamentSlug(),
      this.stage().id,
      { seeds: this.currentSeeds() }
    ).pipe(takeUntil(this.destroy$)).subscribe({
      next: (result) => {
        this.saving.set(false);
        this.seedsUpdated.emit(result);
      },
      error: (err) => {
        this.saving.set(false);
        this.error.set(err.error?.error || 'Failed to save seeds');
      }
    });
  }
}
