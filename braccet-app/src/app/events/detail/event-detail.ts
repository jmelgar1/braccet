import { Component, inject, signal, computed, OnInit, HostListener } from '@angular/core';
import { DatePipe, TitleCasePipe } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { EventService } from '../../services/event.service';
import { AuthService } from '../../services/auth.service';
import { UploadService } from '../../services/upload.service';
import { Event as EventModel, EventTournament, AddTournamentToEventRequest, EventGraphResponse, ConnectionUpdate } from '../../models/event.model';
import { Tournament } from '../../models/tournament.model';
import { Breadcrumb, BreadcrumbItem } from '../../components/breadcrumb/breadcrumb';
import { QualifierDagEditor } from '../../components/qualifier-dag-editor/qualifier-dag-editor';

@Component({
  selector: 'app-event-detail',
  imports: [DatePipe, TitleCasePipe, Breadcrumb, FormsModule, QualifierDagEditor],
  templateUrl: './event-detail.html',
  styleUrl: './event-detail.css'
})
export class EventDetail implements OnInit {
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private eventService = inject(EventService);
  private authService = inject(AuthService);
  private uploadService = inject(UploadService);

  event = signal<EventModel | null>(null);
  loading = signal(true);
  error = signal('');

  // Add tournament dropdown state
  addTournamentDropdownOpen = signal(false);

  // Select tournament dialog state
  showSelectTournamentDialog = signal(false);
  availableTournaments = signal<Tournament[]>([]);
  loadingTournaments = signal(false);
  addingTournament = signal(false);

  // Logo upload state
  uploadingLogo = signal(false);

  // DAG editor state
  viewMode = signal<'list' | 'dag'>('list');
  eventGraph = signal<EventGraphResponse | null>(null);
  loadingGraph = signal(false);

  // Computed properties
  isOrganizer = computed(() => {
    const e = this.event();
    const user = this.authService.user();
    return e && user ? e.organizer_id === user.id : false;
  });

  // Check if a main event tournament already exists
  hasMainEvent = computed(() => {
    const e = this.event();
    return e?.tournaments?.some(t => t.role === 'main') ?? false;
  });

  breadcrumbs: BreadcrumbItem[] = [
    { label: 'Events', route: '/events' },
    { label: 'Loading...' }
  ];

  ngOnInit(): void {
    const slug = this.route.snapshot.paramMap.get('slug');
    if (slug) {
      // Check for addTournament query param (returning from tournament creation)
      const addTournamentId = this.route.snapshot.queryParamMap.get('addTournament');

      this.loadEvent(slug, addTournamentId ? parseInt(addTournamentId, 10) : undefined);

      // Clear query param from URL
      if (addTournamentId) {
        this.router.navigate([], { relativeTo: this.route, queryParams: {}, replaceUrl: true });
      }
    } else {
      this.error.set('Event not found');
      this.loading.set(false);
    }
  }

  loadEvent(slug: string, pendingTournamentId?: number): void {
    this.loading.set(true);
    this.error.set('');

    this.eventService.getEvent(slug).subscribe({
      next: (event) => {
        this.event.set(event);
        this.breadcrumbs = [
          { label: 'Events', route: '/events' },
          { label: event.name }
        ];
        this.loading.set(false);

        // If returning from tournament creation, add it now
        if (pendingTournamentId) {
          this.handleAddTournamentReturn(pendingTournamentId);
        }
      },
      error: (err) => {
        this.error.set(err.error?.error || 'Failed to load event');
        this.loading.set(false);
      }
    });
  }

  getStatusLabel(status: string): string {
    const labels: Record<string, string> = {
      draft: 'Draft',
      registration: 'Registration Open',
      in_progress: 'In Progress',
      completed: 'Completed',
      cancelled: 'Cancelled'
    };
    return labels[status] || status;
  }

  getStatusColor(status: string): string {
    const colors: Record<string, string> = {
      draft: 'bg-gray-100 text-gray-600',
      registration: 'bg-blue-100 text-blue-600',
      in_progress: 'bg-green-100 text-green-600',
      completed: 'bg-purple-100 text-purple-600',
      cancelled: 'bg-red-100 text-red-600'
    };
    return colors[status] || 'bg-gray-100 text-gray-600';
  }

  getRoleLabel(role: string): string {
    return role === 'qualifier' ? 'Qualifier' : 'Main Event';
  }

  getRoleColor(role: string): string {
    return role === 'qualifier'
      ? 'bg-yellow-100 text-yellow-700'
      : 'bg-purple-100 text-purple-700';
  }

  getTournamentStatusColor(status: string): string {
    const colors: Record<string, string> = {
      registration: 'text-blue-600',
      in_progress: 'text-green-600',
      completed: 'text-purple-600',
      cancelled: 'text-red-600'
    };
    return colors[status] || 'text-gray-600';
  }

  onTournamentClick(tournament: EventTournament): void {
    this.router.navigate(['/tournaments', tournament.tournament_slug]);
  }

  onEditEvent(): void {
    // TODO: Implement edit functionality
  }

  deleteEvent(): void {
    const e = this.event();
    if (!e) return;

    if (!confirm(`Are you sure you want to delete "${e.name}"? This cannot be undone.`)) {
      return;
    }

    this.eventService.deleteEvent(e.slug).subscribe({
      next: () => {
        this.router.navigate(['/events']);
      },
      error: (err) => {
        this.error.set(err.error?.error || 'Failed to delete event');
      }
    });
  }

  // Add tournament dropdown methods
  @HostListener('document:click', ['$event'])
  onDocumentClick(event: MouseEvent): void {
    const target = event.target as HTMLElement;
    if (!target.closest('.add-tournament-dropdown')) {
      this.addTournamentDropdownOpen.set(false);
    }
  }

  toggleAddTournamentDropdown(): void {
    this.addTournamentDropdownOpen.update(open => !open);
  }

  onExistingTournament(): void {
    this.addTournamentDropdownOpen.set(false);
    this.loadAvailableTournaments();
    this.showSelectTournamentDialog.set(true);
  }

  onNewTournament(): void {
    this.addTournamentDropdownOpen.set(false);
    const e = this.event();
    if (e) {
      this.router.navigate(['/tournaments/new'], { queryParams: { event: e.slug } });
    }
  }

  loadAvailableTournaments(): void {
    const e = this.event();
    if (!e) return;

    this.loadingTournaments.set(true);
    this.eventService.getAvailableTournaments(e.slug).subscribe({
      next: (tournaments) => {
        this.availableTournaments.set(tournaments);
        this.loadingTournaments.set(false);
      },
      error: () => {
        this.availableTournaments.set([]);
        this.loadingTournaments.set(false);
      }
    });
  }

  onSelectTournament(tournament: Tournament): void {
    this.addTournamentToEvent(tournament.id);
  }

  closeSelectTournamentDialog(): void {
    this.showSelectTournamentDialog.set(false);
  }

  private addTournamentToEvent(tournamentId: number): void {
    const e = this.event();
    if (!e) return;

    const request: AddTournamentToEventRequest = {
      tournament_id: tournamentId,
      role: this.hasMainEvent() ? 'qualifier' : 'main'
    };

    this.addingTournament.set(true);
    this.eventService.addTournament(e.slug, request).subscribe({
      next: () => {
        this.showSelectTournamentDialog.set(false);
        this.addingTournament.set(false);
        this.loadEvent(e.slug); // Refresh
      },
      error: (err) => {
        this.error.set(err.error?.error || 'Failed to add tournament');
        this.addingTournament.set(false);
      }
    });
  }

  // Handle returning from tournament creation page
  private handleAddTournamentReturn(tournamentId: number): void {
    this.addTournamentToEvent(tournamentId);
  }

  getFormatLabel(format: string): string {
    const labels: Record<string, string> = {
      single_elimination: 'Single Elim',
      double_elimination: 'Double Elim',
      swiss: 'Swiss',
      multi_stage: 'Multi-Stage'
    };
    return labels[format] || format;
  }

  updateAdvancingCount(tournament: EventTournament, value: string): void {
    const e = this.event();
    if (!e) return;

    const advancingCount = value ? parseInt(value, 10) : null;

    this.eventService.updateTournament(e.slug, tournament.tournament_id, {
      advancing_count: advancingCount ?? undefined
    }).subscribe({
      next: () => {
        // Update local state
        const tournaments = e.tournaments?.map(t =>
          t.id === tournament.id ? { ...t, advancing_count: advancingCount ?? undefined } : t
        );
        this.event.set({ ...e, tournaments });
      },
      error: (err) => {
        this.error.set(err.error?.error || 'Failed to update advancing count');
      }
    });
  }

  onAdvancingInputClick(event: Event): void {
    event.stopPropagation(); // Prevent navigation when clicking the input
  }

  onLogoFileSelected(event: Event): void {
    const input = event.target as HTMLInputElement;
    if (!input.files || input.files.length === 0) return;

    const file = input.files[0];
    const e = this.event();
    if (!e) return;

    // DEBUG: Log file info
    console.log('File info:', { name: file.name, type: file.type, size: file.size });

    // Validate file
    const validation = this.uploadService.validateImageFile(file);
    if (!validation.valid) {
      this.error.set(validation.error || 'Invalid file');
      return;
    }

    this.uploadLogo(e.slug, file);
  }

  private uploadLogo(eventSlug: string, file: File): void {
    this.uploadingLogo.set(true);
    this.error.set('');

    // Step 1: Get presigned upload URL
    this.eventService.getLogoUploadUrl(eventSlug, file.type).subscribe({
      next: (response) => {
        // Step 2: Upload file directly to presigned URL
        this.uploadService.uploadToPresignedUrl(response.upload_url, file).subscribe({
          next: () => {
            // Step 3: Update event with new logo URL
            this.eventService.updateLogoUrl(eventSlug, response.logo_url).subscribe({
              next: () => {
                // Update local state
                const e = this.event();
                if (e) {
                  this.event.set({ ...e, logo_url: response.logo_url });
                }
                this.uploadingLogo.set(false);
              },
              error: (err) => {
                this.error.set(err.error?.error || 'Failed to update logo URL');
                this.uploadingLogo.set(false);
              }
            });
          },
          error: () => {
            this.error.set('Failed to upload logo');
            this.uploadingLogo.set(false);
          }
        });
      },
      error: (err) => {
        this.error.set(err.error?.error || 'Failed to get upload URL');
        this.uploadingLogo.set(false);
      }
    });
  }

  // DAG Editor methods
  setViewMode(mode: 'list' | 'dag'): void {
    this.viewMode.set(mode);
    if (mode === 'dag' && !this.eventGraph()) {
      this.loadEventGraph();
    }
  }

  loadEventGraph(): void {
    const e = this.event();
    if (!e) return;

    this.loadingGraph.set(true);
    this.eventService.getEventGraph(e.slug).subscribe({
      next: (graph) => {
        this.eventGraph.set(graph);
        this.loadingGraph.set(false);
      },
      error: (err) => {
        this.error.set(err.error?.error || 'Failed to load graph');
        this.loadingGraph.set(false);
      }
    });
  }

  onConnectionCreated(event: { sourceId: number; targetId: number }): void {
    const e = this.event();
    if (!e) return;

    // Find the source tournament to get main event as target
    const mainEvent = e.tournaments?.find(t => t.role === 'main');
    if (!mainEvent) return;

    // Update the source tournament to point to main event
    this.eventService.updateTournament(e.slug, event.sourceId, {
      target_tournament_id: event.targetId,
      advancing_count: 8 // Default advancing count
    }).subscribe({
      next: () => {
        this.loadEventGraph(); // Refresh graph
        this.loadEvent(e.slug); // Refresh event data
      },
      error: (err) => {
        this.error.set(err.error?.error || 'Failed to create connection');
      }
    });
  }

  onConnectionRemoved(event: { sourceId: number; targetId: number }): void {
    const e = this.event();
    if (!e) return;

    // Remove connection by clearing target and advancing count
    this.eventService.updateTournament(e.slug, event.sourceId, {
      target_tournament_id: undefined,
      advancing_count: undefined
    }).subscribe({
      next: () => {
        this.loadEventGraph();
        this.loadEvent(e.slug);
      },
      error: (err) => {
        this.error.set(err.error?.error || 'Failed to remove connection');
      }
    });
  }

  onAdvancingCountChanged(event: { sourceId: number; count: number }): void {
    const e = this.event();
    if (!e) return;

    this.eventService.updateTournament(e.slug, event.sourceId, {
      advancing_count: event.count
    }).subscribe({
      next: () => {
        this.loadEventGraph();
        this.loadEvent(e.slug);
      },
      error: (err) => {
        this.error.set(err.error?.error || 'Failed to update advancing count');
      }
    });
  }

  onPositionsChanged(positions: { tournamentId: number; x: number; y: number }[]): void {
    const e = this.event();
    if (!e) return;

    const graph = this.eventGraph();
    if (!graph) return;

    // Build connections array for batch update
    const connections: ConnectionUpdate[] = graph.nodes.map(node => {
      const pos = positions.find(p => p.tournamentId === node.tournament_id);
      const edge = graph.edges.find(edge => edge.source_tournament_id === node.tournament_id);

      return {
        tournament_id: node.tournament_id,
        target_tournament_id: edge?.target_tournament_id,
        advancing_count: edge?.advancing_count,
        position_x: pos?.x ?? node.position_x,
        position_y: pos?.y ?? node.position_y
      };
    });

    this.eventService.updateConnections(e.slug, { connections }).subscribe({
      error: (err) => {
        this.error.set(err.error?.error || 'Failed to save positions');
      }
    });
  }
}
