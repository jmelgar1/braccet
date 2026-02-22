import { Component, input, output, inject, signal, effect } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { Tournament, UpdateTournamentRequest } from '../../../../models/tournament.model';
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
  tournamentUpdated = output<Tournament>();

  // Form fields
  name = signal('');
  game = signal('');
  description = signal('');
  maxParticipants = signal<number | null>(null);
  startsAt = signal('');
  startsAtTentative = signal(false);
  registrationOpen = signal(false);

  saving = signal(false);
  deleting = signal(false);
  error = signal('');
  success = signal('');

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
}
