import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../environments/environment';
import {
  EloSystem,
  CreateEloSystemRequest,
  UpdateEloSystemRequest,
  MemberEloRating,
  EloHistory
} from '../models/elo.model';

@Injectable({ providedIn: 'root' })
export class EloService {
  private http = inject(HttpClient);
  private baseUrl = `${environment.apiUrl}/communities`;

  // ELO System methods
  getEloSystems(slug: string): Observable<EloSystem[]> {
    return this.http.get<EloSystem[]>(`${this.baseUrl}/${slug}/elo-systems`);
  }

  getEloSystem(slug: string, systemId: number): Observable<EloSystem> {
    return this.http.get<EloSystem>(`${this.baseUrl}/${slug}/elo-systems/${systemId}`);
  }

  createEloSystem(slug: string, request: CreateEloSystemRequest): Observable<EloSystem> {
    return this.http.post<EloSystem>(`${this.baseUrl}/${slug}/elo-systems`, request);
  }

  updateEloSystem(slug: string, systemId: number, request: UpdateEloSystemRequest): Observable<EloSystem> {
    return this.http.put<EloSystem>(`${this.baseUrl}/${slug}/elo-systems/${systemId}`, request);
  }

  deleteEloSystem(slug: string, systemId: number): Observable<void> {
    return this.http.delete<void>(`${this.baseUrl}/${slug}/elo-systems/${systemId}`);
  }

  // Leaderboard methods
  getLeaderboard(slug: string, systemId: number, limit?: number): Observable<MemberEloRating[]> {
    const params = limit ? `?limit=${limit}` : '';
    return this.http.get<MemberEloRating[]>(`${this.baseUrl}/${slug}/elo-systems/${systemId}/leaderboard${params}`);
  }

  // Member rating methods
  getMemberRatings(slug: string, memberId: number): Observable<MemberEloRating[]> {
    return this.http.get<MemberEloRating[]>(`${this.baseUrl}/${slug}/members/${memberId}/elo`);
  }

  getMemberHistory(slug: string, memberId: number, systemId: number, limit?: number): Observable<EloHistory[]> {
    const params = limit ? `?limit=${limit}` : '';
    return this.http.get<EloHistory[]>(`${this.baseUrl}/${slug}/members/${memberId}/elo/${systemId}/history${params}`);
  }
}
