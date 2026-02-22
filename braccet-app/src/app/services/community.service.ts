import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../environments/environment';
import { Community, CreateCommunityRequest, CommunityMember, AddMemberRequest, MemberRole } from '../models/community.model';

@Injectable({ providedIn: 'root' })
export class CommunityService {
  private http = inject(HttpClient);
  private baseUrl = `${environment.apiUrl}/communities`;

  getCommunities(): Observable<Community[]> {
    return this.http.get<Community[]>(this.baseUrl);
  }

  getCommunity(slug: string): Observable<Community> {
    return this.http.get<Community>(`${this.baseUrl}/${slug}`);
  }

  getCommunityById(id: number): Observable<Community> {
    return this.http.get<Community>(`${this.baseUrl}/internal/${id}`);
  }

  createCommunity(request: CreateCommunityRequest): Observable<Community> {
    return this.http.post<Community>(this.baseUrl, request);
  }

  updateCommunity(slug: string, request: Partial<Community>): Observable<Community> {
    return this.http.put<Community>(`${this.baseUrl}/${slug}`, request);
  }

  deleteCommunity(slug: string): Observable<void> {
    return this.http.delete<void>(`${this.baseUrl}/${slug}`);
  }

  // Member methods
  getMembers(slug: string): Observable<CommunityMember[]> {
    return this.http.get<CommunityMember[]>(`${this.baseUrl}/${slug}/members`);
  }

  addMember(slug: string, request: AddMemberRequest): Observable<CommunityMember> {
    return this.http.post<CommunityMember>(`${this.baseUrl}/${slug}/members`, request);
  }

  removeMember(slug: string, memberId: number): Observable<void> {
    return this.http.delete<void>(`${this.baseUrl}/${slug}/members/${memberId}`);
  }

  updateMemberRole(slug: string, memberId: number, role: MemberRole): Observable<CommunityMember> {
    return this.http.put<CommunityMember>(`${this.baseUrl}/${slug}/members/${memberId}/role`, { role });
  }
}
