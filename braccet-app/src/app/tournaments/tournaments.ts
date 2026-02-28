import { Component, inject, signal, OnInit } from '@angular/core';
import { CurrencyPipe, DatePipe, TitleCasePipe } from '@angular/common';
import { Router } from '@angular/router';
import { TournamentService } from '../services/tournament.service';
import { EventService } from '../services/event.service';
import { AuthService } from '../services/auth.service';
import { Tournament } from '../models/tournament.model';
import { Event as EventModel } from '../models/event.model';

@Component({
  selector: 'app-tournaments',
  imports: [CurrencyPipe, DatePipe, TitleCasePipe],
  templateUrl: './tournaments.html',
  styleUrl: './tournaments.css'
})
export class Tournaments implements OnInit {
  private tournamentService = inject(TournamentService);
  private eventService = inject(EventService);
  private authService = inject(AuthService);
  private router = inject(Router);

  tournaments = signal<Tournament[]>([]);
  events = signal<EventModel[]>([]);
  loading = signal(true);
  loadingEvents = signal(true);
  error = signal('');

  ngOnInit(): void {
    this.loadTournaments();
    this.loadEvents();
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
        // Sort tournaments: main events first, then qualifiers, then standalone
        // Within each group, maintain created_at DESC order
        const sorted = (tournaments || []).sort((a, b) => {
          // Priority: main > qualifier > no event
          const getPriority = (t: Tournament) => {
            if (t.event_role === 'main') return 0;
            if (t.event_role === 'qualifier') return 1;
            return 2;
          };
          const priorityDiff = getPriority(a) - getPriority(b);
          if (priorityDiff !== 0) return priorityDiff;
          // Same priority, sort by created_at DESC
          return new Date(b.created_at).getTime() - new Date(a.created_at).getTime();
        });
        this.tournaments.set(sorted);
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

  loadEvents(): void {
    if (!this.authService.isLoggedIn()) {
      this.events.set([]);
      this.loadingEvents.set(false);
      return;
    }

    this.loadingEvents.set(true);

    this.eventService.getEvents().subscribe({
      next: (events) => {
        this.events.set(events || []);
        this.loadingEvents.set(false);
      },
      error: (err) => {
        if (err.status === 401 || err.status === 403) {
          this.events.set([]);
          this.loadingEvents.set(false);
          return;
        }
        this.loadingEvents.set(false);
      }
    });
  }

  onCreateTournament(): void {
    this.router.navigate(['/tournaments/new']);
  }

  onCreateEvent(): void {
    this.router.navigate(['/events/new']);
  }

  onEventClick(slug: string): void {
    this.router.navigate(['/events', slug]);
  }

  deleteEvent(slug: string, name: string, event: MouseEvent): void {
    event.stopPropagation();

    if (!confirm(`Are you sure you want to delete "${name}"? This cannot be undone.`)) {
      return;
    }

    this.eventService.deleteEvent(slug).subscribe({
      next: () => {
        this.events.update(events =>
          events.filter(e => e.slug !== slug)
        );
      },
      error: (err) => {
        this.error.set(err.error?.error || 'Failed to delete event');
      }
    });
  }

  getEventStatusClass(status: string): string {
    switch (status) {
      case 'draft':
        return 'bg-gray-100 text-gray-600';
      case 'registration':
        return 'bg-blue-100 text-blue-600';
      case 'in_progress':
        return 'bg-green-100 text-green-600';
      case 'completed':
        return 'bg-purple-100 text-purple-600';
      case 'cancelled':
        return 'bg-red-100 text-red-600';
      default:
        return 'bg-gray-100 text-gray-600';
    }
  }

  formatStatus(status: string): string {
    return status.replace('_', ' ');
  }

  onTournamentClick(slug: string): void {
    this.router.navigate(['/tournaments', slug]);
  }

  deleteTournament(slug: string, name: string, event: Event): void {
    event.stopPropagation();

    if (!confirm(`Are you sure you want to delete "${name}"? This cannot be undone.`)) {
      return;
    }

    this.tournamentService.deleteTournament(slug).subscribe({
      next: () => {
        this.tournaments.update(tournaments =>
          tournaments.filter(t => t.slug !== slug)
        );
      },
      error: (err) => {
        this.error.set(err.error?.error || 'Failed to delete tournament');
      }
    });
  }
}
