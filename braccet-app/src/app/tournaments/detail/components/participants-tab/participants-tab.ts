import { Component, input, output, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { CdkDragDrop, DragDropModule, moveItemInArray } from '@angular/cdk/drag-drop';
import { Tournament, Participant } from '../../../../models/tournament.model';
import { TournamentService } from '../../../../services/tournament.service';
import { AuthService } from '../../../../services/auth.service';

@Component({
  selector: 'app-participants-tab',
  imports: [FormsModule, DragDropModule],
  templateUrl: './participants-tab.html'
})
export class ParticipantsTab {
  private tournamentService = inject(TournamentService);
  authService = inject(AuthService);

  tournament = input.required<Tournament>();
  participants = input.required<Participant[]>();
  isOrganizer = input.required<boolean>();
  currentUserParticipant = input<Participant | null>(null);
  canSelfRegister = input(false);

  participantAdded = output<Participant>();
  participantRemoved = output<number>();
  participantWithdrawn = output<number>();
  seedingChanged = output<Participant[]>();
  selfRegistered = output<Participant>();
  left = output<number>();

  newParticipantName = '';
  addingParticipant = signal(false);
  savingSeeding = signal(false);
  error = signal('');

  addParticipant(): void {
    const t = this.tournament();
    const name = this.newParticipantName.trim();
    if (!t || !name) return;

    // Check for duplicate name (case-insensitive)
    if (this.isDuplicateName(name)) {
      this.error.set('A participant with this name already exists');
      return;
    }

    this.addingParticipant.set(true);
    this.error.set('');

    this.tournamentService.addParticipant(t.slug, {
      display_name: name
    }).subscribe({
      next: (participant) => {
        this.participantAdded.emit(participant);
        this.newParticipantName = '';
        this.addingParticipant.set(false);
      },
      error: (err) => {
        this.error.set(err.error?.error || 'Failed to add participant');
        this.addingParticipant.set(false);
      }
    });
  }

  selfRegister(): void {
    const t = this.tournament();
    const user = this.authService.user();
    if (!t || !user) return;

    this.addingParticipant.set(true);
    this.error.set('');

    this.tournamentService.addParticipant(t.slug, {
      user_id: user.id,
      display_name: user.display_name
    }).subscribe({
      next: (participant) => {
        this.selfRegistered.emit(participant);
        this.addingParticipant.set(false);
      },
      error: (err) => {
        this.error.set(err.error?.error || 'Failed to join tournament');
        this.addingParticipant.set(false);
      }
    });
  }

  removeParticipant(participant: Participant): void {
    const t = this.tournament();
    if (!t) return;

    this.tournamentService.removeParticipant(t.slug, participant.id).subscribe({
      next: () => {
        this.participantRemoved.emit(participant.id);
      },
      error: (err) => {
        this.error.set(err.error?.error || 'Failed to remove participant');
      }
    });
  }

  leaveTournament(): void {
    const participant = this.currentUserParticipant();
    if (participant) {
      this.tournamentService.removeParticipant(this.tournament().slug, participant.id).subscribe({
        next: () => {
          this.left.emit(participant.id);
        },
        error: (err) => {
          this.error.set(err.error?.error || 'Failed to leave tournament');
        }
      });
    }
  }

  withdrawParticipant(participant: Participant): void {
    const t = this.tournament();
    if (!t) return;

    if (!confirm(`Are you sure you want to withdraw ${participant.display_name}? This will forfeit all their pending matches.`)) {
      return;
    }

    this.tournamentService.withdrawParticipant(t.slug, participant.id).subscribe({
      next: () => {
        this.participantWithdrawn.emit(participant.id);
      },
      error: (err) => {
        this.error.set(err.error?.error || 'Failed to withdraw participant');
      }
    });
  }

  canWithdraw(participant: Participant): boolean {
    const t = this.tournament();
    return t.status === 'in_progress' &&
           participant.status !== 'eliminated' &&
           participant.status !== 'disqualified' &&
           participant.status !== 'withdrawn';
  }

  onDrop(event: CdkDragDrop<Participant[]>): void {
    if (event.previousIndex === event.currentIndex) return;

    const list = [...this.participants()];
    moveItemInArray(list, event.previousIndex, event.currentIndex);
    this.saveSeeding(list);
  }

  private isDuplicateName(name: string): boolean {
    const lowerName = name.toLowerCase();
    return this.participants().some(p => p.display_name.toLowerCase() === lowerName);
  }

  private saveSeeding(orderedParticipants: Participant[]): void {
    const t = this.tournament();
    if (!t) return;

    const seeds: Record<number, number> = {};
    orderedParticipants.forEach((p, index) => {
      seeds[p.id] = index + 1;
    });

    this.savingSeeding.set(true);
    this.tournamentService.updateSeeding(t.slug, { seeds }).subscribe({
      next: (updatedParticipants) => {
        this.seedingChanged.emit(updatedParticipants);
        this.savingSeeding.set(false);
      },
      error: (err) => {
        this.error.set(err.error?.error || 'Failed to update seeding');
        this.savingSeeding.set(false);
      }
    });
  }
}
