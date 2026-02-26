import { Component, input, output, inject, signal, computed, OnInit, OnDestroy, viewChild } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { CdkDragDrop, DragDropModule, moveItemInArray } from '@angular/cdk/drag-drop';
import { Subject, of, forkJoin, from } from 'rxjs';
import { debounceTime, distinctUntilChanged, switchMap, takeUntil, catchError, concatMap, toArray } from 'rxjs/operators';
import { Participant, MemberSearchResult, RegionInviteMember } from '../../../../models/tournament.model';
import { CommunityMember } from '../../../../models/community.model';
import { TournamentService } from '../../../../services/tournament.service';
import { TournamentUIService } from '../../../../services/tournament-ui.service';
import { CommunityService } from '../../../../services/community.service';
import { AuthService } from '../../../../services/auth.service';
import { EditMemberModal } from '../../../../components/edit-member-modal/edit-member-modal';

@Component({
  selector: 'app-participants-tab',
  imports: [FormsModule, DragDropModule, EditMemberModal],
  templateUrl: './participants-tab.html'
})
export class ParticipantsTab implements OnInit, OnDestroy {
  private tournamentService = inject(TournamentService);
  private communityService = inject(CommunityService);
  private tournamentUI = inject(TournamentUIService);
  authService = inject(AuthService);

  // Read from service
  tournament = computed(() => this.tournamentUI.tournament()!);
  participants = computed(() => this.tournamentUI.participants());
  isOrganizer = computed(() => this.tournamentUI.isOrganizer());

  // These inputs are still passed from side-panel
  currentUserParticipant = input<Participant | null>(null);
  canSelfRegister = input(false);
  communitySlug = input<string | null>(null);

  participantAdded = output<Participant>();
  participantRemoved = output<number>();
  participantWithdrawn = output<number>();
  participantUpdated = output<Participant>();
  seedingChanged = output<Participant[]>();
  selfRegistered = output<Participant>();
  left = output<number>();

  // Edit member modal state
  editingParticipant = signal<Participant | null>(null);
  editingMember = signal<CommunityMember | null>(null);
  editModal = viewChild<EditMemberModal>('editModal');

  newParticipantName = '';
  addingParticipant = signal(false);
  savingSeeding = signal(false);
  error = signal('');

  // Autocomplete state
  searchResults = signal<MemberSearchResult[]>([]);
  showDropdown = signal(false);
  selectedMember = signal<MemberSearchResult | null>(null);
  isSearching = signal(false);

  // Bulk invite by region state
  showRegionInviteModal = signal(false);
  availableRegions = signal<string[]>([]);
  selectedRegion = signal<string | null>(null);
  inviteCount = signal(10);
  regionMembers = signal<RegionInviteMember[]>([]);
  selectedMemberIds = signal<Set<number>>(new Set());
  loadingRegions = signal(false);
  loadingRegionMembers = signal(false);
  bulkInviting = signal(false);

  private searchSubject = new Subject<string>();
  private destroy$ = new Subject<void>();

  // Check if this is a community tournament (autocomplete enabled)
  get isEloEnabled(): boolean {
    const t = this.tournament();
    return !!t.community_id;
  }

  // Check if "Order by ELO" button should be shown (community tournament in registration with participants that have ELO)
  canOrderByElo = computed(() => {
    const t = this.tournament();
    if (!t.community_id || t.status !== 'registration') return false;
    // Check if at least one participant has an ELO rating
    return this.participants().some(p => p.elo_rating != null);
  });

  ngOnInit(): void {
    // Set up debounced search
    this.searchSubject.pipe(
      debounceTime(300),
      distinctUntilChanged(),
      switchMap(query => {
        if (!query.trim() || query.length < 2 || !this.isEloEnabled) {
          return of([]);
        }
        this.isSearching.set(true);
        return this.tournamentService.searchAvailableMembers(
          this.tournament().slug,
          query
        ).pipe(
          catchError(() => of([]))
        );
      }),
      takeUntil(this.destroy$)
    ).subscribe(results => {
      this.searchResults.set(results);
      this.showDropdown.set(results.length > 0);
      this.isSearching.set(false);
    });
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }

  onSearchInput(value: string): void {
    this.newParticipantName = value;
    this.selectedMember.set(null);  // Clear selection when typing

    if (this.isEloEnabled) {
      this.searchSubject.next(value);
    }
  }

  selectMember(member: MemberSearchResult): void {
    this.selectedMember.set(member);
    this.newParticipantName = member.display_name;
    this.showDropdown.set(false);
    this.searchResults.set([]);
  }

  hideDropdown(): void {
    // Delay to allow click on dropdown item
    setTimeout(() => this.showDropdown.set(false), 200);
  }

  // Order participants by ELO ranking and save the new seeding
  orderByElo(): void {
    const list = [...this.participants()];

    // Sort by ELO descending (highest first), participants without ELO go to end
    list.sort((a, b) => {
      const eloA = a.elo_rating ?? 0;
      const eloB = b.elo_rating ?? 0;
      return eloB - eloA;
    });

    this.saveSeeding(list);
  }

  addParticipant(): void {
    const t = this.tournament();
    const name = this.newParticipantName.trim();
    if (!t || !name) return;

    // Check for duplicate name (case-insensitive)
    if (this.isDuplicateName(name)) {
      this.error.set('A participant with this name already exists');
      return;
    }

    this.addingParticipant.set(true);
    this.error.set('');

    // Build request - include community_member_id if a member was selected
    const request: { display_name: string; community_member_id?: number } = {
      display_name: name
    };

    const selected = this.selectedMember();
    if (selected) {
      request.community_member_id = selected.id;
    }

    this.tournamentService.addParticipant(t.slug, request).subscribe({
      next: (participant) => {
        this.participantAdded.emit(participant);
        this.newParticipantName = '';
        this.selectedMember.set(null);
        this.addingParticipant.set(false);
      },
      error: (err) => {
        this.error.set(err.error?.error || 'Failed to add participant');
        this.addingParticipant.set(false);
      }
    });
  }

  selfRegister(): void {
    const t = this.tournament();
    const user = this.authService.user();
    if (!t || !user) return;

    this.addingParticipant.set(true);
    this.error.set('');

    this.tournamentService.addParticipant(t.slug, {
      user_id: user.id,
      display_name: user.display_name
    }).subscribe({
      next: (participant) => {
        this.selfRegistered.emit(participant);
        this.addingParticipant.set(false);
      },
      error: (err) => {
        this.error.set(err.error?.error || 'Failed to join tournament');
        this.addingParticipant.set(false);
      }
    });
  }

  removeParticipant(participant: Participant): void {
    const t = this.tournament();
    if (!t) return;

    this.tournamentService.removeParticipant(t.slug, participant.id).subscribe({
      next: () => {
        this.participantRemoved.emit(participant.id);
      },
      error: (err) => {
        this.error.set(err.error?.error || 'Failed to remove participant');
      }
    });
  }

  leaveTournament(): void {
    const participant = this.currentUserParticipant();
    if (participant) {
      this.tournamentService.removeParticipant(this.tournament().slug, participant.id).subscribe({
        next: () => {
          this.left.emit(participant.id);
        },
        error: (err) => {
          this.error.set(err.error?.error || 'Failed to leave tournament');
        }
      });
    }
  }

  withdrawParticipant(participant: Participant): void {
    const t = this.tournament();
    if (!t) return;

    if (!confirm(`Are you sure you want to withdraw ${participant.display_name}? This will forfeit all their pending matches.`)) {
      return;
    }

    this.tournamentService.withdrawParticipant(t.slug, participant.id).subscribe({
      next: () => {
        this.participantWithdrawn.emit(participant.id);
      },
      error: (err) => {
        this.error.set(err.error?.error || 'Failed to withdraw participant');
      }
    });
  }

  canWithdraw(participant: Participant): boolean {
    const t = this.tournament();
    return t.status === 'in_progress' &&
           participant.status !== 'eliminated' &&
           participant.status !== 'disqualified' &&
           participant.status !== 'withdrawn';
  }

  onDrop(event: CdkDragDrop<Participant[]>): void {
    if (event.previousIndex === event.currentIndex) return;

    const list = [...this.participants()];
    moveItemInArray(list, event.previousIndex, event.currentIndex);
    this.saveSeeding(list);
  }

  private isDuplicateName(name: string): boolean {
    const lowerName = name.toLowerCase();
    return this.participants().some(p => p.display_name.toLowerCase() === lowerName);
  }

  private saveSeeding(orderedParticipants: Participant[]): void {
    const t = this.tournament();
    if (!t) return;

    const seeds: Record<number, number> = {};
    orderedParticipants.forEach((p, index) => {
      seeds[p.id] = index + 1;
    });

    this.savingSeeding.set(true);
    this.tournamentService.updateSeeding(t.slug, { seeds }).subscribe({
      next: (updatedParticipants) => {
        this.seedingChanged.emit(updatedParticipants);
        this.savingSeeding.set(false);
      },
      error: (err) => {
        this.error.set(err.error?.error || 'Failed to update seeding');
        this.savingSeeding.set(false);
      }
    });
  }

  // Check if participant can be edited (any non-user participant in a community tournament)
  canEdit(participant: Participant): boolean {
    return !participant.user_id && !!this.communitySlug();
  }

  // Open edit modal for a participant
  openEditModal(participant: Participant): void {
    this.editingParticipant.set(participant);

    // Construct a minimal CommunityMember for the modal
    // Use id: 0 as marker for participants without community_member_id (will be promoted on save)
    const member: CommunityMember = {
      id: participant.community_member_id ?? 0,
      community_id: this.tournament().community_id!,
      display_name: participant.display_name,
      role: 'member',
      is_ghost: true,
      icon_url: participant.icon_url,
      region: participant.region,
      matches_played: 0,
      matches_won: 0,
      joined_at: '',
      created_at: '',
      updated_at: ''
    };

    this.editingMember.set(member);
  }

  closeEditModal(): void {
    this.editingParticipant.set(null);
    this.editingMember.set(null);
  }

  onMemberUpdated(updatedMember: CommunityMember): void {
    const participant = this.editingParticipant();
    if (!participant) return;

    // Create updated participant with new data
    const updatedParticipant: Participant = {
      ...participant,
      community_member_id: updatedMember.id || participant.community_member_id,
      display_name: updatedMember.display_name,
      icon_url: updatedMember.icon_url,
      region: updatedMember.region
    };

    this.participantUpdated.emit(updatedParticipant);
    this.closeEditModal();
  }

  // Handle promotion request from edit modal (for participants without community_member_id)
  onRequestPromotion(): void {
    const participant = this.editingParticipant();
    if (!participant) return;

    this.tournamentService.promoteParticipant(this.tournament().slug, participant.id).subscribe({
      next: (promotedParticipant) => {
        // Update local participant state
        this.editingParticipant.set(promotedParticipant);

        // Update the member with the new community_member_id
        const currentMember = this.editingMember();
        if (currentMember) {
          this.editingMember.set({
            ...currentMember,
            id: promotedParticipant.community_member_id!
          });
        }

        // Tell the modal to continue with the pending action
        const modal = this.editModal();
        if (modal) {
          modal.completePromotion(promotedParticipant.community_member_id!);
        }
      },
      error: (err) => {
        this.error.set(err.error?.error || 'Failed to promote participant');
      }
    });
  }

  // Bulk invite by region methods

  openRegionInviteModal(): void {
    this.showRegionInviteModal.set(true);
    this.loadRegions();
  }

  closeRegionInviteModal(): void {
    this.showRegionInviteModal.set(false);
    this.selectedRegion.set(null);
    this.regionMembers.set([]);
    this.selectedMemberIds.set(new Set());
    this.inviteCount.set(10);
  }

  private loadRegions(): void {
    const slug = this.communitySlug();
    if (!slug) return;

    this.loadingRegions.set(true);
    this.communityService.getRegions(slug).subscribe({
      next: (regions) => {
        this.availableRegions.set(regions);
        this.loadingRegions.set(false);
      },
      error: () => {
        this.error.set('Failed to load regions');
        this.loadingRegions.set(false);
      }
    });
  }

  onRegionSelected(region: string): void {
    this.selectedRegion.set(region || null);
    if (region) {
      this.loadRegionMembers();
    } else {
      this.regionMembers.set([]);
    }
  }

  onInviteCountChanged(count: number): void {
    this.inviteCount.set(count);
    if (this.selectedRegion()) {
      this.loadRegionMembers();
    }
  }

  private loadRegionMembers(): void {
    const region = this.selectedRegion();
    if (!region) return;

    this.loadingRegionMembers.set(true);
    this.tournamentService.getTopMembersForRegionInvite(
      this.tournament().slug,
      region,
      this.inviteCount()
    ).subscribe({
      next: (members) => {
        this.regionMembers.set(members);
        // Select all members by default
        this.selectedMemberIds.set(new Set(members.map(m => m.id)));
        this.loadingRegionMembers.set(false);
      },
      error: () => {
        this.error.set('Failed to load members');
        this.loadingRegionMembers.set(false);
      }
    });
  }

  toggleMemberSelection(memberId: number): void {
    const current = this.selectedMemberIds();
    const updated = new Set(current);
    if (updated.has(memberId)) {
      updated.delete(memberId);
    } else {
      updated.add(memberId);
    }
    this.selectedMemberIds.set(updated);
  }

  toggleSelectAll(): void {
    const members = this.regionMembers();
    const selected = this.selectedMemberIds();
    if (selected.size === members.length) {
      // Deselect all
      this.selectedMemberIds.set(new Set());
    } else {
      // Select all
      this.selectedMemberIds.set(new Set(members.map(m => m.id)));
    }
  }

  isMemberSelected(memberId: number): boolean {
    return this.selectedMemberIds().has(memberId);
  }

  get selectedCount(): number {
    return this.selectedMemberIds().size;
  }

  bulkInviteRegionMembers(): void {
    const allMembers = this.regionMembers();
    const selectedIds = this.selectedMemberIds();
    const membersToInvite = allMembers.filter(m => selectedIds.has(m.id));

    if (membersToInvite.length === 0) return;

    this.bulkInviting.set(true);
    this.error.set('');

    // Add members sequentially to avoid race conditions
    from(membersToInvite).pipe(
      concatMap(member =>
        this.tournamentService.addParticipant(this.tournament().slug, {
          display_name: member.display_name,
          community_member_id: member.id
        }).pipe(
          catchError(() => of(null)) // Continue on individual failures
        )
      ),
      toArray()
    ).subscribe({
      next: (results) => {
        const added = results.filter((p): p is Participant => p !== null);
        added.forEach(p => this.participantAdded.emit(p));
        this.bulkInviting.set(false);
        this.closeRegionInviteModal();
      },
      error: () => {
        this.error.set('Failed to invite members');
        this.bulkInviting.set(false);
      }
    });
  }
}
