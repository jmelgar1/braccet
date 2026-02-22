import { Component, inject, signal, computed, OnInit } from '@angular/core';
import { DatePipe } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { CommunityService } from '../../services/community.service';
import { TournamentService } from '../../services/tournament.service';
import { EloService } from '../../services/elo.service';
import { AuthService } from '../../services/auth.service';
import { Community, CommunityMember, AddMemberRequest, MemberRole } from '../../models/community.model';
import { Tournament } from '../../models/tournament.model';
import { EloSystem, MemberEloRating, CreateEloSystemRequest } from '../../models/elo.model';

@Component({
  selector: 'app-community-detail',
  imports: [DatePipe, FormsModule, RouterLink],
  templateUrl: './community-detail.html',
  styleUrl: './community-detail.css'
})
export class CommunityDetail implements OnInit {
  private communityService = inject(CommunityService);
  private tournamentService = inject(TournamentService);
  private eloService = inject(EloService);
  private authService = inject(AuthService);
  private route = inject(ActivatedRoute);
  private router = inject(Router);

  community = signal<Community | null>(null);
  members = signal<CommunityMember[]>([]);
  tournaments = signal<Tournament[]>([]);
  loading = signal(true);
  error = signal('');

  // Tab state
  activeTab = signal<'members' | 'tournaments' | 'leaderboards' | 'elo'>('members');

  // ELO state
  eloSystems = signal<EloSystem[]>([]);
  selectedEloSystem = signal<EloSystem | null>(null);
  leaderboard = signal<MemberEloRating[]>([]);
  loadingElo = signal(false);
  showCreateEloForm = signal(false);
  creatingElo = signal(false);
  createEloError = signal('');
  newEloSystem: CreateEloSystemRequest = {
    name: '',
    description: undefined,
    starting_rating: 1000,
    k_factor: 32,
    floor_rating: 100,
    provisional_games: 10,
    provisional_k_factor: 64,
    win_streak_enabled: false,
    win_streak_threshold: 3,
    win_streak_bonus: 5,
    decay_enabled: false,
    decay_days: 30,
    decay_amount: 10,
    decay_floor: 800,
    is_default: false
  };

  // Add member form
  showAddMemberForm = signal(false);
  addingMember = signal(false);
  addMemberError = signal('');
  newMember: AddMemberRequest = { display_name: '' };

  // Current user info
  currentUser = computed(() => this.authService.user());

  // Check if current user is owner or admin
  isOwnerOrAdmin = computed(() => {
    const user = this.currentUser();
    const comm = this.community();
    if (!user || !comm) return false;

    // Check if user is the community owner
    if (comm.owner_id === user.id) return true;

    // Check if user is an admin in the members list
    const member = this.members().find(m => m.user_id === user.id);
    return member?.role === 'admin' || member?.role === 'owner';
  });

  isOwner = computed(() => {
    const user = this.currentUser();
    const comm = this.community();
    if (!user || !comm) return false;
    return comm.owner_id === user.id;
  });

  ngOnInit(): void {
    const slug = this.route.snapshot.paramMap.get('slug');
    if (slug) {
      this.loadCommunity(slug);
    }
  }

  loadCommunity(slug: string): void {
    this.loading.set(true);
    this.error.set('');

    this.communityService.getCommunity(slug).subscribe({
      next: (community) => {
        this.community.set(community);
        this.loadMembers(slug);
        this.loadTournaments(community.id);
        this.loading.set(false);
      },
      error: (err) => {
        this.error.set(err.error?.error || 'Failed to load community');
        this.loading.set(false);
      }
    });
  }

  loadMembers(slug: string): void {
    this.communityService.getMembers(slug).subscribe({
      next: (members) => {
        this.members.set(members || []);
      },
      error: (err) => {
        console.error('Failed to load members:', err);
      }
    });
  }

  loadTournaments(communityId: number): void {
    this.tournamentService.getTournamentsByCommunity(communityId).subscribe({
      next: (tournaments) => {
        this.tournaments.set(tournaments || []);
      },
      error: (err) => {
        console.error('Failed to load tournaments:', err);
      }
    });
  }

  setActiveTab(tab: 'members' | 'tournaments' | 'leaderboards' | 'elo'): void {
    this.activeTab.set(tab);
    if ((tab === 'elo' || tab === 'leaderboards') && this.eloSystems().length === 0) {
      this.loadEloSystems();
    }
  }

  loadEloSystems(): void {
    const slug = this.community()?.slug;
    if (!slug) return;

    this.loadingElo.set(true);
    this.eloService.getEloSystems(slug).subscribe({
      next: (systems) => {
        this.eloSystems.set(systems || []);
        // Select the default system or the first one
        const defaultSystem = systems?.find(s => s.is_default) || systems?.[0];
        if (defaultSystem) {
          this.selectEloSystem(defaultSystem);
        }
        this.loadingElo.set(false);
      },
      error: (err) => {
        console.error('Failed to load ELO systems:', err);
        this.loadingElo.set(false);
      }
    });
  }

  selectEloSystem(system: EloSystem): void {
    this.selectedEloSystem.set(system);
    this.loadLeaderboard(system.id);
  }

  onSystemSelect(systemIdStr: string): void {
    const systemId = parseInt(systemIdStr, 10);
    const system = this.eloSystems().find(s => s.id === systemId);
    if (system) {
      this.selectEloSystem(system);
    }
  }

  loadLeaderboard(systemId: number): void {
    const slug = this.community()?.slug;
    if (!slug) return;

    this.eloService.getLeaderboard(slug, systemId, 50).subscribe({
      next: (ratings) => {
        this.leaderboard.set(ratings || []);
      },
      error: (err) => {
        console.error('Failed to load leaderboard:', err);
      }
    });
  }

  toggleCreateEloForm(): void {
    this.showCreateEloForm.update(v => !v);
    if (!this.showCreateEloForm()) {
      this.resetCreateEloForm();
    }
  }

  resetCreateEloForm(): void {
    this.newEloSystem = {
      name: '',
      description: undefined,
      starting_rating: 1000,
      k_factor: 32,
      floor_rating: 100,
      provisional_games: 10,
      provisional_k_factor: 64,
      win_streak_enabled: false,
      win_streak_threshold: 3,
      win_streak_bonus: 5,
      decay_enabled: false,
      decay_days: 30,
      decay_amount: 10,
      decay_floor: 800,
      is_default: false
    };
    this.createEloError.set('');
  }

  onCreateEloSystem(): void {
    if (!this.newEloSystem.name?.trim()) {
      this.createEloError.set('Name is required');
      return;
    }

    const slug = this.community()?.slug;
    if (!slug) return;

    this.creatingElo.set(true);
    this.createEloError.set('');

    this.eloService.createEloSystem(slug, this.newEloSystem).subscribe({
      next: (system) => {
        this.eloSystems.update(list => [...list, system]);
        this.selectEloSystem(system);
        this.showCreateEloForm.set(false);
        this.resetCreateEloForm();
        this.creatingElo.set(false);
      },
      error: (err) => {
        this.createEloError.set(err.error?.error || 'Failed to create ELO system');
        this.creatingElo.set(false);
      }
    });
  }

  deleteEloSystem(system: EloSystem, event: Event): void {
    event.stopPropagation();

    if (!confirm(`Are you sure you want to delete the "${system.name}" rating system? This will remove all ratings and history.`)) {
      return;
    }

    const slug = this.community()?.slug;
    if (!slug) return;

    this.eloService.deleteEloSystem(slug, system.id).subscribe({
      next: () => {
        this.eloSystems.update(list => list.filter(s => s.id !== system.id));
        if (this.selectedEloSystem()?.id === system.id) {
          const remaining = this.eloSystems();
          if (remaining.length > 0) {
            this.selectEloSystem(remaining[0]);
          } else {
            this.selectedEloSystem.set(null);
            this.leaderboard.set([]);
          }
        }
      },
      error: (err) => {
        this.error.set(err.error?.error || 'Failed to delete ELO system');
      }
    });
  }

  getWinRate(rating: MemberEloRating): number {
    if (rating.games_played === 0) return 0;
    return Math.round((rating.games_won / rating.games_played) * 100);
  }

  toggleAddMemberForm(): void {
    this.showAddMemberForm.update(v => !v);
    if (!this.showAddMemberForm()) {
      this.resetAddMemberForm();
    }
  }

  resetAddMemberForm(): void {
    this.newMember = { display_name: '' };
    this.addMemberError.set('');
  }

  onAddMember(): void {
    if (!this.newMember.display_name.trim()) {
      this.addMemberError.set('Display name is required');
      return;
    }

    const slug = this.community()?.slug;
    if (!slug) return;

    this.addingMember.set(true);
    this.addMemberError.set('');

    const request: AddMemberRequest = {
      display_name: this.newMember.display_name.trim()
    };

    this.communityService.addMember(slug, request).subscribe({
      next: (member) => {
        this.members.update(list => [...list, member]);
        this.showAddMemberForm.set(false);
        this.resetAddMemberForm();
        this.addingMember.set(false);
      },
      error: (err) => {
        this.addMemberError.set(err.error?.error || 'Failed to add member');
        this.addingMember.set(false);
      }
    });
  }

  removeMember(memberId: number, displayName: string, event: Event): void {
    event.stopPropagation();

    if (!confirm(`Are you sure you want to remove "${displayName}" from this community?`)) {
      return;
    }

    const slug = this.community()?.slug;
    if (!slug) return;

    this.communityService.removeMember(slug, memberId).subscribe({
      next: () => {
        this.members.update(members => members.filter(m => m.id !== memberId));
      },
      error: (err) => {
        this.error.set(err.error?.error || 'Failed to remove member');
      }
    });
  }

  updateMemberRole(memberId: number, role: MemberRole): void {
    const slug = this.community()?.slug;
    if (!slug) return;

    this.communityService.updateMemberRole(slug, memberId, role).subscribe({
      next: (updatedMember) => {
        this.members.update(members =>
          members.map(m => m.id === memberId ? updatedMember : m)
        );
      },
      error: (err) => {
        this.error.set(err.error?.error || 'Failed to update member role');
      }
    });
  }

  onTournamentClick(slug: string): void {
    this.router.navigate(['/tournaments', slug]);
  }

  getRoleBadgeClass(role: MemberRole): string {
    switch (role) {
      case 'owner':
        return 'bg-yellow-100 text-yellow-800';
      case 'admin':
        return 'bg-blue-100 text-blue-800';
      default:
        return 'bg-gray-100 text-gray-600';
    }
  }

  getStatusBadgeClass(status: string): string {
    switch (status) {
      case 'in_progress':
        return 'bg-green-100 text-green-800';
      case 'completed':
        return 'bg-gray-100 text-gray-600';
      default:
        return 'bg-blue-100 text-blue-800';
    }
  }
}
