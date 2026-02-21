import { Injectable, inject, signal, computed } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, tap, of, catchError } from 'rxjs';
import { environment } from '../../environments/environment';

export interface User {
  id: number;
  email: string;
  username?: string;
  display_name: string;
  avatar_url?: string;
}

export interface AuthResponse {
  access_token: string;
  refresh_token: string;
  user: User;
}

export interface RefreshResponse {
  access_token: string;
  refresh_token: string;
}

export interface SignupRequest {
  email: string;
  username: string;
  password: string;
  display_name?: string;
}

export interface LoginRequest {
  identifier: string;
  password: string;
}

@Injectable({ providedIn: 'root' })
export class AuthService {
  private http = inject(HttpClient);
  private currentUser = signal<User | null>(null);

  private readonly ACCESS_TOKEN_KEY = 'access_token';
  private readonly REFRESH_TOKEN_KEY = 'refresh_token';

  isLoggedIn = computed(() => this.currentUser() !== null);
  user = computed(() => this.currentUser());

  loginWithGoogle(): void {
    window.location.href = `${environment.apiUrl}/auth/google`;
  }

  loginWithDiscord(): void {
    window.location.href = `${environment.apiUrl}/auth/discord`;
  }

  handleCallback(accessToken: string, refreshToken: string): void {
    this.storeTokens(accessToken, refreshToken);
    this.loadCurrentUser();
  }

  loadCurrentUser(): Observable<User | null> {
    const token = this.getAccessToken();
    if (!token) {
      return of(null);
    }

    return this.http.get<User>(`${environment.apiUrl}/auth/me`).pipe(
      tap(user => this.currentUser.set(user)),
      catchError(() => {
        this.clearTokens();
        return of(null);
      })
    );
  }

  logout(): void {
    this.clearTokens();
    this.currentUser.set(null);
  }

  getAccessToken(): string | null {
    return localStorage.getItem(this.ACCESS_TOKEN_KEY);
  }

  getRefreshToken(): string | null {
    return localStorage.getItem(this.REFRESH_TOKEN_KEY);
  }

  refreshAccessToken(): Observable<RefreshResponse> {
    const refreshToken = this.getRefreshToken();
    return this.http.post<RefreshResponse>(`${environment.apiUrl}/auth/refresh`, {
      refresh_token: refreshToken
    }).pipe(
      tap(response => this.storeTokens(response.access_token, response.refresh_token))
    );
  }

  signup(request: SignupRequest): Observable<AuthResponse> {
    return this.http.post<AuthResponse>(`${environment.apiUrl}/auth/signup`, request)
      .pipe(
        tap(response => {
          this.storeTokens(response.access_token, response.refresh_token);
          this.currentUser.set(response.user);
        })
      );
  }

  login(identifier: string, password: string): Observable<AuthResponse> {
    return this.http.post<AuthResponse>(`${environment.apiUrl}/auth/login`, { identifier, password })
      .pipe(
        tap(response => {
          this.storeTokens(response.access_token, response.refresh_token);
          this.currentUser.set(response.user);
        })
      );
  }

  private storeTokens(accessToken: string, refreshToken: string): void {
    localStorage.setItem(this.ACCESS_TOKEN_KEY, accessToken);
    localStorage.setItem(this.REFRESH_TOKEN_KEY, refreshToken);
  }

  clearTokens(): void {
    localStorage.removeItem(this.ACCESS_TOKEN_KEY);
    localStorage.removeItem(this.REFRESH_TOKEN_KEY);
  }
}
