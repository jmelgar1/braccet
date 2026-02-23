import { Component, input, output, signal, inject, effect } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { BracketStage, UpdateStageRequest } from '../../models/bracket.model';
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
      stage_name: this.stageName(),
      best_of: this.bestOf()
    };

    this.bracketService.updateStage(this.tournamentId(), this.stage().round, request)
      .subscribe({
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
