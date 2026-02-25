import { Component, input, output, signal, inject, effect } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { BracketStage, UpdateStageRequest, BracketType } from '../../models/bracket.model';
import { BracketService } from '../../services/bracket.service';

@Component({
  selector: 'app-edit-stage-modal',
  imports: [FormsModule],
  templateUrl: './edit-stage-modal.html',
  styleUrl: './edit-stage-modal.css'
})
export class EditStageModal {
  private bracketService = inject(BracketService);

  tournamentId = input.required<number>();
  stage = input.required<BracketStage>();
  stageId = input<number | null>(null); // For multi-stage group stages
  groupId = input<number | null>(null); // For multi-stage group stages
  hideNameField = input(false); // Hide stage name field (for Swiss brackets)

  close = output<void>();
  stageUpdated = output<BracketStage>();

  stageName = signal('');
  bestOf = signal(1);
  saving = signal(false);
  error = signal('');

  bestOfOptions = [1, 3, 5, 7, 9];

  constructor() {
    effect(() => {
      const s = this.stage();
      this.stageName.set(s.stage_name);
      this.bestOf.set(s.best_of);
    });
  }

  canSave(): boolean {
    return !this.saving();
  }

  save(): void {
    this.saving.set(true);
    this.error.set('');

    const request: UpdateStageRequest = {
      best_of: this.bestOf(),
      bracket_type: this.stage().bracket_type as BracketType
    };

    // Only include stage_name if we're not hiding the field
    if (!this.hideNameField()) {
      request.stage_name = this.stageName();
    }

    const sId = this.stageId();
    const gId = this.groupId();

    // Use group-specific API if stageId and groupId are provided
    const updateCall = sId !== null && gId !== null
      ? this.bracketService.updateGroupStage(this.tournamentId(), sId, gId, this.stage().round, request)
      : this.bracketService.updateStage(this.tournamentId(), this.stage().round, request);

    updateCall.subscribe({
      next: (updated) => {
        this.saving.set(false);
        this.stageUpdated.emit(updated);
        this.close.emit();
      },
      error: (err) => {
        this.saving.set(false);
        this.error.set(err.error?.error || 'Failed to update stage');
      }
    });
  }

  onBackdropClick(event: MouseEvent): void {
    if ((event.target as HTMLElement).classList.contains('modal-backdrop')) {
      this.close.emit();
    }
  }
}
