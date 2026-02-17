import { Injectable, inject, signal, computed } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, tap } from 'rxjs';
import { environment } from '../../environments/environment';

export interface User {
  id: number;
  email: string;
  username?: string;
  display_name: string;
  avatar_url?: string;
}

export interface AuthResponse {
  token: string;
  user: User;
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

  isLoggedIn = computed(() => this.currentUser() !== null);
  user = computed(() => this.currentUser());

  loginWithGoogle(): void {
    window.location.href = `${environment.apiUrl}/auth/google`;
  }

  loginWithDiscord(): void {
    window.location.href = `${environment.apiUrl}/auth/discord`;
  }

  handleCallback(token: string): void {
    localStorage.setItem('token', token);
    this.loadCurrentUser();
  }

  loadCurrentUser(): void {
    const token = localStorage.getItem('token');
    if (!token) return;

    this.http.get<User>(`${environment.apiUrl}/auth/me`)
      .subscribe({
        next: (user) => this.currentUser.set(user),
        error: () => {
          localStorage.removeItem('token');
          this.currentUser.set(null);
        }
      });
  }

  logout(): void {
    localStorage.removeItem('token');
    this.currentUser.set(null);
  }

  getToken(): string | null {
    return localStorage.getItem('token');
  }

  signup(request: SignupRequest): Observable<AuthResponse> {
    return this.http.post<AuthResponse>(`${environment.apiUrl}/auth/signup`, request)
      .pipe(
        tap(response => {
          localStorage.setItem('token', response.token);
          this.currentUser.set(response.user);
        })
      );
  }

  login(identifier: string, password: string): Observable<AuthResponse> {
    return this.http.post<AuthResponse>(`${environment.apiUrl}/auth/login`, { identifier, password })
      .pipe(
        tap(response => {
          localStorage.setItem('token', response.token);
          this.currentUser.set(response.user);
        })
      );
  }
}
