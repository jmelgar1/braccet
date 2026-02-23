import { Component, input, computed, output, AfterViewInit, OnDestroy, ElementRef, ViewChild, signal } from '@angular/core';
import { PreviewMatch, BracketPreview } from '../../services/bracket-generator.service';
import { Match, BracketStage, BracketType } from '../../models/bracket.model';
import Panzoom, { PanzoomObject } from '@panzoom/panzoom';

type DisplayMatch = PreviewMatch | Match;

// Flexible bracket data type that accepts both preview and actual bracket
interface BracketData {
  totalRounds: number;
  matches: (PreviewMatch | Match)[];
}

@Component({
  selector: 'app-bracket-viewer',
  templateUrl: './bracket-viewer.html',
  styleUrl: './bracket-viewer.css'
})
export class BracketViewer implements AfterViewInit, OnDestroy {
  // ViewChild references for panzoom
  @ViewChild('panzoomContainer') containerRef!: ElementRef<HTMLElement>;
  @ViewChild('bracketGrid') bracketGridRef!: ElementRef<HTMLElement>;

  // Panzoom instance and state
  private panzoomInstance: PanzoomObject | null = null;
  currentScale = signal(1);

  preview = input<BracketData | null>(null);
  isPreview = input(true);
  isOrganizer = input(false);
  stages = input<BracketStage[]>([]);
  isLosersBracket = input(false);
  bracketType = input<BracketType>('winners');
  enablePanzoom = input(true);  // Set to false when used inside DoubleElimBracket

  matchClicked = output<Match>();
  matchReopened = output<Match>();
  matchEditClicked = output<Match>();
  matchDetailsClicked = output<{ match: DisplayMatch; event: MouseEvent }>();
  stageClicked = output<{ round: number; stage: BracketStage }>();

  // Modal state
  showDetailsModal = false;
  selectedMatch: DisplayMatch | null = null;
  modalPosition = { top: 0, left: 0 };

  // Lifecycle hooks for panzoom
  ngAfterViewInit(): void {
    if (this.enablePanzoom() && this.bracketGridRef?.nativeElement) {
      this.initPanzoom();
    }
  }

  ngOnDestroy(): void {
    this.destroyPanzoom();
  }

  private initPanzoom(): void {
    const element = this.bracketGridRef.nativeElement;
    const container = this.containerRef.nativeElement;

    console.log('[BracketViewer] initPanzoom() called');
    console.log('[BracketViewer] Element dimensions:', {
      scrollWidth: element.scrollWidth,
      scrollHeight: element.scrollHeight,
      clientWidth: element.clientWidth,
      clientHeight: element.clientHeight,
      offsetWidth: element.offsetWidth,
      offsetHeight: element.offsetHeight
    });
    console.log('[BracketViewer] Container dimensions:', {
      scrollWidth: container.scrollWidth,
      scrollHeight: container.scrollHeight,
      clientWidth: container.clientWidth,
      clientHeight: container.clientHeight,
      offsetWidth: container.offsetWidth,
      offsetHeight: container.offsetHeight
    });

    const panzoomConfig = {
      minScale: 0.25,
      maxScale: 3,
      excludeClass: 'panzoom-exclude',
      cursor: 'grab',
      disablePan: false,
      disableZoom: false,
    };
    console.log('[BracketViewer] Panzoom config:', panzoomConfig);

    this.panzoomInstance = Panzoom(element, panzoomConfig);
    console.log('[BracketViewer] Panzoom instance created');

    // Bind mouse wheel for zoom/pan (passive: false allows preventDefault)
    this.containerRef.nativeElement.addEventListener('wheel', this.handleWheel, { passive: false });
    console.log('[BracketViewer] Wheel event listener attached');

    // Track scale changes
    element.addEventListener('panzoomchange', this.handlePanzoomChange);
    console.log('[BracketViewer] Panzoomchange event listener attached');

    // Auto-fit to view on load (use setTimeout to ensure DOM is ready)
    console.log('[BracketViewer] Scheduling fitToView() via setTimeout');
    setTimeout(() => this.fitToView(), 0);
  }

  private destroyPanzoom(): void {
    console.log('[BracketViewer] destroyPanzoom() called, instance exists:', !!this.panzoomInstance);
    if (this.panzoomInstance) {
      this.containerRef?.nativeElement.removeEventListener('wheel', this.handleWheel);
      console.log('[BracketViewer] Wheel event listener removed');
      this.bracketGridRef?.nativeElement.removeEventListener('panzoomchange', this.handlePanzoomChange);
      console.log('[BracketViewer] Panzoomchange event listener removed');
      this.panzoomInstance.destroy();
      console.log('[BracketViewer] Panzoom instance destroyed');
      this.panzoomInstance = null;
    }
  }

  private handleWheel = (event: WheelEvent): void => {
    console.log('[BracketViewer] handleWheel() called:', {
      deltaX: event.deltaX,
      deltaY: event.deltaY,
      deltaZ: event.deltaZ,
      deltaMode: event.deltaMode,
      ctrlKey: event.ctrlKey,
      shiftKey: event.shiftKey,
      metaKey: event.metaKey,
      clientX: event.clientX,
      clientY: event.clientY,
      currentScale: this.currentScale()
    });

    event.preventDefault();

    if (event.ctrlKey || event.shiftKey) {
      // Zoom with Ctrl/Shift + wheel
      console.log('[BracketViewer] Zoom triggered via Ctrl/Shift+wheel');
      this.panzoomInstance?.zoomWithWheel(event);
    } else {
      // Pan with regular wheel scroll
      const currentPan = this.panzoomInstance?.getPan();
      console.log('[BracketViewer] Pan mode - current pan position:', currentPan);

      if (currentPan && this.panzoomInstance) {
        // deltaY for vertical scroll, deltaX for horizontal (trackpad)
        const panX = currentPan.x - event.deltaX;
        const panY = currentPan.y - event.deltaY;
        console.log('[BracketViewer] Calculating new pan position:', {
          currentX: currentPan.x,
          currentY: currentPan.y,
          deltaX: event.deltaX,
          deltaY: event.deltaY,
          newPanX: panX,
          newPanY: panY
        });
        this.panzoomInstance.pan(panX, panY, { animate: false });
        console.log('[BracketViewer] Pan applied');
      } else {
        console.log('[BracketViewer] Pan skipped - no currentPan or panzoomInstance');
      }
    }
  };

  private handlePanzoomChange = (event: Event): void => {
    const detail = (event as CustomEvent).detail;
    console.log('[BracketViewer] handlePanzoomChange() - panzoom state changed:', {
      scale: detail.scale,
      x: detail.x,
      y: detail.y,
      isSVG: detail.isSVG,
      originalEvent: detail.originalEvent?.type
    });
    this.currentScale.set(detail.scale);
  };

  // Public methods for zoom controls
  zoomIn(): void {
    console.log('[BracketViewer] zoomIn() called, current scale:', this.currentScale());
    this.panzoomInstance?.zoomIn();
  }

  zoomOut(): void {
    console.log('[BracketViewer] zoomOut() called, current scale:', this.currentScale());
    this.panzoomInstance?.zoomOut();
  }

  resetZoom(): void {
    console.log('[BracketViewer] resetZoom() called, current scale:', this.currentScale());
    this.panzoomInstance?.reset({ animate: true });
  }

  fitToView(): void {
    if (!this.panzoomInstance || !this.bracketGridRef || !this.containerRef) {
      return;
    }

    const grid = this.bracketGridRef.nativeElement;
    const container = this.containerRef.nativeElement;

    // For single elimination, fit to width - the bracket is horizontal
    // Use scrollWidth to get the full content width
    const scaleX = container.clientWidth / grid.scrollWidth;

    // Clamp to minScale
    const minScale = 0.25;
    const fitScale = Math.max(scaleX, minScale);

    this.panzoomInstance.zoom(fitScale, { animate: false });
  }

  getZoomPercent(): string {
    return Math.round(this.currentScale() * 100) + '%';
  }

  rounds = computed(() => {
    const p = this.preview();
    if (!p) return [];

    const roundsArray: { round: number; matches: DisplayMatch[]; isStraight: boolean }[] = [];

    for (let r = 1; r <= p.totalRounds; r++) {
      const roundMatches = p.matches.filter(m => m.round === r);
      const nextRoundMatches = p.matches.filter(m => m.round === r + 1);
      // "Straight" means same number of matches in current and next round (no merging)
      const isStraight = roundMatches.length === nextRoundMatches.length && roundMatches.length > 0;
      roundsArray.push({
        round: r,
        matches: roundMatches,
        isStraight
      });
    }

    return roundsArray;
  });

  getRoundLabel(round: number): string {
    // Check for custom stage name first
    const stagesData = this.stages();
    const stage = stagesData.find(s => s.round === round);
    if (stage?.stage_name) {
      return stage.stage_name;
    }

    // Fallback to computed name
    const total = this.preview()?.totalRounds ?? 0;
    const bt = this.bracketType();

    // Grand final
    if (bt === 'grand_final') {
      return 'Grand Final';
    }

    // Losers bracket has different naming
    if (this.isLosersBracket() || bt === 'losers') {
      if (round === total) return 'Losers Final';
      if (round === total - 1) return 'Losers Semis';
      return `Losers Round ${round}`;
    }

    // Winners bracket in double elimination
    if (bt === 'winners') {
      if (round === total) return 'Winners Final';
      if (round === total - 1) return 'Winners Semis';
      if (round === total - 2) return 'Winners Quarters';
      return `Winners Round ${round}`;
    }

    // Single elimination (default)
    if (round === total) return 'Final';
    if (round === total - 1) return 'Semifinals';
    if (round === total - 2) return 'Quarterfinals';
    return `Round ${round}`;
  }

  // Handle stage header click
  onStageHeaderClick(round: number): void {
    if (!this.isOrganizer() || this.isPreview()) return;

    const stage = this.stages().find(s => s.round === round);
    if (stage) {
      this.stageClicked.emit({ round, stage });
    }
  }

  // Check if stage header should be clickable
  isStageClickable(): boolean {
    return this.isOrganizer() && !this.isPreview() && this.stages().length > 0;
  }

  getParticipant1Display(match: DisplayMatch): string {
    if ('participant1Name' in match && match.participant1Name) {
      return match.participant1Name;
    }
    if ('participant1_name' in match && match.participant1_name) {
      return match.participant1_name;
    }
    if ('seed1' in match && match.seed1 && match.seed1 > 0) {
      return `Seed ${match.seed1}`;
    }
    // If this is a bye match and slot 1 is empty, show BYE
    if (this.isBye(match) && !this.hasParticipant1(match)) {
      return 'BYE';
    }
    return 'TBD';
  }

  getParticipant2Display(match: DisplayMatch): string {
    if ('participant2Name' in match && match.participant2Name) {
      return match.participant2Name;
    }
    if ('participant2_name' in match && match.participant2_name) {
      return match.participant2_name;
    }
    if ('seed2' in match && match.seed2 && match.seed2 > 0) {
      return `Seed ${match.seed2}`;
    }
    // If this is a bye match and slot 2 is empty, show BYE
    if (this.isBye(match) && !this.hasParticipant2(match)) {
      return 'BYE';
    }
    return 'TBD';
  }

  // Check if participant 1 slot has a participant
  private hasParticipant1(match: DisplayMatch): boolean {
    if ('participant1Name' in match && match.participant1Name) return true;
    if ('participant1_name' in match && match.participant1_name) return true;
    if ('participant1_id' in match && match.participant1_id) return true;
    return false;
  }

  // Check if participant 2 slot has a participant
  private hasParticipant2(match: DisplayMatch): boolean {
    if ('participant2Name' in match && match.participant2Name) return true;
    if ('participant2_name' in match && match.participant2_name) return true;
    if ('participant2_id' in match && match.participant2_id) return true;
    return false;
  }

  getSeed1(match: DisplayMatch): number | null {
    if ('seed1' in match) {
      return match.seed1 || null;
    }
    return null;
  }

  getSeed2(match: DisplayMatch): number | null {
    if ('seed2' in match) {
      return match.seed2 || null;
    }
    return null;
  }

  getIconURL1(match: DisplayMatch): string | null {
    // Check PreviewMatch format (camelCase)
    if ('participant1IconURL' in match && match.participant1IconURL) {
      return match.participant1IconURL;
    }
    // Check Match format (snake_case)
    if ('participant1_icon_url' in match && match.participant1_icon_url) {
      return match.participant1_icon_url;
    }
    return null;
  }

  getIconURL2(match: DisplayMatch): string | null {
    // Check PreviewMatch format (camelCase)
    if ('participant2IconURL' in match && match.participant2IconURL) {
      return match.participant2IconURL;
    }
    // Check Match format (snake_case)
    if ('participant2_icon_url' in match && match.participant2_icon_url) {
      return match.participant2_icon_url;
    }
    return null;
  }

  isBye(match: DisplayMatch): boolean {
    // PreviewMatch has explicit isBye flag
    if ('isBye' in match) {
      return match.isBye;
    }
    // For actual Match: BYE only occurs in round 1 when exactly one participant has an ID
    // Later rounds with one participant are just waiting for opponent (TBD), not BYE
    if (match.round !== 1) {
      return false;
    }
    // Note: backend uses omitempty, so missing participant won't have the property at all
    const hasP1 = 'participant1_id' in match && match.participant1_id != null;
    const hasP2 = 'participant2_id' in match && match.participant2_id != null;
    return (hasP1 && !hasP2) || (!hasP1 && hasP2);
  }

  isMatchTBD(match: DisplayMatch): boolean {
    const p1 = this.getParticipant1Display(match);
    const p2 = this.getParticipant2Display(match);
    return p1 === 'TBD' || p2 === 'TBD';
  }

  // Check if this match was won by forfeit
  isMatchForfeit(match: DisplayMatch): boolean {
    if ('forfeit_winner_id' in match) {
      return match.forfeit_winner_id != null;
    }
    return false;
  }

  // Check if participant in slot 1 was forfeited (withdrew)
  isParticipant1Forfeited(match: DisplayMatch): boolean {
    if (!('forfeit_winner_id' in match) || !match.forfeit_winner_id) {
      return false;
    }
    // The forfeited participant is the one who is NOT the forfeit winner
    if ('participant1_id' in match && match.participant1_id) {
      return match.participant1_id !== match.forfeit_winner_id;
    }
    return false;
  }

  // Check if participant in slot 2 was forfeited (withdrew)
  isParticipant2Forfeited(match: DisplayMatch): boolean {
    if (!('forfeit_winner_id' in match) || !match.forfeit_winner_id) {
      return false;
    }
    // The forfeited participant is the one who is NOT the forfeit winner
    if ('participant2_id' in match && match.participant2_id) {
      return match.participant2_id !== match.forfeit_winner_id;
    }
    return false;
  }

  // Type guard to check if this is an actual Match (not a preview)
  isActualMatch(match: DisplayMatch): match is Match {
    return 'id' in match && typeof match.id === 'number';
  }

  // Check if participant is the winner
  isWinner(match: DisplayMatch, participantId: number | undefined): boolean {
    if (!participantId) return false;
    if ('winner_id' in match && match.winner_id) {
      return match.winner_id === participantId;
    }
    if ('forfeit_winner_id' in match && match.forfeit_winner_id) {
      return match.forfeit_winner_id === participantId;
    }
    return false;
  }

  // Get participant 1 ID
  getParticipant1Id(match: DisplayMatch): number | undefined {
    if ('participant1_id' in match) {
      return match.participant1_id;
    }
    return undefined;
  }

  // Get participant 2 ID
  getParticipant2Id(match: DisplayMatch): number | undefined {
    if ('participant2_id' in match) {
      return match.participant2_id;
    }
    return undefined;
  }

  // Get participant 1 sets won (for display in bracket)
  getParticipant1Score(match: DisplayMatch): number | null {
    if ('participant1_sets' in match && match.participant1_sets !== undefined) {
      return match.participant1_sets;
    }
    return null;
  }

  // Get participant 2 sets won (for display in bracket)
  getParticipant2Score(match: DisplayMatch): number | null {
    if ('participant2_sets' in match && match.participant2_sets !== undefined) {
      return match.participant2_sets;
    }
    return null;
  }

  // Check if match is completed
  isCompleted(match: DisplayMatch): boolean {
    if ('status' in match) {
      return match.status === 'completed';
    }
    return false;
  }

  // Check if action area should be shown (not TBD, and not preview; BYE matches show "BYE" label)
  showActionArea(match: DisplayMatch): boolean {
    if (this.isPreview()) return false;
    if (this.isBye(match)) return true;
    return !this.isMatchTBD(match);
  }

  // Handle report button click
  onReportClick(match: DisplayMatch, event: Event): void {
    event.stopPropagation();
    if (this.isActualMatch(match)) {
      this.matchClicked.emit(match);
    }
  }

  // Check if reopen button should be shown for a match
  canReopenMatch(match: DisplayMatch): boolean {
    if (!this.isOrganizer()) return false;
    if (this.isPreview()) return false;
    if (this.isBye(match)) return false;
    return this.isCompleted(match);
  }

  // Handle reopen button click
  onReopenClick(match: DisplayMatch, event: Event): void {
    event.stopPropagation();
    if (this.isActualMatch(match)) {
      this.matchReopened.emit(match);
    }
  }

  // Check if edit button should be shown for a match
  canEditMatch(match: DisplayMatch): boolean {
    if (!this.isOrganizer()) return false;
    if (this.isPreview()) return false;
    if (this.isBye(match)) return false;
    return this.isCompleted(match);
  }

  // Handle edit button click
  onEditClick(match: DisplayMatch, event: Event): void {
    event.stopPropagation();
    if (this.isActualMatch(match)) {
      this.matchEditClicked.emit(match);
    }
  }

  // Handle details button click
  onDetailsClick(match: DisplayMatch, event: Event): void {
    event.stopPropagation();
    const mouseEvent = event as MouseEvent;

    // If panzoom is disabled, we're inside a parent with panzoom (e.g., double-elim-bracket)
    // Emit event to let parent handle the modal (outside the transform)
    if (!this.enablePanzoom()) {
      this.matchDetailsClicked.emit({ match, event: mouseEvent });
      return;
    }

    // Standalone mode: show modal locally
    this.selectedMatch = match;

    // Position modal centered in viewport
    const modalWidth = 400;
    const modalHeight = 300;

    this.modalPosition = {
      top: (window.innerHeight - modalHeight) / 2,
      left: (window.innerWidth - modalWidth) / 2
    };
    this.showDetailsModal = true;
  }

  // Close details modal
  closeDetailsModal(): void {
    this.showDetailsModal = false;
    this.selectedMatch = null;
  }

  // Get sets for display
  getSets(match: DisplayMatch): { p1: number; p2: number }[] {
    if ('sets' in match && Array.isArray(match.sets)) {
      return match.sets.map(s => ({
        p1: s.participant1_score,
        p2: s.participant2_score
      }));
    }
    return [];
  }
}
