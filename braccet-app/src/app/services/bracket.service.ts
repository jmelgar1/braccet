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
  Participant as BracketParticipant,
  SwissBracketState,
  EliminationStandingsResponse,
  GroupBracketState,
  GroupStanding,
  StageStanding
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

  updateGroupStage(tournamentId: number, stageId: number, groupId: number, round: number, request: UpdateStageRequest): Observable<BracketStage> {
    return this.http.put<BracketStage>(
      `${this.apiUrl}/${tournamentId}/stages/${stageId}/groups/${groupId}/bracket-stages/${round}`,
      request
    );
  }

  // Swiss-specific methods

  /**
   * Get Swiss bracket state including standings and matches.
   */
  getSwissBracket(tournamentId: number): Observable<SwissBracketState> {
    return this.http.get<SwissBracketState>(`${this.apiUrl}/${tournamentId}/standings`);
  }

  /**
   * Get elimination standings for single/double elimination brackets.
   */
  getEliminationStandings(tournamentId: number): Observable<EliminationStandingsResponse> {
    return this.http.get<EliminationStandingsResponse>(`${this.apiUrl}/${tournamentId}/elimination-standings`);
  }

  /**
   * Advance to the next round in a Swiss tournament.
   * Returns the new matches for the next round, or null if tournament is complete.
   */
  advanceSwissRound(tournamentId: number): Observable<{ matches: Match[] } | null> {
    return this.http.post<{ matches: Match[] } | null>(
      `${this.apiUrl}/${tournamentId}/advance-round`,
      {}
    );
  }

  /**
   * Advance to the next round in a Swiss group within a multi-stage tournament.
   * Returns the new matches for the next round, or null if group bracket is complete.
   */
  advanceGroupSwissRound(tournamentId: number, stageId: number, groupId: number): Observable<{ matches: Match[] } | null> {
    return this.http.post<{ matches: Match[] } | null>(
      `${this.apiUrl}/${tournamentId}/stages/${stageId}/groups/${groupId}/advance-round`,
      {}
    );
  }

  /**
   * Reseed a Swiss round with new pairings.
   * If the round is before current_round, cascades and deletes subsequent rounds.
   */
  reseedSwissRound(tournamentId: number, round: number): Observable<{ matches: Match[] }> {
    return this.http.post<{ matches: Match[] }>(
      `${this.apiUrl}/${tournamentId}/reseed-round`,
      { round }
    );
  }

  /**
   * Reseed a Swiss round in a group with new pairings.
   * If the round is before current_round, cascades and deletes subsequent rounds.
   */
  reseedGroupSwissRound(tournamentId: number, stageId: number, groupId: number, round: number): Observable<{ matches: Match[] }> {
    return this.http.post<{ matches: Match[] }>(
      `${this.apiUrl}/${tournamentId}/stages/${stageId}/groups/${groupId}/reseed-round`,
      { round }
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

  // Multi-stage / Group bracket methods

  /**
   * Get bracket for a specific stage in a multi-stage tournament (e.g., finals stage).
   * Unlike getBracket which returns all matches, this filters by stage_id.
   */
  getStageBracket(tournamentId: number, stageId: number): Observable<BracketState> {
    return this.http.get<BracketState>(`${this.apiUrl}/${tournamentId}/stages/${stageId}`);
  }

  /**
   * Get bracket for a specific group in a multi-stage tournament.
   */
  getGroupBracket(tournamentId: number, stageId: number, groupId: number): Observable<GroupBracketState> {
    return this.http.get<GroupBracketState>(`${this.apiUrl}/${tournamentId}/stages/${stageId}/groups/${groupId}`);
  }

  /**
   * Get Swiss bracket for a specific group in a multi-stage tournament.
   */
  getGroupSwissBracket(tournamentId: number, stageId: number, groupId: number): Observable<SwissBracketState> {
    return this.http.get<SwissBracketState>(`${this.apiUrl}/${tournamentId}/stages/${stageId}/groups/${groupId}`);
  }

  /**
   * Get standings for a specific group.
   */
  getGroupStandings(tournamentId: number, groupId: number): Observable<GroupStanding[]> {
    return this.http.get<GroupStanding[]>(`${this.apiUrl}/${tournamentId}/groups/${groupId}/standings`);
  }

  /**
   * Get cross-group standings for a stage.
   */
  getStageStandings(tournamentId: number, stageId: number): Observable<StageStanding[]> {
    return this.http.get<StageStanding[]>(`${this.apiUrl}/${tournamentId}/stages/${stageId}/standings`);
  }

  /**
   * Generate bracket for a group (starts the group's competition).
   */
  generateGroupBracket(tournamentId: number, groupId: number): Observable<GroupBracketState> {
    return this.http.post<GroupBracketState>(`${this.apiUrl}/${tournamentId}/groups/${groupId}/bracket`, {});
  }

  /**
   * Complete a stage (finalize rankings when all groups are done).
   */
  completeStage(tournamentId: number, stageId: number): Observable<StageStanding[]> {
    return this.http.post<StageStanding[]>(`${this.apiUrl}/${tournamentId}/stages/${stageId}/complete`, {});
  }

  /**
   * Reseed (regenerate) the final bracket from advancing participants.
   * This deletes all matches in the final bracket and creates a new bracket.
   */
  reseedFinalBracket(tournamentId: number, stageId: number, format: string): Observable<{ status: string }> {
    return this.http.post<{ status: string }>(
      `${this.apiUrl}/${tournamentId}/stages/${stageId}/reseed`,
      { format }
    );
  }

  /**
   * Reseed (regenerate) a group bracket.
   * This deletes all matches in the group bracket and creates a new bracket.
   */
  reseedGroupBracket(tournamentId: number, stageId: number, groupId: number, format: string): Observable<{ status: string }> {
    return this.http.post<{ status: string }>(
      `${this.apiUrl}/${tournamentId}/stages/${stageId}/groups/${groupId}/reseed`,
      { format }
    );
  }

  /**
   * Reseed all groups in a stage with proper cross-group distribution.
   * This deletes all matches in all groups, redistributes participants using
   * serpentine seeding (1,4,5,8 vs 2,3,6,7 pattern), and regenerates brackets.
   */
  reseedStage(tournamentId: number, stageId: number, format: string): Observable<{ status: string }> {
    return this.http.post<{ status: string }>(
      `${this.apiUrl}/${tournamentId}/stages/${stageId}/reseed-stage`,
      { format }
    );
  }
}
