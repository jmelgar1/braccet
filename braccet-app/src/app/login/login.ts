import { Component, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';
import { AuthService } from '../services/auth.service';

@Component({
  selector: 'app-login',
  imports: [FormsModule, RouterLink],
  templateUrl: './login.html',
  styleUrl: './login.css'
})
export class Login {
  private authService = inject(AuthService);
  private router = inject(Router);

  // Form fields
  identifier = signal('');
  password = signal('');

  // UI state
  loading = signal(false);
  error = signal('');

  loginWithGoogle(): void {
    this.authService.loginWithGoogle();
  }

  loginWithDiscord(): void {
    this.authService.loginWithDiscord();
  }

  onLogin(): void {
    this.error.set('');

    if (!this.identifier() || !this.password()) {
      this.error.set('Please fill in all fields');
      return;
    }

    this.loading.set(true);

    this.authService.login(this.identifier(), this.password()).subscribe({
      next: () => {
        this.loading.set(false);
        this.router.navigate(['/']);
      },
      error: (err) => {
        this.loading.set(false);
        this.error.set(err.error?.error || 'Login failed');
      }
    });
  }
}
