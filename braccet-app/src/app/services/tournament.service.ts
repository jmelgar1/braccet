import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../environments/environment';
import { Tournament, CreateTournamentRequest } from '../models/tournament.model';

@Injectable({ providedIn: 'root' })
export class TournamentService {
  private http = inject(HttpClient);
  private baseUrl = `${environment.apiUrl}/tournaments`;

  getTournaments(): Observable<Tournament[]> {
    return this.http.get<Tournament[]>(this.baseUrl);
  }

  getTournament(id: number): Observable<Tournament> {
    return this.http.get<Tournament>(`${this.baseUrl}/${id}`);
  }

  createTournament(request: CreateTournamentRequest): Observable<Tournament> {
    return this.http.post<Tournament>(this.baseUrl, request);
  }

  updateTournament(id: number, request: Partial<Tournament>): Observable<Tournament> {
    return this.http.put<Tournament>(`${this.baseUrl}/${id}`, request);
  }

  deleteTournament(id: number): Observable<void> {
    return this.http.delete<void>(`${this.baseUrl}/${id}`);
  }
}
