import { Component, signal, computed, inject, OnInit, effect } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { Breadcrumb, BreadcrumbItem } from '../../components/breadcrumb/breadcrumb';
import { TournamentService } from '../../services/tournament.service';
import { CommunityService } from '../../services/community.service';
import { EloService } from '../../services/elo.service';
import { CreateTournamentRequest } from '../../models/tournament.model';
import { Community } from '../../models/community.model';
import { EloSystem } from '../../models/elo.model';

@Component({
  selector: 'app-tournament-new',
  imports: [Breadcrumb, FormsModule],
  templateUrl: './tournament-new.html',
  styleUrl: './tournament-new.css'
})
export class TournamentNew implements OnInit {
  private router = inject(Router);
  private tournamentService = inject(TournamentService);
  private communityService = inject(CommunityService);
  private eloService = inject(EloService);

  breadcrumbs: BreadcrumbItem[] = [
    { label: 'Tournaments', route: '/tournaments' },
    { label: 'New Tournament' }
  ];

  // Communities
  communities = signal<Community[]>([]);
  loadingCommunities = signal(false);

  // ELO Systems (for selected community)
  eloSystems = signal<EloSystem[]>([]);
  loadingEloSystems = signal(false);

  // Form fields
  name = signal('');
  game = signal('');
  description = signal('');
  format = signal<'single_elimination' | 'double_elimination'>('single_elimination');
  maxParticipants = signal<number | null>(null);
  startsAt = signal('');
  startsAtTentative = signal(false);
  communityId = signal<number | null>(null);
  eloSystemId = signal<number | null>(null);

  constructor() {
    // Load ELO systems when community changes
    effect(() => {
      const cid = this.communityId();
      if (cid) {
        this.loadEloSystems(cid);
      } else {
        this.eloSystems.set([]);
        this.eloSystemId.set(null);
      }
    });
  }

  // Touched state for validation
  nameTouched = signal(false);

  // Form state
  loading = signal(false);
  error = signal('');

  // Validation
  nameError = computed(() => {
    if (!this.nameTouched()) return '';
    if (!this.name().trim()) return 'Tournament name is required';
    if (this.name().length > 200) return 'Name must be 200 characters or less';
    return '';
  });

  isValid = computed(() => {
    return this.name().trim().length > 0 && this.name().length <= 200;
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

  loadEloSystems(communityId: number): void {
    const community = this.communities().find(c => c.id === communityId);
    if (!community) return;

    this.loadingEloSystems.set(true);
    this.eloSystemId.set(null);

    this.eloService.getEloSystems(community.slug).subscribe({
      next: (systems) => {
        this.eloSystems.set(systems || []);
        // Auto-select the default system if one exists
        const defaultSystem = systems?.find(s => s.is_default);
        if (defaultSystem) {
          this.eloSystemId.set(defaultSystem.id);
        }
        this.loadingEloSystems.set(false);
      },
      error: () => {
        this.eloSystems.set([]);
        this.loadingEloSystems.set(false);
      }
    });
  }

  onSubmit() {
    this.nameTouched.set(true);

    if (!this.isValid()) {
      return;
    }

    this.loading.set(true);
    this.error.set('');

    const request: CreateTournamentRequest = {
      name: this.name().trim(),
      format: this.format(),
    };

    if (this.description().trim()) {
      request.description = this.description().trim();
    }
    if (this.game().trim()) {
      request.game = this.game().trim();
    }
    if (this.maxParticipants()) {
      request.max_participants = this.maxParticipants()!;
    }
    if (this.startsAt()) {
      request.starts_at = new Date(this.startsAt()).toISOString();
      request.starts_at_tentative = this.startsAtTentative();
    }
    if (this.communityId()) {
      request.community_id = this.communityId()!;
    }
    if (this.eloSystemId()) {
      request.elo_system_id = this.eloSystemId()!;
    }

    this.tournamentService.createTournament(request).subscribe({
      next: (tournament) => {
        this.router.navigate(['/tournaments', tournament.slug]);
      },
      error: (err) => {
        this.loading.set(false);
        this.error.set(err.error?.message || 'Failed to create tournament');
      }
    });
  }
}
