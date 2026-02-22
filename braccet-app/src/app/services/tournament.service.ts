import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../environments/environment';
import { Tournament, CreateTournamentRequest, Participant, AddParticipantRequest, UpdateSeedingRequest } from '../models/tournament.model';

@Injectable({ providedIn: 'root' })
export class TournamentService {
  private http = inject(HttpClient);
  private baseUrl = `${environment.apiUrl}/tournaments`;

  getTournaments(): Observable<Tournament[]> {
    return this.http.get<Tournament[]>(this.baseUrl);
  }

  getTournamentsByCommunity(communityId: number): Observable<Tournament[]> {
    return this.http.get<Tournament[]>(`${this.baseUrl}/community/${communityId}`);
  }

  getTournament(slug: string): Observable<Tournament> {
    return this.http.get<Tournament>(`${this.baseUrl}/${slug}`);
  }

  createTournament(request: CreateTournamentRequest): Observable<Tournament> {
    return this.http.post<Tournament>(this.baseUrl, request);
  }

  updateTournament(slug: string, request: Partial<Tournament>): Observable<Tournament> {
    return this.http.put<Tournament>(`${this.baseUrl}/${slug}`, request);
  }

  deleteTournament(slug: string): Observable<void> {
    return this.http.delete<void>(`${this.baseUrl}/${slug}`);
  }

  // Participant methods
  getParticipants(slug: string): Observable<Participant[]> {
    return this.http.get<Participant[]>(`${this.baseUrl}/${slug}/participants`);
  }

  addParticipant(slug: string, request: AddParticipantRequest): Observable<Participant> {
    return this.http.post<Participant>(`${this.baseUrl}/${slug}/participants`, request);
  }

  removeParticipant(slug: string, participantId: number): Observable<void> {
    return this.http.delete<void>(`${this.baseUrl}/${slug}/participants/${participantId}`);
  }

  withdrawParticipant(slug: string, participantId: number): Observable<void> {
    return this.http.post<void>(`${this.baseUrl}/${slug}/participants/${participantId}/withdraw`, {});
  }

  updateSeeding(slug: string, request: UpdateSeedingRequest): Observable<Participant[]> {
    return this.http.put<Participant[]>(`${this.baseUrl}/${slug}/participants/seeding`, request);
  }
}
