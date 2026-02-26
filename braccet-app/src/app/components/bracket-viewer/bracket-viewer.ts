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
  styleUrls: ['../shared/bracket-common.css', './bracket-viewer.css']
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
      minScale: 0.1,  // Lowered to allow larger brackets to fit
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

    // Scale to fit full bracket width in viewport
    const scaleX = container.clientWidth / grid.scrollWidth;
    const fitScale = Math.max(scaleX, 0.1);  // Lower minimum for large brackets

    // Calculate pan to position top-left at (0, 0)
    // With transform-origin at center, scaling shifts the top-left corner
    // Formula: pan = dimension * (scale - 1) / (2 * scale)
    const panX = grid.scrollWidth * (fitScale - 1) / (2 * fitScale);
    const panY = grid.scrollHeight * (fitScale - 1) / (2 * fitScale);

    this.panzoomInstance.zoom(fitScale, { animate: false });
    this.panzoomInstance.pan(panX, panY, { animate: false });
  }

  getZoomPercent(): string {
    return Math.round(this.currentScale() * 100) + '%';
  }

  rounds = computed(() => {
    const p = this.preview();
    if (!p) return [];

    const bt = this.bracketType();
    const roundsArray: { round: number; matches: DisplayMatch[]; isStraight: boolean }[] = [];

    for (let r = 1; r <= p.totalRounds; r++) {
      let roundMatches = p.matches.filter(m => m.round === r);
      const nextRoundMatches = p.matches.filter(m => m.round === r + 1);

      // "Straight" means same number of matches in current and next round (no merging)
      // In losers bracket: L-R1/L-R2 are straight, L-R3/L-R4 are straight, etc.
      let isStraight = roundMatches.length === nextRoundMatches.length && roundMatches.length > 0;

      // For losers bracket "straight" rounds where current has fewer matches than next,
      // add spacer entries for missing positions to maintain alignment
      if (bt === 'losers' && r % 2 === 1 && nextRoundMatches.length > roundMatches.length) {
        // This round should have the same number of matches as the next round
        // but some were skipped. Add spacers for missing positions.
        const expectedCount = nextRoundMatches.length;

        const matchesWithSpacers: DisplayMatch[] = [];
        for (let pos = 1; pos <= expectedCount; pos++) {
          const existingMatch = roundMatches.find(m => m.position === pos);
          if (existingMatch) {
            matchesWithSpacers.push(existingMatch);
          } else {
            // Add a spacer match for the missing position
            matchesWithSpacers.push(this.createSpacerMatch(r, pos));
          }
        }
        roundMatches = matchesWithSpacers;

        // After adding spacers, this is now a straight round
        isStraight = true;
      }

      roundsArray.push({
        round: r,
        matches: roundMatches,
        isStraight
      });
    }

    return roundsArray;
  });

  // Create a spacer match for alignment purposes (invisible in rendering)
  private createSpacerMatch(round: number, position: number): DisplayMatch {
    return {
      round,
      position,
      bracket_type: 'losers' as any,
      seed1: 0,
      seed2: 0,
      isBye: false,
      isSpacer: true // Flag to identify spacer matches
    } as PreviewMatch & { isSpacer: boolean };
  }

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

    let stage = this.stages().find(s => s.round === round);

    // If no stage exists yet, create a default one so the modal can be opened
    // to preset configuration before matches are populated
    if (!stage) {
      stage = {
        tournament_id: 0, // Will be filled by the API
        bracket_type: this.bracketType(),
        round: round,
        stage_name: this.getRoundLabel(round),
        best_of: 1
      };
    }

    this.stageClicked.emit({ round, stage });
  }

  // Check if stage header should be clickable
  // Allow clicking even if no stages exist yet - organizers can preset configuration
  isStageClickable(): boolean {
    return this.isOrganizer() && !this.isPreview();
  }

  // Get best_of for a round
  getRoundBestOf(round: number): number {
    const stage = this.stages().find(s => s.round === round);
    return stage?.best_of ?? 1;
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
    // BYE is ONLY displayed in the bottom slot (participant 2), never in the top slot
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

  // Get first letter for fallback icon when no logo is available
  getParticipant1Initial(match: DisplayMatch): string {
    const name = this.getParticipant1Display(match);
    if (!name || name === 'TBD' || name === 'BYE' || name.startsWith('Seed ')) {
      return '';
    }
    return name.charAt(0).toUpperCase();
  }

  getParticipant2Initial(match: DisplayMatch): string {
    const name = this.getParticipant2Display(match);
    if (!name || name === 'TBD' || name === 'BYE' || name.startsWith('Seed ')) {
      return '';
    }
    return name.charAt(0).toUpperCase();
  }

  // Check if this is a spacer match (invisible placeholder for alignment)
  isSpacer(match: DisplayMatch): boolean {
    return 'isSpacer' in match && (match as any).isSpacer === true;
  }

  isBye(match: DisplayMatch): boolean {
    // Spacer matches are not BYEs
    if (this.isSpacer(match)) {
      return false;
    }

    // PreviewMatch has explicit isBye flag
    if ('isBye' in match) {
      return match.isBye;
    }

    const bt = this.bracketType();

    // Grand final is never a BYE
    if (bt === 'grand_final') {
      return false;
    }

    // For losers bracket: check if participant2_name is "BYE" in any round
    // Backend puts BYE in slot 2 for:
    // - L-R1: when one W-R1 source is a BYE
    // - L-R2+: when source L-R(n-1) match was skipped (cascade effect)
    if (bt === 'losers') {
      if ('participant2_name' in match && match.participant2_name === 'BYE') {
        return true;
      }
      return false;
    }

    // For winners bracket: BYE only occurs in round 1
    if (match.round !== 1) {
      return false;
    }

    // For winners bracket round 1: BYE when exactly one participant has an ID
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
  // BYE matches don't show winner styling - advancing due to BYE isn't a "win"
  isWinner(match: DisplayMatch, participantId: number | undefined): boolean {
    if (!participantId) return false;
    if (this.isBye(match)) return false;
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

  // Check if action area should be shown (not TBD, not preview, not BYE)
  showActionArea(match: DisplayMatch): boolean {
    if (this.isPreview()) return false;
    if (this.isBye(match)) return false;
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
