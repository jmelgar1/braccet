import { Component, input, output, signal, computed } from '@angular/core';
import { Tournament, Participant } from '../../../../models/tournament.model';
import { User } from '../../../../services/auth.service';
import { BracketTab } from '../bracket-tab/bracket-tab';
import { ParticipantsTab } from '../participants-tab/participants-tab';
import { SettingsTab } from '../settings-tab/settings-tab';

type TabId = 'bracket' | 'participants' | 'settings';

interface Tab {
  id: TabId;
  label: string;
}

@Component({
  selector: 'app-side-panel',
  imports: [BracketTab, ParticipantsTab, SettingsTab],
  templateUrl: './side-panel.html'
})
export class SidePanel {
  tournament = input.required<Tournament>();
  participants = input.required<Participant[]>();
  isOrganizer = input.required<boolean>();
  isLoggedIn = input.required<boolean>();
  currentUser = input<User | null>(null);

  // Forward participant events
  participantAdded = output<Participant>();
  participantRemoved = output<number>();
  seedingChanged = output<Participant[]>();
  selfRegistered = output<Participant>();
  left = output<number>();

  // Forward tournament update events
  tournamentUpdated = output<Tournament>();

  activeTab = signal<TabId>('bracket');

  tabs = computed<Tab[]>(() => {
    const baseTabs: Tab[] = [
      { id: 'bracket', label: 'Bracket' },
      { id: 'participants', label: 'Participants' }
    ];

    // Add settings tab only for organizers
    if (this.isOrganizer()) {
      baseTabs.push({ id: 'settings', label: 'Settings' });
    }

    return baseTabs;
  });

  currentUserParticipant = computed(() => {
    const user = this.currentUser();
    if (!user) return null;
    return this.participants().find(p => p.user_id === user.id) || null;
  });

  canSelfRegister = computed(() => {
    const t = this.tournament();
    return t && t.registration_open && this.isLoggedIn() && !this.isOrganizer() && !this.currentUserParticipant();
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
}
