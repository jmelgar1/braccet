import { Injectable, signal, computed } from '@angular/core';
import { Tournament, Participant, TournamentStage } from '../models/tournament.model';

/**
 * Shared UI state for tournament detail view.
 * Eliminates prop drilling between tournament-detail, side-panel, bracket-tab, and multi-stage-bracket.
 *
 * Usage:
 * - tournament-detail sets the core data (tournament, participants, stages, isOrganizer)
 * - Child components inject and read from signals
 * - Call reset() when navigating away from tournament detail
 */
@Injectable({ providedIn: 'root' })
export class TournamentUIService {
  // Core tournament data
  tournament = signal<Tournament | null>(null);
  participants = signal<Participant[]>([]);
  stages = signal<TournamentStage[]>([]);
  isOrganizer = signal(false);
  isLoggedIn = signal(false);

  // UI state for multi-stage bracket
  selectedStageId = signal<number | null>(null);

  // Refresh triggers
  bracketRefreshKey = signal(0);

  // Computed helpers
  isMultiStage = computed(() => this.tournament()?.format === 'multi_stage');
  isSwiss = computed(() => this.tournament()?.format === 'swiss');
  isInProgress = computed(() => {
    const status = this.tournament()?.status;
    return status === 'in_progress' || status === 'completed';
  });

  /** Trigger a bracket refresh in child components */
  refreshBracket(): void {
    this.bracketRefreshKey.update(k => k + 1);
  }

  /** Select a stage by ID (for multi-stage tournaments) */
  selectStage(stageId: number): void {
    this.selectedStageId.set(stageId);
  }

  /** Update tournament data */
  setTournament(tournament: Tournament | null): void {
    this.tournament.set(tournament);
    // Also update stages from tournament if present
    if (tournament?.stages) {
      this.stages.set(tournament.stages);
    }
  }

  /** Update participant in the list */
  updateParticipant(updated: Participant): void {
    this.participants.update(list =>
      list.map(p => p.id === updated.id ? updated : p)
    );
  }

  /** Add participant to the list */
  addParticipant(participant: Participant): void {
    this.participants.update(list => [...list, participant]);
  }

  /** Remove participant from the list */
  removeParticipant(participantId: number): void {
    this.participants.update(list => list.filter(p => p.id !== participantId));
  }

  /** Reset all state - call when navigating away from tournament */
  reset(): void {
    this.tournament.set(null);
    this.participants.set([]);
    this.stages.set([]);
    this.isOrganizer.set(false);
    this.isLoggedIn.set(false);
    this.selectedStageId.set(null);
    this.bracketRefreshKey.set(0);
  }
}
