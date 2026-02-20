import { Component, signal, computed, inject } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { Breadcrumb, BreadcrumbItem } from '../../components/breadcrumb/breadcrumb';
import { TournamentService } from '../../services/tournament.service';
import { CreateTournamentRequest } from '../../models/tournament.model';

@Component({
  selector: 'app-tournament-new',
  imports: [Breadcrumb, FormsModule],
  templateUrl: './tournament-new.html',
  styleUrl: './tournament-new.css'
})
export class TournamentNew {
  private router = inject(Router);
  private tournamentService = inject(TournamentService);

  breadcrumbs: BreadcrumbItem[] = [
    { label: 'Tournaments', route: '/tournaments' },
    { label: 'New Tournament' }
  ];

  // Form fields
  name = signal('');
  game = signal('');
  description = signal('');
  format = signal<'single_elimination' | 'double_elimination'>('single_elimination');
  maxParticipants = signal<number | null>(null);
  startsAt = signal('');
  startsAtTentative = signal(false);

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

  onSubmit() {
    this.nameTouched.set(true);

    if (!this.isValid()) {
      return;
    }

    this.loading.set(true);
    this.error.set('');

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
    if (this.startsAt()) {
      request.starts_at = new Date(this.startsAt()).toISOString();
      request.starts_at_tentative = this.startsAtTentative();
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
}
