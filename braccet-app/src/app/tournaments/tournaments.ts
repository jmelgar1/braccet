import { Component, inject, signal, OnInit } from '@angular/core';
import { DatePipe } from '@angular/common';
import { Router } from '@angular/router';
import { TournamentService } from '../services/tournament.service';
import { AuthService } from '../services/auth.service';
import { Tournament } from '../models/tournament.model';

@Component({
  selector: 'app-tournaments',
  imports: [DatePipe],
  templateUrl: './tournaments.html',
  styleUrl: './tournaments.css'
})
export class Tournaments implements OnInit {
  private tournamentService = inject(TournamentService);
  private authService = inject(AuthService);
  private router = inject(Router);

  tournaments = signal<Tournament[]>([]);
  loading = signal(true);
  error = signal('');

  ngOnInit(): void {
    this.loadTournaments();
  }

  loadTournaments(): void {
    // If not logged in, show empty state immediately
    if (!this.authService.isLoggedIn()) {
      this.tournaments.set([]);
      this.loading.set(false);
      return;
    }

    this.loading.set(true);
    this.error.set('');

    this.tournamentService.getTournaments().subscribe({
      next: (tournaments) => {
        this.tournaments.set(tournaments || []);
        this.loading.set(false);
      },
      error: (err) => {
        // For auth errors (401/403), just show empty state instead of error
        if (err.status === 401 || err.status === 403) {
          this.tournaments.set([]);
          this.loading.set(false);
          return;
        }
        this.error.set(err.error?.error || 'Failed to load tournaments');
        this.loading.set(false);
      }
    });
  }

  onCreateTournament(): void {
    this.router.navigate(['/tournaments/new']);
  }

  onTournamentClick(slug: string): void {
    this.router.navigate(['/tournaments', slug]);
  }
}
