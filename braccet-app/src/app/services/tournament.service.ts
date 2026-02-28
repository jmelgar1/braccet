import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../environments/environment';
import { Tournament, CreateTournamentRequest, CreateMultiStageTournamentRequest, Participant, AddParticipantRequest, UpdateSeedingRequest, MemberSearchResult, RegionInviteMember, TournamentStage, StageGroup, UpdateStageRequest, StageConfigRequest, StageSeedAssignment, UpdateStageSeedsRequest, StagePoolResponse, UpdateStagePoolRequest, SuggestedPrizeTiersResponse } from '../models/tournament.model';

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

  searchAvailableMembers(slug: string, query: string): Observable<MemberSearchResult[]> {
    return this.http.get<MemberSearchResult[]>(`${this.baseUrl}/${slug}/participants/search`, {
      params: { q: query }
    });
  }

  promoteParticipant(slug: string, participantId: number): Observable<Participant> {
    return this.http.post<Participant>(`${this.baseUrl}/${slug}/participants/${participantId}/promote`, {});
  }

  getTopMembersForRegionInvite(slug: string, region: string, count: number): Observable<RegionInviteMember[]> {
    return this.http.get<RegionInviteMember[]>(`${this.baseUrl}/${slug}/participants/invite-by-region`, {
      params: { region, count: count.toString() }
    });
  }

  // Multi-stage tournament methods

  createMultiStageTournament(request: CreateMultiStageTournamentRequest): Observable<Tournament> {
    return this.http.post<Tournament>(`${this.baseUrl}/multi-stage`, request);
  }

  getStages(slug: string): Observable<TournamentStage[]> {
    return this.http.get<TournamentStage[]>(`${this.baseUrl}/${slug}/stages`);
  }

  getGroups(slug: string, stageId: number): Observable<StageGroup[]> {
    return this.http.get<StageGroup[]>(`${this.baseUrl}/${slug}/stages/${stageId}/groups`);
  }

  startStage(slug: string): Observable<StageGroup[]> {
    return this.http.post<StageGroup[]>(`${this.baseUrl}/${slug}/stages/start`, {});
  }

  advanceStage(slug: string): Observable<TournamentStage> {
    return this.http.post<TournamentStage>(`${this.baseUrl}/${slug}/stages/advance`, {});
  }

  updateStage(slug: string, stageId: number, request: UpdateStageRequest): Observable<TournamentStage> {
    return this.http.put<TournamentStage>(`${this.baseUrl}/${slug}/stages/${stageId}`, request);
  }

  addStage(slug: string, request: StageConfigRequest): Observable<TournamentStage> {
    return this.http.post<TournamentStage>(`${this.baseUrl}/${slug}/stages`, request);
  }

  deleteStage(slug: string, stageId: number): Observable<void> {
    return this.http.delete<void>(`${this.baseUrl}/${slug}/stages/${stageId}`);
  }

  // Stage seeding methods (manual seed assignments before stage starts)

  getStageSeeds(slug: string, stageId: number): Observable<StageSeedAssignment[]> {
    return this.http.get<StageSeedAssignment[]>(`${this.baseUrl}/${slug}/stages/${stageId}/seeds`);
  }

  updateStageSeeds(slug: string, stageId: number, request: UpdateStageSeedsRequest): Observable<StageSeedAssignment[]> {
    return this.http.put<StageSeedAssignment[]>(`${this.baseUrl}/${slug}/stages/${stageId}/seeds`, request);
  }

  // Stage participant pool methods (for assigning participants to start in specific stages)

  getStagePool(slug: string): Observable<StagePoolResponse> {
    return this.http.get<StagePoolResponse>(`${this.baseUrl}/${slug}/stages/pool`);
  }

  updateStagePool(slug: string, request: UpdateStagePoolRequest): Observable<StagePoolResponse> {
    return this.http.put<StagePoolResponse>(`${this.baseUrl}/${slug}/stages/pool`, request);
  }

  // Logo upload
  getLogoUploadUrl(slug: string, contentType: string): Observable<{ upload_url: string; logo_url: string; expires_at: string }> {
    return this.http.post<{ upload_url: string; logo_url: string; expires_at: string }>(
      `${this.baseUrl}/${slug}/logo/upload-url`,
      { content_type: contentType }
    );
  }

  updateLogoUrl(slug: string, logoUrl: string): Observable<{ logo_url: string }> {
    return this.http.put<{ logo_url: string }>(`${this.baseUrl}/${slug}/logo`, { logo_url: logoUrl });
  }

  // Prize distribution methods
  getSuggestedPrizeTiers(slug: string, participants?: number): Observable<SuggestedPrizeTiersResponse> {
    const options: { params?: { participants: string } } = {};
    if (participants) {
      options.params = { participants: participants.toString() };
    }
    return this.http.get<SuggestedPrizeTiersResponse>(`${this.baseUrl}/${slug}/prize-tiers`, options);
  }
}
