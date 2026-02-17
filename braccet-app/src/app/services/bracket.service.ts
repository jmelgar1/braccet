import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../environments/environment';
import {
  BracketState,
  Match,
  CreateBracketRequest,
  ReportResultRequest
} from '../models/bracket.model';

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
}
