import { Component, inject, signal, computed, OnInit, OnDestroy } from '@angular/core';
import { DatePipe } from '@angular/common';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { switchMap } from 'rxjs';
import { TournamentService } from '../../services/tournament.service';
import { BracketService } from '../../services/bracket.service';
import { AuthService } from '../../services/auth.service';
import { CommunityService } from '../../services/community.service';
import { EloService } from '../../services/elo.service';
import { UploadService } from '../../services/upload.service';
import { TournamentUIService } from '../../services/tournament-ui.service';
import { Tournament, Participant, TournamentStage, StagePoolEntry } from '../../models/tournament.model';
import { CreateBracketRequest } from '../../models/bracket.model';
import { Community } from '../../models/community.model';
import { EloSystem } from '../../models/elo.model';
import { Breadcrumb, BreadcrumbItem } from '../../components/breadcrumb/breadcrumb';
import { SidePanel } from './components/side-panel/side-panel';
import { StageSeedPopover } from '../../components/stage-seed-popover/stage-seed-popover';

@Component({
  selector: 'app-tournament-detail',
  imports: [DatePipe, Breadcrumb, SidePanel, RouterLink, StageSeedPopover],
  templateUrl: './tournament-detail.html',
  styleUrl: './tournament-detail.css'
})
export class TournamentDetail implements OnInit, OnDestroy {
  private route = inject(ActivatedRoute);
  private tournamentService = inject(TournamentService);
  private bracketService = inject(BracketService);
  private communityService = inject(CommunityService);
  private eloService = inject(EloService);
  private uploadService = inject(UploadService);
  authService = inject(AuthService);
  tournamentUI = inject(TournamentUIService);

  tournament = signal<Tournament | null>(null);
  community = signal<Community | null>(null);
  eloSystem = signal<EloSystem | null>(null);
  loading = signal(true);
  error = signal('');
  startingTournament = signal(false);

  // Participant state
  participants = signal<Participant[]>([]);
  participantsLoading = signal(false);

  // Bracket refresh trigger - increment to trigger reload
  bracketRefreshKey = signal(0);

  // Stage details toggle for multi-stage tournaments
  showStageDetails = signal(false);

  // Seeding popover state
  activeSeedPopoverStage = signal<TournamentStage | null>(null);
  stagePool = signal<StagePoolEntry[]>([]);
  stagePoolLoaded = signal(false);

  // Logo upload state
  uploadingLogo = signal(false);

  // Computed properties
  isOrganizer = computed(() => {
    const t = this.tournament();
    const user = this.authService.user();
    return t && user ? t.organizer_id === user.id : false;
  });

  isLoggedIn = computed(() => this.authService.isLoggedIn());

  canStartTournament = computed(() => {
    const t = this.tournament();
    if (!t) return false;
    return this.isOrganizer() &&
           t.status === 'registration' &&
           this.participants().length >= 2;
  });

  breadcrumbs: BreadcrumbItem[] = [
    { label: 'Tournaments', route: '/tournaments' },
    { label: 'Loading...' }
  ];

  ngOnInit(): void {
    // Sync auth state to UI service
    this.tournamentUI.isLoggedIn.set(this.authService.isLoggedIn());
    const slug = this.route.snapshot.paramMap.get('slug');
    if (slug) {
      this.loadTournament(slug);
    } else {
      this.error.set('Tournament not found');
      this.loading.set(false);
    }
  }

  loadTournament(slug: string): void {
    this.loading.set(true);
    this.error.set('');

    this.tournamentService.getTournament(slug).subscribe({
      next: (tournament) => {
        this.tournament.set(tournament);
        this.tournamentUI.setTournament(tournament);
        this.breadcrumbs = [
          { label: 'Tournaments', route: '/tournaments' },
          { label: tournament.name }
        ];
        this.loading.set(false);
        this.loadParticipants(slug);
        if (tournament.community_id) {
          this.loadCommunity(tournament.community_id);
        }
        // Load stages for multi-stage tournaments
        if (tournament.format === 'multi_stage') {
          this.loadStages(slug);
        }
        // Sync organizer status to UI service
        this.syncOrganizerStatus();
      },
      error: (err) => {
        this.error.set(err.error?.error || 'Failed to load tournament');
        this.loading.set(false);
      }
    });
  }

  loadStages(slug: string): void {
    this.tournamentService.getStages(slug).subscribe({
      next: (stages) => {
        // Update tournament with loaded stages
        this.tournament.update(t => t ? { ...t, stages } : null);
        this.tournamentUI.stages.set(stages);
        // Also load stage pool for participant assignments
        this.loadStagePool();
      },
      error: () => {
        // Silently fail - stages display is optional
      }
    });
  }

  loadCommunity(communityId: number): void {
    this.communityService.getCommunityById(communityId).subscribe({
      next: (community) => {
        this.community.set(community);
        // Load ELO system if tournament has one
        const t = this.tournament();
        if (t?.elo_system_id && community.slug) {
          this.loadEloSystem(community.slug, t.elo_system_id);
        }
      },
      error: () => {
        // Silently fail - community display is optional
        this.community.set(null);
      }
    });
  }

  loadEloSystem(communitySlug: string, eloSystemId: number): void {
    this.eloService.getEloSystem(communitySlug, eloSystemId).subscribe({
      next: (eloSystem) => {
        this.eloSystem.set(eloSystem);
      },
      error: () => {
        // Silently fail - ELO system display is optional
        this.eloSystem.set(null);
      }
    });
  }

  loadParticipants(slug: string): void {
    this.participantsLoading.set(true);

    this.tournamentService.getParticipants(slug).subscribe({
      next: (participants) => {
        this.participants.set(participants || []);
        this.tournamentUI.participants.set(participants || []);
        this.participantsLoading.set(false);
      },
      error: () => {
        this.participants.set([]);
        this.tournamentUI.participants.set([]);
        this.participantsLoading.set(false);
      }
    });
  }

  private syncOrganizerStatus(): void {
    this.tournamentUI.isOrganizer.set(this.isOrganizer());
  }

  getStatusLabel(status: string): string {
    const labels: Record<string, string> = {
      registration: 'Registration Open',
      in_progress: 'In Progress',
      completed: 'Completed',
      cancelled: 'Cancelled'
    };
    return labels[status] || status;
  }

  getStatusColor(status: string): string {
    const colors: Record<string, string> = {
      registration: 'bg-green-100 text-green-800',
      in_progress: 'bg-blue-100 text-blue-800',
      completed: 'bg-purple-100 text-purple-800',
      cancelled: 'bg-red-100 text-red-800'
    };
    return colors[status] || 'bg-gray-100 text-gray-800';
  }

  // Stage details toggle and helpers
  toggleStageDetails(): void {
    this.showStageDetails.update(v => !v);
  }

  getSortedStages(): TournamentStage[] {
    const stages = this.tournament()?.stages;
    if (!stages) return [];
    // Sort: group stages by stage_order first, then final stage last
    return [...stages].sort((a, b) => {
      if (a.stage_type === 'final') return 1;
      if (b.stage_type === 'final') return -1;
      return a.stage_order - b.stage_order;
    });
  }

  getStageFormatLabel(format: string): string {
    const labels: Record<string, string> = {
      single_elimination: 'Single Elim',
      double_elimination: 'Double Elim',
      swiss: 'Swiss'
    };
    return labels[format] || format;
  }

  ngOnDestroy(): void {
    this.tournamentUI.reset();
  }

  // Event handlers from SidePanel
  onParticipantAdded(participant: Participant): void {
    this.participants.update(list => [...list, participant]);
    this.tournamentUI.addParticipant(participant);
  }

  onParticipantRemoved(id: number): void {
    this.participants.update(list => list.filter(p => p.id !== id));
    this.tournamentUI.removeParticipant(id);
  }

  onParticipantWithdrawn(id: number): void {
    // Update participant status locally
    this.participants.update(list =>
      list.map(p => p.id === id ? { ...p, status: 'withdrawn' as const } : p)
    );
    const updated = this.participants().find(p => p.id === id);
    if (updated) {
      this.tournamentUI.updateParticipant(updated);
    }
    // Trigger bracket refresh to show forfeited matches
    this.bracketRefreshKey.update(k => k + 1);
    this.tournamentUI.refreshBracket();
  }

  onParticipantUpdated(participant: Participant): void {
    this.participants.update(list =>
      list.map(p => p.id === participant.id ? participant : p)
    );
    this.tournamentUI.updateParticipant(participant);
  }

  onSeedingChanged(participants: Participant[]): void {
    this.participants.set(participants);
    this.tournamentUI.participants.set(participants);
  }

  onSelfRegistered(participant: Participant): void {
    this.participants.update(list => [...list, participant]);
    this.tournamentUI.addParticipant(participant);
  }

  onLeft(id: number): void {
    this.participants.update(list => list.filter(p => p.id !== id));
    this.tournamentUI.removeParticipant(id);
  }

  onTournamentUpdated(tournament: Tournament): void {
    this.tournament.set(tournament);
    this.tournamentUI.setTournament(tournament);
    this.breadcrumbs = [
      { label: 'Tournaments', route: '/tournaments' },
      { label: tournament.name }
    ];
  }

  onStageUpdated(updatedStage: TournamentStage): void {
    this.tournament.update(t => {
      if (!t || !t.stages) return t;
      return {
        ...t,
        stages: t.stages.map(s => s.id === updatedStage.id ? updatedStage : s)
      };
    });
    // Sync to UI service
    const stages = this.tournament()?.stages;
    if (stages) {
      this.tournamentUI.stages.set(stages);
    }
  }

  startTournament(): void {
    const t = this.tournament();
    const p = this.participants();
    if (!t || p.length < 2) return;

    this.startingTournament.set(true);
    this.error.set('');

    // Multi-stage tournaments use a different flow - start the first stage
    if (t.format === 'multi_stage') {
      this.tournamentService.startStage(t.slug).subscribe({
        next: () => {
          // Reload tournament to get updated status and stages
          this.loadTournament(t.slug);
          this.startingTournament.set(false);
        },
        error: (err) => {
          this.error.set(err.error?.error || 'Failed to start tournament');
          this.startingTournament.set(false);
        }
      });
      return;
    }

    const bracketParticipants = p.map((participant, index) => ({
      id: participant.id,
      name: participant.display_name,
      seed: index + 1,
      icon_url: participant.icon_url
    }));

    const bracketRequest: CreateBracketRequest = {
      tournament_id: t.id,
      format: t.format,
      participants: bracketParticipants
    };

    // Pass swiss_rounds for Swiss format tournaments
    if (t.format === 'swiss' && t.swiss_rounds) {
      bracketRequest.swiss_rounds = t.swiss_rounds;
    }

    this.bracketService.createBracket(bracketRequest).pipe(
      switchMap(() => this.tournamentService.updateTournament(t.slug, { status: 'in_progress' }))
    ).subscribe({
      next: (updatedTournament) => {
        this.tournament.set(updatedTournament);
        this.startingTournament.set(false);
      },
      error: (err) => {
        this.error.set(err.error?.error || 'Failed to start tournament');
        this.startingTournament.set(false);
      }
    });
  }

  // Seeding popover methods

  canSeedStage(stage: TournamentStage): boolean {
    return this.isOrganizer() &&
           !stage.is_active &&
           !stage.is_complete &&
           this.tournament()?.status === 'registration';
  }

  openSeedPopover(stage: TournamentStage, event: Event): void {
    event.stopPropagation();
    // Load stage pool if not loaded
    if (!this.stagePoolLoaded()) {
      this.loadStagePool();
    }
    this.activeSeedPopoverStage.set(stage);
  }

  closeSeedPopover(): void {
    this.activeSeedPopoverStage.set(null);
  }

  loadStagePool(): void {
    const t = this.tournament();
    if (!t) return;

    this.tournamentService.getStagePool(t.slug).subscribe({
      next: (response) => {
        this.stagePool.set(response.entries);
        this.stagePoolLoaded.set(true);
      },
      error: () => {
        // Silently fail, pool is optional
        this.stagePoolLoaded.set(true);
      }
    });
  }

  onPoolUpdated(entries: StagePoolEntry[]): void {
    this.stagePool.set(entries);
    this.closeSeedPopover();
  }

  getParticipantById(participantId: number): Participant | undefined {
    return this.participants().find(p => p.id === participantId);
  }

  // Returns participants assigned to a specific stage
  getPoolParticipantsForStage(stageId: number): Participant[] {
    return this.stagePool()
      .filter(e => e.stage_id === stageId)
      .map(e => this.getParticipantById(e.participant_id))
      .filter((p): p is Participant => p !== undefined);
  }

  getStagePoolArray(): StagePoolEntry[] {
    return this.stagePool();
  }

  /** Navigate to a stage in the bracket (for multi-stage tournaments) */
  onStageCardClick(stage: TournamentStage, event: Event): void {
    // If seeding is available, that takes priority
    if (this.canSeedStage(stage)) {
      this.openSeedPopover(stage, event);
      return;
    }

    // Otherwise, if tournament is in progress, select this stage in the bracket
    const t = this.tournament();
    if (t && (t.status === 'in_progress' || t.status === 'completed')) {
      this.tournamentUI.selectStage(stage.id);
    }
  }

  /** Check if a stage card should be clickable for navigation */
  isStageClickable(stage: TournamentStage): boolean {
    const t = this.tournament();
    if (!t) return false;
    // Clickable for seeding or for navigation when in progress
    return this.canSeedStage(stage) ||
           t.status === 'in_progress' ||
           t.status === 'completed';
  }

  // Logo upload methods
  onLogoFileSelected(event: Event): void {
    const input = event.target as HTMLInputElement;
    if (!input.files || input.files.length === 0) return;

    const file = input.files[0];
    const t = this.tournament();
    if (!t) return;

    // Validate file
    const validation = this.uploadService.validateImageFile(file);
    if (!validation.valid) {
      this.error.set(validation.error || 'Invalid file');
      return;
    }

    this.uploadLogo(t.slug, file);
  }

  private uploadLogo(slug: string, file: File): void {
    this.uploadingLogo.set(true);
    this.error.set('');

    // Step 1: Get presigned upload URL
    this.tournamentService.getLogoUploadUrl(slug, file.type).subscribe({
      next: (response) => {
        // Step 2: Upload file directly to presigned URL
        this.uploadService.uploadToPresignedUrl(response.upload_url, file).subscribe({
          next: () => {
            // Step 3: Update tournament with new logo URL
            this.tournamentService.updateLogoUrl(slug, response.logo_url).subscribe({
              next: () => {
                // Update local state
                const t = this.tournament();
                if (t) {
                  this.tournament.set({ ...t, logo_url: response.logo_url });
                  this.tournamentUI.setTournament({ ...t, logo_url: response.logo_url });
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
}
