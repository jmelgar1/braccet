import { Component, input, computed, output, signal, effect, AfterViewInit, OnDestroy, ElementRef, ViewChild } from '@angular/core';
import { Match, SwissBracketState, BracketStage } from '../../models/bracket.model';
import Panzoom, { PanzoomObject } from '@panzoom/panzoom';

@Component({
  selector: 'app-swiss-bracket',
  templateUrl: './swiss-bracket.html',
  styleUrls: ['../shared/bracket-common.css', './swiss-bracket.css']
})
export class SwissBracket implements AfterViewInit, OnDestroy {
  // ViewChild references for panzoom
  @ViewChild('panzoomContainer') containerRef!: ElementRef<HTMLElement>;
  @ViewChild('matchesContent') matchesContentRef!: ElementRef<HTMLElement>;

  // Panzoom instance and state
  private panzoomInstance: PanzoomObject | null = null;
  currentScale = signal(1);

  // Input data
  bracketState = input.required<SwissBracketState>();
  isPreview = input(false);
  isOrganizer = input(false);

  // Events
  matchClicked = output<Match>();
  matchReopened = output<Match>();
  matchEditClicked = output<Match>();
  stageClicked = output<{ round: number; stage: BracketStage }>();
  reseedRoundClicked = output<{ round: number }>();

  // Current selected round (1-indexed)
  selectedRound = signal(1);

  // Modal state for match details
  showDetailsModal = false;
  selectedMatch: Match | null = null;
  modalPosition = { top: 0, left: 0 };

  // Compute total rounds
  totalRounds = computed(() => this.bracketState().total_rounds);

  // Compute current round from backend
  currentRound = computed(() => this.bracketState().current_round);

  // Compute stages from bracket state
  stages = computed(() => this.bracketState().stages ?? []);

  // Generate round tabs
  roundTabs = computed(() => {
    const total = this.totalRounds();
    const current = this.currentRound();
    const stagesData = this.stages();
    const tabs: { round: number; label: string; status: 'completed' | 'current' | 'upcoming'; bestOf: number }[] = [];

    // For threshold mode (totalRounds = 0), generate tabs up to current round
    // For fixed rounds mode, generate tabs up to totalRounds
    const maxRound = total > 0 ? total : current;

    for (let r = 1; r <= maxRound; r++) {
      let status: 'completed' | 'current' | 'upcoming';
      if (r < current) {
        status = 'completed';
      } else if (r === current) {
        status = 'current';
      } else {
        status = 'upcoming';
      }

      // Find stage for this round to get best_of
      const stage = stagesData.find(s => s.round === r);

      tabs.push({
        round: r,
        label: `Round ${r}`,
        status,
        bestOf: stage?.best_of ?? 1
      });
    }

    return tabs;
  });

  // Get matches for the selected round
  roundMatches = computed(() => {
    const matches = this.bracketState().matches;
    const round = this.selectedRound();
    return matches.filter(m => m.round === round);
  });

  // Check if all matches in the selected round are completed
  isRoundComplete = computed(() => {
    const matches = this.roundMatches();
    if (matches.length === 0) return false;
    return matches.every(m => m.status === 'completed');
  });

  // Track the last known current round to detect changes
  private lastCurrentRound = 0;

  constructor() {
    // Auto-update selectedRound when currentRound changes (e.g., after advancing a round)
    effect(() => {
      const current = this.currentRound();
      if (current !== this.lastCurrentRound) {
        this.lastCurrentRound = current;
        this.selectedRound.set(current);
      }
    });
  }

  ngOnInit(): void {
    // Initial setup handled by effect
  }

  // Lifecycle hooks for panzoom
  ngAfterViewInit(): void {
    if (this.matchesContentRef?.nativeElement) {
      this.initPanzoom();
    }
  }

  ngOnDestroy(): void {
    this.destroyPanzoom();
  }

  private initPanzoom(): void {
    const element = this.matchesContentRef.nativeElement;

    const panzoomConfig = {
      minScale: 0.3,
      maxScale: 3,
      excludeClass: 'panzoom-exclude',
      cursor: 'grab',
      disablePan: false,
      disableZoom: false,
    };

    this.panzoomInstance = Panzoom(element, panzoomConfig);

    // Bind mouse wheel for zoom/pan (passive: false allows preventDefault)
    this.containerRef.nativeElement.addEventListener('wheel', this.handleWheel, { passive: false });

    // Track scale changes
    element.addEventListener('panzoomchange', this.handlePanzoomChange);

    // Auto-fit to view on load
    setTimeout(() => this.fitToView(), 0);
  }

  private destroyPanzoom(): void {
    if (this.panzoomInstance) {
      this.containerRef?.nativeElement.removeEventListener('wheel', this.handleWheel);
      this.matchesContentRef?.nativeElement.removeEventListener('panzoomchange', this.handlePanzoomChange);
      this.panzoomInstance.destroy();
      this.panzoomInstance = null;
    }
  }

  private handleWheel = (event: WheelEvent): void => {
    event.preventDefault();

    if (event.ctrlKey || event.shiftKey) {
      // Zoom with Ctrl/Shift + wheel
      this.panzoomInstance?.zoomWithWheel(event);
    } else {
      // Pan with regular wheel scroll
      const currentPan = this.panzoomInstance?.getPan();

      if (currentPan && this.panzoomInstance) {
        const panX = currentPan.x - event.deltaX;
        const panY = currentPan.y - event.deltaY;
        this.panzoomInstance.pan(panX, panY, { animate: false });
      }
    }
  };

  private handlePanzoomChange = (event: Event): void => {
    const detail = (event as CustomEvent).detail;
    this.currentScale.set(detail.scale);
  };

  // Public methods for zoom controls
  zoomIn(): void {
    this.panzoomInstance?.zoomIn();
  }

  zoomOut(): void {
    this.panzoomInstance?.zoomOut();
  }

  resetZoom(): void {
    this.panzoomInstance?.reset({ animate: true });
  }

  fitToView(): void {
    if (!this.panzoomInstance || !this.matchesContentRef || !this.containerRef) {
      return;
    }

    const grid = this.matchesContentRef.nativeElement;
    const container = this.containerRef.nativeElement;

    // Scale to fit content in viewport
    const scaleX = container.clientWidth / grid.scrollWidth;
    const scaleY = container.clientHeight / grid.scrollHeight;
    const fitScale = Math.min(Math.max(Math.min(scaleX, scaleY), 0.3), 1);

    // Calculate pan to position top-left
    const panX = grid.scrollWidth * (fitScale - 1) / (2 * fitScale);
    const panY = grid.scrollHeight * (fitScale - 1) / (2 * fitScale);

    this.panzoomInstance.zoom(fitScale, { animate: false });
    this.panzoomInstance.pan(panX, panY, { animate: false });
  }

  getZoomPercent(): string {
    return Math.round(this.currentScale() * 100) + '%';
  }

  selectRound(round: number): void {
    this.selectedRound.set(round);
    // Reset zoom when changing rounds
    setTimeout(() => this.fitToView(), 0);
  }

  // Handle stage tab click (for editing best_of)
  onStageTabClick(round: number, event: MouseEvent): void {
    // Only allow editing by organizers and not in preview mode
    if (!this.isOrganizer() || this.isPreview()) return;

    event.stopPropagation();

    // Find the stage for this round, or create a default one
    let stage = this.stages().find(s => s.round === round);

    // If no stage exists yet, create a default one so the modal can be opened
    // to preset configuration before matches are populated
    if (!stage) {
      stage = {
        tournament_id: this.bracketState().tournament_id,
        bracket_type: 'swiss',
        round: round,
        stage_name: `Round ${round}`,
        best_of: 1
      };
    }

    this.stageClicked.emit({ round, stage });
  }

  // Handle reseed button click
  onReseedRound(round: number, event: MouseEvent): void {
    // Only allow reseeding by organizers and not in preview mode
    if (!this.isOrganizer() || this.isPreview()) return;

    event.stopPropagation();
    this.reseedRoundClicked.emit({ round });
  }

  // Check if stage tab should show edit indicator
  // Allow clicking even if no stages exist yet - organizers can preset configuration
  canEditStage(): boolean {
    return this.isOrganizer() && !this.isPreview();
  }

  // Get best_of for current round
  getCurrentRoundBestOf(): number {
    const stage = this.stages().find(s => s.round === this.selectedRound());
    return stage?.best_of ?? 1;
  }

  // Match helper methods (similar to bracket-viewer)
  getParticipant1Display(match: Match): string {
    return match.participant1_name || 'TBD';
  }

  getParticipant2Display(match: Match): string {
    if (this.isBye(match)) {
      return 'BYE';
    }
    return match.participant2_name || 'TBD';
  }

  getIconURL1(match: Match): string | null {
    return match.participant1_icon_url || null;
  }

  getIconURL2(match: Match): string | null {
    return match.participant2_icon_url || null;
  }

  // Get first letter for fallback icon when no logo is available
  getParticipant1Initial(match: Match): string {
    const name = this.getParticipant1Display(match);
    if (!name || name === 'TBD' || name === 'BYE' || name.startsWith('Seed ')) {
      return '';
    }
    return name.charAt(0).toUpperCase();
  }

  getParticipant2Initial(match: Match): string {
    const name = this.getParticipant2Display(match);
    if (!name || name === 'TBD' || name === 'BYE' || name.startsWith('Seed ')) {
      return '';
    }
    return name.charAt(0).toUpperCase();
  }

  getSeed1(match: Match): number | null {
    return match.seed1 || null;
  }

  getSeed2(match: Match): number | null {
    return match.seed2 || null;
  }

  isBye(match: Match): boolean {
    // A bye match has participant1 but no participant2
    return !!match.participant1_id && !match.participant2_id;
  }

  isCompleted(match: Match): boolean {
    return match.status === 'completed';
  }

  isMatchTBD(match: Match): boolean {
    return !match.participant1_id || !match.participant2_id;
  }

  isWinner(match: Match, participantId?: number): boolean {
    if (!participantId || !match.winner_id) return false;
    return match.winner_id === participantId;
  }

  isMatchForfeit(match: Match): boolean {
    return !!match.forfeit_winner_id;
  }

  isParticipant1Forfeited(match: Match): boolean {
    return !!match.forfeit_winner_id && match.forfeit_winner_id === match.participant2_id;
  }

  isParticipant2Forfeited(match: Match): boolean {
    return !!match.forfeit_winner_id && match.forfeit_winner_id === match.participant1_id;
  }

  getParticipant1Score(match: Match): number {
    return match.participant1_sets;
  }

  getParticipant2Score(match: Match): number {
    return match.participant2_sets;
  }

  getSets(match: Match): { p1: number; p2: number }[] {
    if (!match.sets) return [];
    return match.sets.map(s => ({
      p1: s.participant1_score,
      p2: s.participant2_score
    }));
  }

  // Action handlers
  showActionArea(match: Match): boolean {
    // Show action area for all non-bye matches
    return !this.isBye(match);
  }

  canEditMatch(match: Match): boolean {
    return this.isOrganizer() && !this.isPreview() && this.isCompleted(match);
  }

  canReopenMatch(match: Match): boolean {
    return this.isOrganizer() && !this.isPreview() && this.isCompleted(match);
  }

  canReportMatch(match: Match): boolean {
    return !this.isPreview() && !this.isCompleted(match) && !this.isBye(match) && !this.isMatchTBD(match);
  }

  onReportClick(match: Match, event: MouseEvent): void {
    event.stopPropagation();
    if (this.isActualMatch(match) && this.canReportMatch(match)) {
      this.matchClicked.emit(match);
    }
  }

  onEditClick(match: Match, event: MouseEvent): void {
    event.stopPropagation();
    if (this.isActualMatch(match) && this.canEditMatch(match)) {
      this.matchEditClicked.emit(match);
    }
  }

  onReopenClick(match: Match, event: MouseEvent): void {
    event.stopPropagation();
    if (this.isActualMatch(match) && this.canReopenMatch(match)) {
      this.matchReopened.emit(match);
    }
  }

  onDetailsClick(match: Match, event: MouseEvent): void {
    event.stopPropagation();
    this.selectedMatch = match;
    this.showDetailsModal = true;

    // Position modal near the click
    const rect = (event.target as HTMLElement).getBoundingClientRect();
    this.modalPosition = {
      top: Math.min(rect.top, window.innerHeight - 400),
      left: Math.min(rect.left + 50, window.innerWidth - 420)
    };
  }

  closeDetailsModal(): void {
    this.showDetailsModal = false;
    this.selectedMatch = null;
  }

  private isActualMatch(match: Match): match is Match {
    return 'id' in match && typeof match.id === 'number';
  }
}
