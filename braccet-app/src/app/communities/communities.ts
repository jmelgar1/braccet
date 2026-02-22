import { Component, inject, signal, OnInit } from '@angular/core';
import { DatePipe } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { CommunityService } from '../services/community.service';
import { AuthService } from '../services/auth.service';
import { Community, CreateCommunityRequest } from '../models/community.model';

@Component({
  selector: 'app-communities',
  imports: [DatePipe, FormsModule],
  templateUrl: './communities.html',
  styleUrl: './communities.css'
})
export class Communities implements OnInit {
  private communityService = inject(CommunityService);
  private authService = inject(AuthService);
  private router = inject(Router);

  communities = signal<Community[]>([]);
  loading = signal(true);
  error = signal('');

  // Create form state
  showCreateForm = signal(false);
  creating = signal(false);
  createError = signal('');
  newCommunity: CreateCommunityRequest = {
    name: '',
    description: '',
    game: ''
  };

  ngOnInit(): void {
    this.loadCommunities();
  }

  loadCommunities(): void {
    if (!this.authService.isLoggedIn()) {
      this.communities.set([]);
      this.loading.set(false);
      return;
    }

    this.loading.set(true);
    this.error.set('');

    this.communityService.getCommunities().subscribe({
      next: (communities) => {
        this.communities.set(communities || []);
        this.loading.set(false);
      },
      error: (err) => {
        if (err.status === 401 || err.status === 403) {
          this.communities.set([]);
          this.loading.set(false);
          return;
        }
        this.error.set(err.error?.error || 'Failed to load communities');
        this.loading.set(false);
      }
    });
  }

  toggleCreateForm(): void {
    this.showCreateForm.update(v => !v);
    if (!this.showCreateForm()) {
      this.resetCreateForm();
    }
  }

  resetCreateForm(): void {
    this.newCommunity = { name: '', description: '', game: '' };
    this.createError.set('');
  }

  onCreateCommunity(): void {
    if (!this.newCommunity.name.trim()) {
      this.createError.set('Name is required');
      return;
    }

    this.creating.set(true);
    this.createError.set('');

    const request: CreateCommunityRequest = {
      name: this.newCommunity.name.trim(),
      description: this.newCommunity.description?.trim() || undefined,
      game: this.newCommunity.game?.trim() || undefined
    };

    this.communityService.createCommunity(request).subscribe({
      next: (community) => {
        this.communities.update(list => [community, ...list]);
        this.showCreateForm.set(false);
        this.resetCreateForm();
        this.creating.set(false);
      },
      error: (err) => {
        this.createError.set(err.error?.error || 'Failed to create community');
        this.creating.set(false);
      }
    });
  }

  onCommunityClick(slug: string): void {
    this.router.navigate(['/communities', slug]);
  }

  deleteCommunity(slug: string, name: string, event: Event): void {
    event.stopPropagation();

    if (!confirm(`Are you sure you want to delete "${name}"? This cannot be undone.`)) {
      return;
    }

    this.communityService.deleteCommunity(slug).subscribe({
      next: () => {
        this.communities.update(communities =>
          communities.filter(c => c.slug !== slug)
        );
      },
      error: (err) => {
        this.error.set(err.error?.error || 'Failed to delete community');
      }
    });
  }
}
