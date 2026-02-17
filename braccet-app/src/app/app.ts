import { Component, signal, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { FormsModule } from '@angular/forms';
import { environment } from '../environments/environment';

interface Participant {
  id: number;
  name: string;
  seed: number;
}

interface Match {
  id: number;
  round: number;
  position: number;
  participant1_id?: number;
  participant2_id?: number;
  participant1_name?: string;
  participant2_name?: string;
  participant1_score?: number;
  participant2_score?: number;
  winner_id?: number;
  status: string;
  next_match_id?: number;
}

interface BracketState {
  tournament_id: number;
  total_rounds: number;
  current_round: number;
  is_complete: boolean;
  champion_id?: number;
  matches: Match[];
}

@Component({
  selector: 'app-root',
  imports: [FormsModule],
  templateUrl: './app.html',
  styleUrl: './app.css'
})
export class App {
  private http = inject(HttpClient);
  private apiUrl = environment.apiUrl;

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

    this.http.post<BracketState>(`${this.apiUrl}/brackets`, {
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
    this.http.get<BracketState>(`${this.apiUrl}/brackets/${this.tournamentId()}`)
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
    this.http.post<Match>(`${this.apiUrl}/brackets/matches/${match.id}/start`, {})
      .subscribe({
        next: () => this.refreshBracket(),
        error: (err) => this.error.set(err.error?.error || 'Failed to start match')
      });
  }

  reportResult(winnerId: number) {
    const match = this.selectedMatch();
    if (!match) return;

    this.http.post<Match>(`${this.apiUrl}/brackets/matches/${match.id}/result`, {
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
