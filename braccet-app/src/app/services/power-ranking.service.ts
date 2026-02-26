import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../environments/environment';
import {
  PRSystem,
  CreatePRSystemRequest,
  UpdatePRSystemRequest,
  MemberPowerRanking,
  TournamentPlacement
} from '../models/power-ranking.model';

@Injectable({ providedIn: 'root' })
export class PowerRankingService {
  private http = inject(HttpClient);
  private baseUrl = `${environment.apiUrl}/communities`;

  // PR System CRUD
  getPRSystems(slug: string): Observable<PRSystem[]> {
    return this.http.get<PRSystem[]>(`${this.baseUrl}/${slug}/power-rankings/systems`);
  }

  getPRSystem(slug: string, systemId: number): Observable<PRSystem> {
    return this.http.get<PRSystem>(`${this.baseUrl}/${slug}/power-rankings/systems/${systemId}`);
  }

  createPRSystem(slug: string, request: CreatePRSystemRequest): Observable<PRSystem> {
    return this.http.post<PRSystem>(`${this.baseUrl}/${slug}/power-rankings/systems`, request);
  }

  updatePRSystem(slug: string, systemId: number, request: UpdatePRSystemRequest): Observable<PRSystem> {
    return this.http.put<PRSystem>(`${this.baseUrl}/${slug}/power-rankings/systems/${systemId}`, request);
  }

  deletePRSystem(slug: string, systemId: number): Observable<void> {
    return this.http.delete<void>(`${this.baseUrl}/${slug}/power-rankings/systems/${systemId}`);
  }

  // Leaderboard
  getLeaderboard(slug: string, systemId: number, limit?: number): Observable<MemberPowerRanking[]> {
    const params = limit ? `?limit=${limit}` : '';
    return this.http.get<MemberPowerRanking[]>(
      `${this.baseUrl}/${slug}/power-rankings/systems/${systemId}/leaderboard${params}`
    );
  }

  // Member rankings and placements
  getMemberRankings(slug: string, memberId: number): Observable<MemberPowerRanking[]> {
    return this.http.get<MemberPowerRanking[]>(
      `${this.baseUrl}/${slug}/members/${memberId}/power-rankings`
    );
  }

  getMemberPlacements(slug: string, memberId: number, systemId: number): Observable<TournamentPlacement[]> {
    return this.http.get<TournamentPlacement[]>(
      `${this.baseUrl}/${slug}/members/${memberId}/power-rankings/${systemId}/placements`
    );
  }
}
