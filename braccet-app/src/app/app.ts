import { Component, signal, inject } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { BracketService } from './services/bracket.service';
import { Participant, Match, BracketState } from './models/bracket.model';

@Component({
  selector: 'app-root',
  imports: [FormsModule],
  templateUrl: './app.html',
  styleUrl: './app.css'
})
export class App {
  private bracketService = inject(BracketService);

  participants = signal<Participant[]>([]);
  newParticipantName = signal('');
  tournamentId = signal(1);
  bracket = signal<BracketState | null>(null);
  error = signal('');
  loading = signal(false);
  selectedMatch = signal<Match | null>(null);
  score1 = signal(0);
  score2 = signal(0);

  addParticipant() {
    const name = this.newParticipantName().trim();
    if (!name) return;

    const current = this.participants();
    const newParticipant: Participant = {
      id: current.length + 1,
      name: name,
      seed: current.length + 1
    };

    this.participants.set([...current, newParticipant]);
    this.newParticipantName.set('');
  }

  removeParticipant(id: number) {
    const filtered = this.participants().filter(p => p.id !== id);
    const reseeded = filtered.map((p, i) => ({ ...p, id: i + 1, seed: i + 1 }));
    this.participants.set(reseeded);
  }

  generateBracket() {
    const participants = this.participants();
    if (participants.length < 2) {
      this.error.set('Need at least 2 participants');
      return;
    }

    this.loading.set(true);
    this.error.set('');

    this.bracketService.createBracket({
      tournament_id: this.tournamentId(),
      format: 'single_elimination',
      participants: participants
    }).subscribe({
      next: (bracket) => {
        this.bracket.set(bracket);
        this.loading.set(false);
      },
      error: (err) => {
        this.error.set(err.message || 'Failed to generate bracket');
        this.loading.set(false);
      }
    });
  }

  refreshBracket() {
    this.loading.set(true);
    this.bracketService.getBracket(this.tournamentId())
      .subscribe({
        next: (bracket) => {
          this.bracket.set(bracket);
          this.loading.set(false);
        },
        error: (err) => {
          this.error.set(err.message || 'Failed to fetch bracket');
          this.loading.set(false);
        }
      });
  }

  selectMatch(match: Match) {
    if (match.status === 'completed') return;
    this.selectedMatch.set(match);
    this.score1.set(0);
    this.score2.set(0);
  }

  startMatch(match: Match) {
    this.bracketService.startMatch(match.id)
      .subscribe({
        next: () => this.refreshBracket(),
        error: (err) => this.error.set(err.error?.error || 'Failed to start match')
      });
  }

  reportResult(winnerId: number) {
    const match = this.selectedMatch();
    if (!match) return;

    this.bracketService.reportResult(match.id, {
      winner_id: winnerId,
      participant1_score: this.score1(),
      participant2_score: this.score2()
    }).subscribe({
      next: () => {
        this.selectedMatch.set(null);
        this.refreshBracket();
      },
      error: (err) => this.error.set(err.error?.error || 'Failed to report result')
    });
  }

  getMatchesByRound(round: number): Match[] {
    return this.bracket()?.matches.filter(m => m.round === round) || [];
  }

  getRounds(): number[] {
    const total = this.bracket()?.total_rounds || 0;
    return Array.from({ length: total }, (_, i) => i + 1);
  }

  clearAll() {
    this.participants.set([]);
    this.bracket.set(null);
    this.error.set('');
    this.selectedMatch.set(null);
  }
}
