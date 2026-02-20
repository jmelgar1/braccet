import { Component, inject, signal, HostListener } from '@angular/core';
import { Router, RouterLink, NavigationEnd } from '@angular/router';
import { AuthService } from '../../services/auth.service';
import { filter } from 'rxjs/operators';

@Component({
  selector: 'app-header',
  imports: [RouterLink],
  templateUrl: './header.html',
  styleUrl: './header.css'
})
export class Header {
  private authService = inject(AuthService);
  private router = inject(Router);

  dropdownOpen = signal(false);
  showHeader = signal(true);

  private authRoutes = ['/login', '/signup'];

  constructor() {
    this.checkRoute(this.router.url);

    this.router.events.pipe(
      filter((event): event is NavigationEnd => event instanceof NavigationEnd)
    ).subscribe((event) => {
      this.checkRoute(event.urlAfterRedirects);
      this.dropdownOpen.set(false);
    });
  }

  private checkRoute(url: string): void {
    this.showHeader.set(!this.authRoutes.includes(url));
  }

  toggleDropdown(): void {
    this.dropdownOpen.update(open => !open);
  }

  @HostListener('document:click', ['$event'])
  onDocumentClick(event: MouseEvent): void {
    const target = event.target as HTMLElement;
    if (!target.closest('.profile-dropdown')) {
      this.dropdownOpen.set(false);
    }
  }

  onSettings(): void {
    this.dropdownOpen.set(false);
  }

  onLogout(): void {
    this.authService.logout();
    this.router.navigate(['/login']);
    this.dropdownOpen.set(false);
  }
}
