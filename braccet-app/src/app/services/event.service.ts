import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../environments/environment';
import {
  Event,
  CreateEventRequest,
  UpdateEventRequest,
  AddTournamentToEventRequest,
  UpdateEventTournamentRequest,
  QualificationStatusResponse,
  AdvancementResponse,
  EventGraphResponse,
  UpdateConnectionsRequest
} from '../models/event.model';
import { Tournament } from '../models/tournament.model';

@Injectable({ providedIn: 'root' })
export class EventService {
  private http = inject(HttpClient);
  private baseUrl = `${environment.apiUrl}/events`;

  getEvents(): Observable<Event[]> {
    return this.http.get<Event[]>(this.baseUrl);
  }

  getEventsByCommunity(communityId: number): Observable<Event[]> {
    return this.http.get<Event[]>(`${environment.apiUrl}/communities/${communityId}/events`);
  }

  getEvent(slug: string): Observable<Event> {
    return this.http.get<Event>(`${this.baseUrl}/${slug}`);
  }

  createEvent(request: CreateEventRequest): Observable<Event> {
    return this.http.post<Event>(this.baseUrl, request);
  }

  updateEvent(slug: string, request: UpdateEventRequest): Observable<Event> {
    return this.http.put<Event>(`${this.baseUrl}/${slug}`, request);
  }

  deleteEvent(slug: string): Observable<void> {
    return this.http.delete<void>(`${this.baseUrl}/${slug}`);
  }

  // Event tournament management
  addTournament(eventSlug: string, request: AddTournamentToEventRequest): Observable<any> {
    return this.http.post(`${this.baseUrl}/${eventSlug}/tournaments`, request);
  }

  updateTournament(eventSlug: string, tournamentId: number, request: UpdateEventTournamentRequest): Observable<void> {
    return this.http.put<void>(`${this.baseUrl}/${eventSlug}/tournaments/${tournamentId}`, request);
  }

  removeTournament(eventSlug: string, tournamentId: number): Observable<void> {
    return this.http.delete<void>(`${this.baseUrl}/${eventSlug}/tournaments/${tournamentId}`);
  }

  // Qualification and advancement
  getQualificationStatus(slug: string): Observable<QualificationStatusResponse> {
    return this.http.get<QualificationStatusResponse>(`${this.baseUrl}/${slug}/qualification-status`);
  }

  triggerAdvancement(eventSlug: string, tournamentId: number): Observable<AdvancementResponse> {
    return this.http.post<AdvancementResponse>(`${this.baseUrl}/${eventSlug}/tournaments/${tournamentId}/advance`, {});
  }

  // Get tournaments available to add to an event (same community, not assigned to any event)
  getAvailableTournaments(eventSlug: string): Observable<Tournament[]> {
    return this.http.get<Tournament[]>(`${this.baseUrl}/${eventSlug}/available-tournaments`);
  }

  // Logo upload
  getLogoUploadUrl(eventSlug: string, contentType: string): Observable<{ upload_url: string; logo_url: string; expires_at: string }> {
    return this.http.post<{ upload_url: string; logo_url: string; expires_at: string }>(
      `${this.baseUrl}/${eventSlug}/logo/upload-url`,
      { content_type: contentType }
    );
  }

  updateLogoUrl(eventSlug: string, logoUrl: string): Observable<{ logo_url: string }> {
    return this.http.put<{ logo_url: string }>(`${this.baseUrl}/${eventSlug}/logo`, { logo_url: logoUrl });
  }

  // DAG Editor endpoints
  getEventGraph(slug: string): Observable<EventGraphResponse> {
    return this.http.get<EventGraphResponse>(`${this.baseUrl}/${slug}/graph`);
  }

  updateConnections(slug: string, request: UpdateConnectionsRequest): Observable<void> {
    return this.http.put<void>(`${this.baseUrl}/${slug}/connections`, request);
  }
}
