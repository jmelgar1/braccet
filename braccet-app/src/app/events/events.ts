import { Component, inject, signal, OnInit } from '@angular/core';
import { DatePipe } from '@angular/common';
import { Router } from '@angular/router';
import { EventService } from '../services/event.service';
import { AuthService } from '../services/auth.service';
import { Event } from '../models/event.model';

@Component({
  selector: 'app-events',
  imports: [DatePipe],
  templateUrl: './events.html',
  styleUrl: './events.css'
})
export class Events implements OnInit {
  private eventService = inject(EventService);
  private authService = inject(AuthService);
  private router = inject(Router);

  events = signal<Event[]>([]);
  loading = signal(true);
  error = signal('');

  ngOnInit(): void {
    this.loadEvents();
  }

  loadEvents(): void {
    if (!this.authService.isLoggedIn()) {
      this.events.set([]);
      this.loading.set(false);
      return;
    }

    this.loading.set(true);
    this.error.set('');

    this.eventService.getEvents().subscribe({
      next: (events) => {
        this.events.set(events || []);
        this.loading.set(false);
      },
      error: (err) => {
        if (err.status === 401 || err.status === 403) {
          this.events.set([]);
          this.loading.set(false);
          return;
        }
        this.error.set(err.error?.error || 'Failed to load events');
        this.loading.set(false);
      }
    });
  }

  onCreateEvent(): void {
    this.router.navigate(['/events/new']);
  }

  onEventClick(slug: string): void {
    this.router.navigate(['/events', slug]);
  }

  deleteEvent(slug: string, name: string, event: MouseEvent): void {
    event.stopPropagation();

    if (!confirm(`Are you sure you want to delete "${name}"? This cannot be undone.`)) {
      return;
    }

    this.eventService.deleteEvent(slug).subscribe({
      next: () => {
        this.events.update(events =>
          events.filter(e => e.slug !== slug)
        );
      },
      error: (err) => {
        this.error.set(err.error?.error || 'Failed to delete event');
      }
    });
  }

  getStatusClass(status: string): string {
    switch (status) {
      case 'draft':
        return 'bg-gray-100 text-gray-600';
      case 'registration':
        return 'bg-blue-100 text-blue-600';
      case 'in_progress':
        return 'bg-green-100 text-green-600';
      case 'completed':
        return 'bg-purple-100 text-purple-600';
      case 'cancelled':
        return 'bg-red-100 text-red-600';
      default:
        return 'bg-gray-100 text-gray-600';
    }
  }

  formatStatus(status: string): string {
    return status.replace('_', ' ');
  }
}
