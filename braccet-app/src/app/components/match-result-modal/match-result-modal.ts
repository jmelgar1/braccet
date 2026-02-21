import { Component, input, output, signal, computed } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Match, SetScore } from '../../models/bracket.model';

export interface MatchResultEvent {
  matchId: number;
  sets: SetScore[];
}

@Component({
  selector: 'app-match-result-modal',
  imports: [FormsModule],
  templateUrl: './match-result-modal.html',
  styleUrl: './match-result-modal.css'
})
export class MatchResultModal {
  match = input.required<Match>();

  close = output<void>();
  resultSubmitted = output<MatchResultEvent>();

  // Dynamic sets array
  sets = signal<SetScore[]>([
    { set_number: 1, participant1_score: 0, participant2_score: 0 }
  ]);

  submitting = signal(false);
  error = signal('');

  isEditable = computed(() => {
    const status = this.match().status;
    return status === 'ready' || status === 'in_progress';
  });

  // Compute sets won by each participant
  setsWon = computed(() => {
    const currentSets = this.sets();
    let p1 = 0, p2 = 0;
    for (const set of currentSets) {
      if (set.participant1_score > set.participant2_score) p1++;
      else if (set.participant2_score > set.participant1_score) p2++;
    }
    return { participant1: p1, participant2: p2 };
  });

  winnerId = computed(() => {
    const match = this.match();
    const won = this.setsWon();
    if (won.participant1 > won.participant2) return match.participant1_id;
    if (won.participant2 > won.participant1) return match.participant2_id;
    return null;
  });

  canSubmit = computed(() => {
    if (!this.isEditable()) return false;
    if (this.winnerId() === null) return false;
    // Ensure no negative scores
    for (const set of this.sets()) {
      if (set.participant1_score < 0 || set.participant2_score < 0) return false;
    }
    return true;
  });

  getWinnerName(): string {
    const match = this.match();
    if (match.winner_id === match.participant1_id) {
      return match.participant1_name || 'Participant 1';
    }
    if (match.winner_id === match.participant2_id) {
      return match.participant2_name || 'Participant 2';
    }
    return 'Unknown';
  }

  addSet(): void {
    const currentSets = this.sets();
    this.sets.set([
      ...currentSets,
      {
        set_number: currentSets.length + 1,
        participant1_score: 0,
        participant2_score: 0
      }
    ]);
  }

  removeSet(index: number): void {
    const currentSets = this.sets();
    if (currentSets.length <= 1) return; // Keep at least one set

    const newSets = currentSets.filter((_, i) => i !== index);
    // Renumber sets
    newSets.forEach((set, i) => set.set_number = i + 1);
    this.sets.set(newSets);
  }

  updateSetScore(index: number, field: 'participant1_score' | 'participant2_score', value: number): void {
    const currentSets = [...this.sets()];
    currentSets[index] = { ...currentSets[index], [field]: value };
    this.sets.set(currentSets);
  }

  submit(): void {
    const match = this.match();

    if (this.winnerId() === null) {
      this.error.set('Sets are tied - there must be a clear winner');
      return;
    }

    this.submitting.set(true);
    this.error.set('');

    this.resultSubmitted.emit({
      matchId: match.id,
      sets: this.sets()
    });
  }

  setError(message: string): void {
    this.error.set(message);
    this.submitting.set(false);
  }

  onBackdropClick(event: MouseEvent): void {
    if ((event.target as HTMLElement).classList.contains('modal-backdrop')) {
      this.close.emit();
    }
  }
}
