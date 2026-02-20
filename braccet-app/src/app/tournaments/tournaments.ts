import { Component, inject, signal, OnInit } from '@angular/core';
import { DatePipe } from '@angular/common';
import { Router } from '@angular/router';
import { TournamentService } from '../services/tournament.service';
import { Tournament } from '../models/tournament.model';

@Component({
  selector: 'app-tournaments',
  imports: [DatePipe],
  templateUrl: './tournaments.html',
  styleUrl: './tournaments.css'
})
export class Tournaments implements OnInit {
  private tournamentService = inject(TournamentService);
  private router = inject(Router);

  tournaments = signal<Tournament[]>([]);
  loading = signal(true);
  error = signal('');

  ngOnInit(): void {
    this.loadTournaments();
  }

  loadTournaments(): void {
    this.loading.set(true);
    this.error.set('');

    this.tournamentService.getTournaments().subscribe({
      next: (tournaments) => {
        this.tournaments.set(tournaments);
        this.loading.set(false);
      },
      error: (err) => {
        this.error.set(err.error?.error || 'Failed to load tournaments');
        this.loading.set(false);
      }
    });
  }

  onCreateTournament(): void {
    this.router.navigate(['/tournaments/new']);
  }
}
