import { Component, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';
import { AuthService } from '../services/auth.service';

@Component({
  selector: 'app-signup',
  imports: [FormsModule, RouterLink],
  templateUrl: './signup.html',
  styleUrl: './signup.css'
})
export class Signup {
  private authService = inject(AuthService);
  private router = inject(Router);

  // Form fields
  email = signal('');
  username = signal('');
  password = signal('');
  confirmPassword = signal('');

  // UI state
  loading = signal(false);
  error = signal('');

  loginWithGoogle(): void {
    this.authService.loginWithGoogle();
  }

  loginWithDiscord(): void {
    this.authService.loginWithDiscord();
  }

  onSignup(): void {
    this.error.set('');

    if (!this.email() || !this.username() || !this.password() || !this.confirmPassword()) {
      this.error.set('Please fill in all fields');
      return;
    }

    if (this.password() !== this.confirmPassword()) {
      this.error.set('Passwords do not match');
      return;
    }

    if (this.password().length < 8) {
      this.error.set('Password must be at least 8 characters');
      return;
    }

    this.loading.set(true);

    this.authService.signup({
      email: this.email(),
      username: this.username(),
      password: this.password()
    }).subscribe({
      next: () => {
        this.loading.set(false);
        this.router.navigate(['/']);
      },
      error: (err) => {
        this.loading.set(false);
        this.error.set(err.error?.error || 'Signup failed');
      }
    });
  }
}
