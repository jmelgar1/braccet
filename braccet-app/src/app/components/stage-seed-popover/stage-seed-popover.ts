import { Component, inject, input, output, signal, OnDestroy, effect } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { TournamentService } from '../../services/tournament.service';
import { TournamentStage, Participant, StagePoolEntry, StagePoolConfigInput } from '../../models/tournament.model';
import { Subject } from 'rxjs';
import { takeUntil } from 'rxjs/operators';

@Component({
  selector: 'app-stage-seed-popover',
  imports: [FormsModule],
  templateUrl: './stage-seed-popover.html',
  styleUrl: './stage-seed-popover.css'
})
export class StageSeedPopover implements OnDestroy {
  private tournamentService = inject(TournamentService);
  private destroy$ = new Subject<void>();
  private initialized = false;

  // Inputs
  stage = input.required<TournamentStage>();
  allStages = input.required<TournamentStage[]>();
  participants = input.required<Participant[]>();
  tournamentSlug = input.required<string>();
  initialPool = input<StagePoolEntry[]>([]);

  // Outputs
  close = output<void>();
  poolUpdated = output<StagePoolEntry[]>();

  // State - track selected participant IDs for THIS stage
  selectedParticipantIds = signal<Set<number>>(new Set());
  saving = signal(false);
  error = signal('');
  searchQuery = signal('');

  constructor() {
    // Use effect to initialize from pool when inputs are ready
    effect(() => {
      const pool = this.initialPool();
      const stage = this.stage();

      // Only initialize once and when we have data
      if (!this.initialized && stage) {
        this.initialized = true;
        const poolForThisStage = pool.filter(e => e.stage_id === stage.id);
        if (poolForThisStage.length > 0) {
          this.selectedParticipantIds.set(new Set(poolForThisStage.map(e => e.participant_id)));
        }
      }
    });
  }

  // Get the first group stage ID (participants here are "default" assignments, not explicit)
  private getFirstGroupStageId(): number | null {
    const stages = this.allStages();
    const groupStages = stages
      .filter(s => s.stage_type === 'group' && s.stage_order > 0)
      .sort((a, b) => a.stage_order - b.stage_order);
    return groupStages.length > 0 ? groupStages[0].id : null;
  }

  // Get participant IDs explicitly assigned to OTHER later stages (not this one, not the first group stage)
  // The first group stage contains "default" participants who weren't assigned elsewhere
  private getParticipantIdsInOtherLaterStages(): Set<number> {
    const currentStageId = this.stage().id;
    const firstGroupStageId = this.getFirstGroupStageId();
    const pool = this.initialPool();

    return new Set(
      pool
        .filter(e => e.stage_id !== currentStageId && e.stage_id !== firstGroupStageId)
        .map(e => e.participant_id)
    );
  }

  // Get participant by ID
  getParticipant(participantId: number): Participant | undefined {
    return this.participants().find(p => p.id === participantId);
  }

  // Get available participants: not selected for THIS stage AND not explicitly assigned to OTHER later stages
  getAvailableParticipants(): Participant[] {
    const selectedIds = this.selectedParticipantIds();
    const idsInOtherLaterStages = this.getParticipantIdsInOtherLaterStages();
    const query = this.searchQuery().toLowerCase();

    return this.participants()
      .filter(p => !selectedIds.has(p.id))
      .filter(p => !idsInOtherLaterStages.has(p.id))
      .filter(p => !query || p.display_name.toLowerCase().includes(query));
  }

  // Get selected participants in display order
  getSelectedParticipants(): Participant[] {
    const selectedIds = this.selectedParticipantIds();
    return this.participants().filter(p => selectedIds.has(p.id));
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }

  onSearchInput(event: Event): void {
    const value = (event.target as HTMLInputElement).value;
    this.searchQuery.set(value);
  }

  addParticipant(participantId: number): void {
    this.selectedParticipantIds.update(ids => {
      const newIds = new Set(ids);
      newIds.add(participantId);
      return newIds;
    });
  }

  removeParticipant(participantId: number): void {
    this.selectedParticipantIds.update(ids => {
      const newIds = new Set(ids);
      newIds.delete(participantId);
      return newIds;
    });
  }

  // Select all available participants (those not explicitly assigned to other later stages)
  selectAll(): void {
    const idsInOtherLaterStages = this.getParticipantIdsInOtherLaterStages();
    const availableIds = new Set(
      this.participants()
        .filter(p => !idsInOtherLaterStages.has(p.id))
        .map(p => p.id)
    );
    this.selectedParticipantIds.set(availableIds);
  }

  // Clear all selections for this stage
  clearAll(): void {
    this.selectedParticipantIds.set(new Set());
  }

  save(): void {
    this.saving.set(true);
    this.error.set('');

    // Build complete config for ALL stages (except first group stage which gets remainder)
    const currentStageId = this.stage().id;
    const firstGroupStageId = this.getFirstGroupStageId();
    const pool = this.initialPool();

    // Collect participant IDs per stage from existing pool (for OTHER later stages, not first group stage)
    // First group stage is handled as "remainder" by the backend, so we don't include it
    const stageParticipantsMap = new Map<number, number[]>();
    for (const entry of pool) {
      if (entry.stage_id !== currentStageId && entry.stage_id !== firstGroupStageId) {
        const ids = stageParticipantsMap.get(entry.stage_id) || [];
        ids.push(entry.participant_id);
        stageParticipantsMap.set(entry.stage_id, ids);
      }
    }

    // Add this stage's selected participant IDs (unless it's the first group stage - those are auto-assigned)
    if (currentStageId !== firstGroupStageId) {
      stageParticipantsMap.set(currentStageId, Array.from(this.selectedParticipantIds()));
    }

    // Build config array with explicit participant IDs
    const config: StagePoolConfigInput[] = [];
    for (const [stageId, participantIds] of stageParticipantsMap) {
      if (participantIds.length > 0) {
        config.push({
          stage_id: stageId,
          participant_ids: participantIds
        });
      }
    }

    this.tournamentService.updateStagePool(
      this.tournamentSlug(),
      { config }
    ).pipe(takeUntil(this.destroy$)).subscribe({
      next: (result) => {
        this.saving.set(false);
        this.poolUpdated.emit(result.entries);
      },
      error: (err) => {
        this.saving.set(false);
        this.error.set(err.error?.error || 'Failed to save stage assignments');
      }
    });
  }
}
