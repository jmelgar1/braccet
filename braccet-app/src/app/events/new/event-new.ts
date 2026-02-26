import { Component, signal, computed, inject, OnInit } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { Breadcrumb, BreadcrumbItem } from '../../components/breadcrumb/breadcrumb';
import { EventService } from '../../services/event.service';
import { CommunityService } from '../../services/community.service';
import { CreateEventRequest } from '../../models/event.model';
import { Community } from '../../models/community.model';

@Component({
  selector: 'app-event-new',
  imports: [Breadcrumb, FormsModule],
  templateUrl: './event-new.html',
  styleUrl: './event-new.css'
})
export class EventNew implements OnInit {
  private router = inject(Router);
  private eventService = inject(EventService);
  private communityService = inject(CommunityService);

  breadcrumbs: BreadcrumbItem[] = [
    { label: 'Events', route: '/events' },
    { label: 'New Event' }
  ];

  // Communities
  communities = signal<Community[]>([]);
  loadingCommunities = signal(false);

  // Form fields
  name = signal('');
  game = signal('');
  description = signal('');
  startsAt = signal('');
  communityId = signal<number | null>(null);

  // Touched state for validation
  nameTouched = signal(false);
  communityTouched = signal(false);

  // Form state
  loading = signal(false);
  error = signal('');

  // Validation
  nameError = computed(() => {
    if (!this.nameTouched()) return '';
    if (!this.name().trim()) return 'Event name is required';
    if (this.name().length > 200) return 'Name must be 200 characters or less';
    return '';
  });

  communityError = computed(() => {
    if (!this.communityTouched()) return '';
    if (!this.communityId()) return 'Community is required';
    return '';
  });

  isValid = computed(() => {
    return this.name().trim().length > 0 &&
           this.name().length <= 200 &&
           this.communityId() !== null;
  });

  ngOnInit(): void {
    this.loadCommunities();
  }

  loadCommunities(): void {
    this.loadingCommunities.set(true);
    this.communityService.getCommunities().subscribe({
      next: (communities) => {
        this.communities.set(communities || []);
        this.loadingCommunities.set(false);
      },
      error: () => {
        this.communities.set([]);
        this.loadingCommunities.set(false);
      }
    });
  }

  onSubmit() {
    this.nameTouched.set(true);
    this.communityTouched.set(true);

    if (!this.isValid()) {
      return;
    }

    this.loading.set(true);
    this.error.set('');

    const request: CreateEventRequest = {
      community_id: this.communityId()!,
      name: this.name().trim(),
    };

    if (this.description().trim()) {
      request.description = this.description().trim();
    }
    if (this.game().trim()) {
      request.game = this.game().trim();
    }
    if (this.startsAt()) {
      request.starts_at = new Date(this.startsAt()).toISOString();
    }

    this.eventService.createEvent(request).subscribe({
      next: (event) => {
        this.router.navigate(['/events', event.slug]);
      },
      error: (err) => {
        this.loading.set(false);
        this.error.set(err.error?.error || 'Failed to create event');
      }
    });
  }
}
