import { Component, input, output, signal, computed, inject } from '@angular/core';
import { Tournament, Participant, TournamentStage } from '../../../../models/tournament.model';
import { User, AuthService } from '../../../../services/auth.service';
import { TournamentUIService } from '../../../../services/tournament-ui.service';
import { BracketTab } from '../bracket-tab/bracket-tab';
import { ParticipantsTab } from '../participants-tab/participants-tab';
import { SettingsTab } from '../settings-tab/settings-tab';
import { StandingsTab } from '../standings-tab/standings-tab';

type TabId = 'bracket' | 'participants' | 'standings' | 'settings';

interface Tab {
  id: TabId;
  label: string;
}

@Component({
  selector: 'app-side-panel',
  imports: [BracketTab, ParticipantsTab, SettingsTab, StandingsTab],
  templateUrl: './side-panel.html'
})
export class SidePanel {
  tournamentUI = inject(TournamentUIService);
  private authService = inject(AuthService);

  // These inputs are still used for some props not in service
  currentUser = input<User | null>(null);
  communitySlug = input<string | null>(null);

  // Forward participant events (still needed for tournament-detail to update local state)
  participantAdded = output<Participant>();
  participantRemoved = output<number>();
  participantWithdrawn = output<number>();
  participantUpdated = output<Participant>();
  seedingChanged = output<Participant[]>();
  selfRegistered = output<Participant>();
  left = output<number>();

  // Forward tournament update events
  tournamentUpdated = output<Tournament>();
  stageUpdated = output<TournamentStage>();

  activeTab = signal<TabId>('bracket');

  tabs = computed<Tab[]>(() => {
    const t = this.tournamentUI.tournament();
    if (!t) return [{ id: 'bracket' as TabId, label: 'Bracket' }];

    const baseTabs: Tab[] = [
      { id: 'bracket', label: 'Bracket' },
      { id: 'participants', label: 'Participants' }
    ];

    // Add standings tab for all formats when tournament is in progress or completed
    if (t.status === 'in_progress' || t.status === 'completed') {
      baseTabs.push({ id: 'standings', label: 'Standings' });
    }

    // Add settings tab only for organizers
    if (this.tournamentUI.isOrganizer()) {
      baseTabs.push({ id: 'settings', label: 'Settings' });
    }

    return baseTabs;
  });

  currentUserParticipant = computed(() => {
    const user = this.currentUser() ?? this.authService.user();
    if (!user) return null;
    return this.tournamentUI.participants().find(p => p.user_id === user.id) || null;
  });

  canSelfRegister = computed(() => {
    const t = this.tournamentUI.tournament();
    if (!t) return false;
    return t.registration_open && this.tournamentUI.isLoggedIn() && !this.tournamentUI.isOrganizer() && !this.currentUserParticipant();
  });

  setActiveTab(tabId: TabId): void {
    this.activeTab.set(tabId);
  }

  isActiveTab(tabId: TabId): boolean {
    return this.activeTab() === tabId;
  }

  // Event handlers
  onParticipantAdded(participant: Participant): void {
    this.participantAdded.emit(participant);
  }

  onParticipantRemoved(id: number): void {
    this.participantRemoved.emit(id);
  }

  onParticipantWithdrawn(id: number): void {
    this.participantWithdrawn.emit(id);
  }

  onSeedingChanged(participants: Participant[]): void {
    this.seedingChanged.emit(participants);
  }

  onSelfRegistered(participant: Participant): void {
    this.selfRegistered.emit(participant);
  }

  onLeft(id: number): void {
    this.left.emit(id);
  }

  onTournamentUpdated(tournament: Tournament): void {
    this.tournamentUpdated.emit(tournament);
  }

  onParticipantUpdated(participant: Participant): void {
    this.participantUpdated.emit(participant);
  }

  onStageUpdated(stage: TournamentStage): void {
    this.stageUpdated.emit(stage);
  }
}
