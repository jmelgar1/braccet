import { Component, inject, signal, computed, OnInit } from '@angular/core';
import { DatePipe } from '@angular/common';
import { ActivatedRoute } from '@angular/router';
import { switchMap } from 'rxjs';
import { TournamentService } from '../../services/tournament.service';
import { BracketService } from '../../services/bracket.service';
import { AuthService } from '../../services/auth.service';
import { Tournament, Participant } from '../../models/tournament.model';
import { Breadcrumb, BreadcrumbItem } from '../../components/breadcrumb/breadcrumb';
import { SidePanel } from './components/side-panel/side-panel';

@Component({
  selector: 'app-tournament-detail',
  imports: [DatePipe, Breadcrumb, SidePanel],
  templateUrl: './tournament-detail.html',
  styleUrl: './tournament-detail.css'
})
export class TournamentDetail implements OnInit {
  private route = inject(ActivatedRoute);
  private tournamentService = inject(TournamentService);
  private bracketService = inject(BracketService);
  authService = inject(AuthService);

  tournament = signal<Tournament | null>(null);
  loading = signal(true);
  error = signal('');
  startingTournament = signal(false);

  // Participant state
  participants = signal<Participant[]>([]);
  participantsLoading = signal(false);

  // Computed properties
  isOrganizer = computed(() => {
    const t = this.tournament();
    const user = this.authService.user();
    return t && user ? t.organizer_id === user.id : false;
  });

  isLoggedIn = computed(() => this.authService.isLoggedIn());

  canStartTournament = computed(() => {
    const t = this.tournament();
    if (!t) return false;
    return this.isOrganizer() &&
           t.status === 'registration' &&
           this.participants().length >= 2;
  });

  breadcrumbs: BreadcrumbItem[] = [
    { label: 'Tournaments', route: '/tournaments' },
    { label: 'Loading...' }
  ];

  ngOnInit(): void {
    const slug = this.route.snapshot.paramMap.get('slug');
    if (slug) {
      this.loadTournament(slug);
    } else {
      this.error.set('Tournament not found');
      this.loading.set(false);
    }
  }

  loadTournament(slug: string): void {
    this.loading.set(true);
    this.error.set('');

    this.tournamentService.getTournament(slug).subscribe({
      next: (tournament) => {
        this.tournament.set(tournament);
        this.breadcrumbs = [
          { label: 'Tournaments', route: '/tournaments' },
          { label: tournament.name }
        ];
        this.loading.set(false);
        this.loadParticipants(slug);
      },
      error: (err) => {
        this.error.set(err.error?.error || 'Failed to load tournament');
        this.loading.set(false);
      }
    });
  }

  loadParticipants(slug: string): void {
    this.participantsLoading.set(true);

    this.tournamentService.getParticipants(slug).subscribe({
      next: (participants) => {
        this.participants.set(participants || []);
        this.participantsLoading.set(false);
      },
      error: () => {
        this.participants.set([]);
        this.participantsLoading.set(false);
      }
    });
  }

  getStatusLabel(status: string): string {
    const labels: Record<string, string> = {
      registration: 'Registration Open',
      in_progress: 'In Progress',
      completed: 'Completed',
      cancelled: 'Cancelled'
    };
    return labels[status] || status;
  }

  getStatusColor(status: string): string {
    const colors: Record<string, string> = {
      registration: 'bg-green-100 text-green-800',
      in_progress: 'bg-blue-100 text-blue-800',
      completed: 'bg-purple-100 text-purple-800',
      cancelled: 'bg-red-100 text-red-800'
    };
    return colors[status] || 'bg-gray-100 text-gray-800';
  }

  // Event handlers from SidePanel
  onParticipantAdded(participant: Participant): void {
    this.participants.update(list => [...list, participant]);
  }

  onParticipantRemoved(id: number): void {
    this.participants.update(list => list.filter(p => p.id !== id));
  }

  onSeedingChanged(participants: Participant[]): void {
    this.participants.set(participants);
  }

  onSelfRegistered(participant: Participant): void {
    this.participants.update(list => [...list, participant]);
  }

  onLeft(id: number): void {
    this.participants.update(list => list.filter(p => p.id !== id));
  }

  onTournamentUpdated(tournament: Tournament): void {
    this.tournament.set(tournament);
    this.breadcrumbs = [
      { label: 'Tournaments', route: '/tournaments' },
      { label: tournament.name }
    ];
  }

  startTournament(): void {
    const t = this.tournament();
    const p = this.participants();
    if (!t || p.length < 2) return;

    this.startingTournament.set(true);
    this.error.set('');

    const bracketParticipants = p.map((participant, index) => ({
      id: participant.id,
      name: participant.display_name,
      seed: index + 1
    }));

    this.bracketService.createBracket({
      tournament_id: t.id,
      format: t.format,
      participants: bracketParticipants
    }).pipe(
      switchMap(() => this.tournamentService.updateTournament(t.slug, { status: 'in_progress' }))
    ).subscribe({
      next: (updatedTournament) => {
        this.tournament.set(updatedTournament);
        this.startingTournament.set(false);
      },
      error: (err) => {
        this.error.set(err.error?.error || 'Failed to start tournament');
        this.startingTournament.set(false);
      }
    });
  }
}
