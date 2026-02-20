import { Component, inject, signal, computed, OnInit } from '@angular/core';
import { DatePipe } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute } from '@angular/router';
import { CdkDragDrop, DragDropModule, moveItemInArray } from '@angular/cdk/drag-drop';
import { TournamentService } from '../../services/tournament.service';
import { AuthService } from '../../services/auth.service';
import { Tournament, Participant } from '../../models/tournament.model';
import { Breadcrumb, BreadcrumbItem } from '../../components/breadcrumb/breadcrumb';

@Component({
  selector: 'app-tournament-detail',
  imports: [DatePipe, FormsModule, DragDropModule, Breadcrumb],
  templateUrl: './tournament-detail.html',
  styleUrl: './tournament-detail.css'
})
export class TournamentDetail implements OnInit {
  private route = inject(ActivatedRoute);
  private tournamentService = inject(TournamentService);
  authService = inject(AuthService);

  tournament = signal<Tournament | null>(null);
  loading = signal(true);
  error = signal('');

  // Participant state
  participants = signal<Participant[]>([]);
  participantsLoading = signal(false);
  participantsError = signal('');
  newParticipantName = '';
  addingParticipant = signal(false);

  // Seeding state
  seedingMode = signal(false);
  seedingOrder = signal<Participant[]>([]);
  savingSeeding = signal(false);

  // Computed properties
  isOrganizer = computed(() => {
    const t = this.tournament();
    const user = this.authService.user();
    return t && user ? t.organizer_id === user.id : false;
  });

  isLoggedIn = computed(() => this.authService.isLoggedIn());

  currentUserParticipant = computed(() => {
    const user = this.authService.user();
    if (!user) return null;
    return this.participants().find(p => p.user_id === user.id) || null;
  });

  canSelfRegister = computed(() => {
    const t = this.tournament();
    return t && t.registration_open && this.isLoggedIn() && !this.isOrganizer() && !this.currentUserParticipant();
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
    this.participantsError.set('');

    this.tournamentService.getParticipants(slug).subscribe({
      next: (participants) => {
        this.participants.set(participants || []);
        this.participantsLoading.set(false);
      },
      error: () => {
        // For any error loading participants, just show empty state
        // The main tournament details are already loaded, no need for error message
        this.participants.set([]);
        this.participantsLoading.set(false);
      }
    });
  }

  addParticipant(): void {
    const t = this.tournament();
    if (!t || !this.newParticipantName.trim()) return;

    this.addingParticipant.set(true);
    this.tournamentService.addParticipant(t.slug, {
      display_name: this.newParticipantName.trim()
    }).subscribe({
      next: (participant) => {
        this.participants.update(list => [...list, participant]);
        this.newParticipantName = '';
        this.addingParticipant.set(false);
      },
      error: (err) => {
        this.participantsError.set(err.error?.error || 'Failed to add participant');
        this.addingParticipant.set(false);
      }
    });
  }

  selfRegister(): void {
    const t = this.tournament();
    const user = this.authService.user();
    if (!t || !user) return;

    this.addingParticipant.set(true);
    this.tournamentService.addParticipant(t.slug, {
      user_id: user.id,
      display_name: user.display_name
    }).subscribe({
      next: (participant) => {
        this.participants.update(list => [...list, participant]);
        this.addingParticipant.set(false);
      },
      error: (err) => {
        this.participantsError.set(err.error?.error || 'Failed to join tournament');
        this.addingParticipant.set(false);
      }
    });
  }

  removeParticipant(participant: Participant): void {
    const t = this.tournament();
    if (!t) return;

    this.tournamentService.removeParticipant(t.slug, participant.id).subscribe({
      next: () => {
        this.participants.update(list => list.filter(p => p.id !== participant.id));
      },
      error: (err) => {
        this.participantsError.set(err.error?.error || 'Failed to remove participant');
      }
    });
  }

  leaveTournament(): void {
    const participant = this.currentUserParticipant();
    if (participant) {
      this.removeParticipant(participant);
    }
  }

  getStatusLabel(status: string): string {
    const labels: Record<string, string> = {
      draft: 'Draft',
      registration: 'Registration Open',
      in_progress: 'In Progress',
      completed: 'Completed',
      cancelled: 'Cancelled'
    };
    return labels[status] || status;
  }

  getStatusColor(status: string): string {
    const colors: Record<string, string> = {
      draft: 'bg-gray-100 text-gray-800',
      registration: 'bg-green-100 text-green-800',
      in_progress: 'bg-blue-100 text-blue-800',
      completed: 'bg-purple-100 text-purple-800',
      cancelled: 'bg-red-100 text-red-800'
    };
    return colors[status] || 'bg-gray-100 text-gray-800';
  }

  // Seeding methods
  enterSeedingMode(): void {
    this.seedingOrder.set([...this.participants()]);
    this.seedingMode.set(true);
  }

  exitSeedingMode(): void {
    this.seedingMode.set(false);
    this.seedingOrder.set([]);
  }

  onDrop(event: CdkDragDrop<Participant[]>): void {
    const list = [...this.seedingOrder()];
    moveItemInArray(list, event.previousIndex, event.currentIndex);
    this.seedingOrder.set(list);
  }

  saveSeeding(): void {
    const t = this.tournament();
    if (!t) return;

    // Build seeds map: participantId -> seed (1-indexed)
    const seeds: Record<number, number> = {};
    this.seedingOrder().forEach((p, index) => {
      seeds[p.id] = index + 1;
    });

    this.savingSeeding.set(true);
    this.tournamentService.updateSeeding(t.slug, { seeds }).subscribe({
      next: (updatedParticipants) => {
        this.participants.set(updatedParticipants);
        this.savingSeeding.set(false);
        this.exitSeedingMode();
      },
      error: (err) => {
        this.participantsError.set(err.error?.error || 'Failed to update seeding');
        this.savingSeeding.set(false);
      }
    });
  }
}
