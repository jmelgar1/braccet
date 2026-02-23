import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { map } from 'rxjs/operators';
import { environment } from '../../environments/environment';
import {
  BracketState,
  BracketStage,
  Match,
  CreateBracketRequest,
  ReportResultRequest,
  EditResultResponse,
  UpdateStageRequest,
  BracketFormat,
  Participant as BracketParticipant
} from '../models/bracket.model';
import { Participant as TournamentParticipant } from '../models/tournament.model';
import { BracketPreview } from './bracket-generator.service';

// Request type for preview endpoint
interface PreviewRequest {
  tournament_id: number;
  format: BracketFormat;
  participants: BracketParticipant[];
}

@Injectable({
  providedIn: 'root'
})
export class BracketService {
  private http = inject(HttpClient);
  private apiUrl = `${environment.apiUrl}/brackets`;

  createBracket(request: CreateBracketRequest): Observable<BracketState> {
    return this.http.post<BracketState>(this.apiUrl, request);
  }

  getBracket(tournamentId: number): Observable<BracketState> {
    return this.http.get<BracketState>(`${this.apiUrl}/${tournamentId}`);
  }

  startMatch(matchId: number): Observable<Match> {
    return this.http.post<Match>(`${this.apiUrl}/matches/${matchId}/start`, {});
  }

  reportResult(matchId: number, request: ReportResultRequest): Observable<Match> {
    return this.http.post<Match>(`${this.apiUrl}/matches/${matchId}/result`, request);
  }

  reopenMatch(matchId: number): Observable<{ reopened_matches: Match[] }> {
    return this.http.post<{ reopened_matches: Match[] }>(
      `${this.apiUrl}/matches/${matchId}/reopen`,
      {}
    );
  }

  editResult(matchId: number, request: ReportResultRequest): Observable<EditResultResponse> {
    return this.http.put<EditResultResponse>(
      `${this.apiUrl}/matches/${matchId}/result`,
      request
    );
  }

  updateStage(tournamentId: number, round: number, request: UpdateStageRequest): Observable<BracketStage> {
    return this.http.put<BracketStage>(
      `${this.apiUrl}/${tournamentId}/stages/${round}`,
      request
    );
  }

  /**
   * Get bracket preview without persisting to database.
   * Uses same backend engine as actual bracket generation, ensuring identical BYE logic.
   */
  getPreview(tournamentId: number, format: BracketFormat, participants: TournamentParticipant[]): Observable<BracketPreview> {
    // Map tournament participants to bracket participant format
    const bracketParticipants: BracketParticipant[] = participants.map((p, index) => ({
      id: p.id,
      name: p.display_name,
      seed: p.seed ?? (index + 1), // Use seed if available, otherwise use index+1
      icon_url: p.icon_url
    }));

    const request: PreviewRequest = {
      tournament_id: tournamentId,
      format,
      participants: bracketParticipants
    };

    return this.http.post<BracketState>(`${this.apiUrl}/preview`, request).pipe(
      map(state => this.toBracketPreview(state, participants.length))
    );
  }

  /**
   * Convert backend BracketState to frontend BracketPreview format.
   * This ensures the preview response matches the format expected by bracket viewer components.
   */
  private toBracketPreview(state: BracketState, participantCount: number): BracketPreview {
    // Calculate bracket size (smallest power of 2 >= participantCount)
    let bracketSize = 1;
    while (bracketSize < participantCount) {
      bracketSize *= 2;
    }

    return {
      format: state.format,
      totalRounds: state.total_rounds,
      winnersRounds: state.winners_rounds,
      losersRounds: state.losers_rounds,
      bracketSize,
      matches: state.matches,
      stages: state.stages
    };
  }
}
