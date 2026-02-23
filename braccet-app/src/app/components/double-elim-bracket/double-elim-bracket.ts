import { Component, input, computed, output, AfterViewInit, OnDestroy, ElementRef, ViewChild, signal } from '@angular/core';
import { BracketPreview, BracketMatch, PreviewMatch } from '../../services/bracket-generator.service';
import { Match, BracketStage, BracketType } from '../../models/bracket.model';
import { BracketViewer } from '../bracket-viewer/bracket-viewer';
import Panzoom, { PanzoomObject } from '@panzoom/panzoom';

type DisplayMatch = PreviewMatch | Match;

interface BracketData {
  totalRounds: number;
  matches: BracketMatch[];
}

@Component({
  selector: 'app-double-elim-bracket',
  standalone: true,
  imports: [BracketViewer],
  templateUrl: './double-elim-bracket.html',
  styleUrl: './double-elim-bracket.css'
})
export class DoubleElimBracket implements AfterViewInit, OnDestroy {
  // ViewChild references for panzoom
  @ViewChild('panzoomContainer') containerRef!: ElementRef<HTMLElement>;
  @ViewChild('bracketContent') contentRef!: ElementRef<HTMLElement>;

  // Panzoom instance and state
  private panzoomInstance: PanzoomObject | null = null;
  currentScale = signal(1);

  // Modal state (rendered outside panzoom transform)
  showDetailsModal = false;
  selectedMatch: DisplayMatch | null = null;
  modalPosition = { top: 0, left: 0 };

  preview = input<BracketPreview | null>(null);
  isPreview = input(true);
  isOrganizer = input(false);
  stages = input<BracketStage[]>([]);

  matchClicked = output<Match>();
  matchReopened = output<Match>();
  matchEditClicked = output<Match>();
  stageClicked = output<{ round: number; stage: BracketStage; bracketType: BracketType }>();

  // Filter matches by bracket type
  winnersMatches = computed(() => {
    const p = this.preview();
    if (!p) return [];
    return p.matches.filter(m => this.getBracketType(m) === 'winners');
  });

  losersMatches = computed(() => {
    const p = this.preview();
    if (!p) return [];
    return p.matches.filter(m => this.getBracketType(m) === 'losers');
  });

  grandFinalMatches = computed(() => {
    const p = this.preview();
    if (!p) return [];
    return p.matches.filter(m => this.getBracketType(m) === 'grand_final');
  });

  // Create bracket data for each section
  winnersBracketData = computed((): BracketData | null => {
    const p = this.preview();
    if (!p) return null;
    const matches = this.winnersMatches();
    const maxRound = Math.max(...matches.map(m => m.round), 0);
    return {
      totalRounds: maxRound,
      matches
    };
  });

  losersBracketData = computed((): BracketData | null => {
    const p = this.preview();
    if (!p) return null;
    const matches = this.losersMatches();
    const maxRound = Math.max(...matches.map(m => m.round), 0);
    return {
      totalRounds: maxRound,
      matches
    };
  });

  grandFinalBracketData = computed((): BracketData | null => {
    const p = this.preview();
    if (!p) return null;
    const matches = this.grandFinalMatches();
    return {
      totalRounds: 1,
      matches
    };
  });

  // Filter stages by bracket type
  winnersStages = computed(() => {
    return this.stages().filter(s => s.bracket_type === 'winners');
  });

  losersStages = computed(() => {
    return this.stages().filter(s => s.bracket_type === 'losers');
  });

  grandFinalStages = computed(() => {
    return this.stages().filter(s => s.bracket_type === 'grand_final');
  });

  // Helper to get bracket type from match
  private getBracketType(match: BracketMatch): BracketType {
    if ('bracket_type' in match) {
      return match.bracket_type;
    }
    return 'winners'; // Default fallback
  }

  // Event handlers that forward to parent
  onMatchClicked(match: Match): void {
    this.matchClicked.emit(match);
  }

  onMatchReopened(match: Match): void {
    this.matchReopened.emit(match);
  }

  onMatchEditClicked(match: Match): void {
    this.matchEditClicked.emit(match);
  }

  onWinnersStageClicked(event: { round: number; stage: BracketStage }): void {
    this.stageClicked.emit({ ...event, bracketType: 'winners' });
  }

  onLosersStageClicked(event: { round: number; stage: BracketStage }): void {
    this.stageClicked.emit({ ...event, bracketType: 'losers' });
  }

  onGrandFinalStageClicked(event: { round: number; stage: BracketStage }): void {
    this.stageClicked.emit({ ...event, bracketType: 'grand_final' });
  }

  // Handle match details clicked from child bracket-viewer
  onMatchDetailsClicked(event: { match: DisplayMatch; event: MouseEvent }): void {
    this.selectedMatch = event.match;

    // Center modal in viewport
    const modalWidth = 400;
    const modalHeight = 300;

    this.modalPosition = {
      top: (window.innerHeight - modalHeight) / 2,
      left: (window.innerWidth - modalWidth) / 2
    };
    this.showDetailsModal = true;
  }

  closeDetailsModal(): void {
    this.showDetailsModal = false;
    this.selectedMatch = null;
  }

  // Helper methods for modal display
  getParticipant1Display(match: DisplayMatch): string {
    if ('participant1Name' in match && match.participant1Name) return match.participant1Name;
    if ('participant1_name' in match && match.participant1_name) return match.participant1_name;
    return 'TBD';
  }

  getParticipant2Display(match: DisplayMatch): string {
    if ('participant2Name' in match && match.participant2Name) return match.participant2Name;
    if ('participant2_name' in match && match.participant2_name) return match.participant2_name;
    return 'TBD';
  }

  getIconURL1(match: DisplayMatch): string | null {
    if ('participant1IconURL' in match && match.participant1IconURL) return match.participant1IconURL;
    if ('participant1_icon_url' in match && match.participant1_icon_url) return match.participant1_icon_url;
    return null;
  }

  getIconURL2(match: DisplayMatch): string | null {
    if ('participant2IconURL' in match && match.participant2IconURL) return match.participant2IconURL;
    if ('participant2_icon_url' in match && match.participant2_icon_url) return match.participant2_icon_url;
    return null;
  }

  isCompleted(match: DisplayMatch): boolean {
    return 'status' in match && match.status === 'completed';
  }

  isWinner(match: DisplayMatch, participantId: number | undefined): boolean {
    if (!participantId) return false;
    if ('winner_id' in match && match.winner_id) return match.winner_id === participantId;
    if ('forfeit_winner_id' in match && match.forfeit_winner_id) return match.forfeit_winner_id === participantId;
    return false;
  }

  getParticipant1Id(match: DisplayMatch): number | undefined {
    return 'participant1_id' in match ? match.participant1_id : undefined;
  }

  getParticipant2Id(match: DisplayMatch): number | undefined {
    return 'participant2_id' in match ? match.participant2_id : undefined;
  }

  getParticipant1Score(match: DisplayMatch): number | null {
    return 'participant1_sets' in match && match.participant1_sets !== undefined ? match.participant1_sets : null;
  }

  getParticipant2Score(match: DisplayMatch): number | null {
    return 'participant2_sets' in match && match.participant2_sets !== undefined ? match.participant2_sets : null;
  }

  getSets(match: DisplayMatch): { p1: number; p2: number }[] {
    if ('sets' in match && Array.isArray(match.sets)) {
      return match.sets.map(s => ({ p1: s.participant1_score, p2: s.participant2_score }));
    }
    return [];
  }

  // Lifecycle hooks for panzoom
  ngAfterViewInit(): void {
    console.log('[DoubleElimBracket] ngAfterViewInit - contentRef:', !!this.contentRef, 'containerRef:', !!this.containerRef);
    if (this.contentRef?.nativeElement && this.containerRef?.nativeElement) {
      this.initPanzoom();
    } else {
      console.error('[DoubleElimBracket] Missing refs - contentRef:', !!this.contentRef, 'containerRef:', !!this.containerRef);
    }
  }

  ngOnDestroy(): void {
    this.destroyPanzoom();
  }

  private initPanzoom(): void {
    const element = this.contentRef.nativeElement;
    const container = this.containerRef.nativeElement;

    console.log('[DoubleElimBracket] initPanzoom() called');
    console.log('[DoubleElimBracket] Content element dimensions:', {
      scrollWidth: element.scrollWidth,
      scrollHeight: element.scrollHeight,
      clientWidth: element.clientWidth,
      clientHeight: element.clientHeight,
      offsetWidth: element.offsetWidth,
      offsetHeight: element.offsetHeight
    });
    console.log('[DoubleElimBracket] Container dimensions:', {
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
    console.log('[DoubleElimBracket] Panzoom config:', panzoomConfig);

    this.panzoomInstance = Panzoom(element, panzoomConfig);
    console.log('[DoubleElimBracket] Panzoom instance created');

    // Bind mouse wheel for zoom/pan (passive: false allows preventDefault)
    this.containerRef.nativeElement.addEventListener('wheel', this.handleWheel, { passive: false });
    console.log('[DoubleElimBracket] Wheel event listener attached');

    // Track scale changes
    element.addEventListener('panzoomchange', this.handlePanzoomChange);
    console.log('[DoubleElimBracket] Panzoomchange event listener attached');

    // Auto-fit to view on load
    console.log('[DoubleElimBracket] Scheduling fitToView() via setTimeout');
    setTimeout(() => this.fitToView(), 0);
  }

  private destroyPanzoom(): void {
    console.log('[DoubleElimBracket] destroyPanzoom() called, instance exists:', !!this.panzoomInstance);
    if (this.panzoomInstance) {
      this.containerRef?.nativeElement.removeEventListener('wheel', this.handleWheel);
      console.log('[DoubleElimBracket] Wheel event listener removed');
      this.contentRef?.nativeElement.removeEventListener('panzoomchange', this.handlePanzoomChange);
      console.log('[DoubleElimBracket] Panzoomchange event listener removed');
      this.panzoomInstance.destroy();
      console.log('[DoubleElimBracket] Panzoom instance destroyed');
      this.panzoomInstance = null;
    }
  }

  private handleWheel = (event: WheelEvent): void => {
    console.log('[DoubleElimBracket] handleWheel() called:', {
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
      console.log('[DoubleElimBracket] Zoom triggered via Ctrl/Shift+wheel');
      this.panzoomInstance?.zoomWithWheel(event);
    } else {
      // Pan with regular wheel scroll
      const currentPan = this.panzoomInstance?.getPan();
      console.log('[DoubleElimBracket] Pan mode - current pan position:', currentPan);

      if (currentPan && this.panzoomInstance) {
        // deltaY for vertical scroll, deltaX for horizontal (trackpad)
        const panX = currentPan.x - event.deltaX;
        const panY = currentPan.y - event.deltaY;
        console.log('[DoubleElimBracket] Calculating new pan position:', {
          currentX: currentPan.x,
          currentY: currentPan.y,
          deltaX: event.deltaX,
          deltaY: event.deltaY,
          newPanX: panX,
          newPanY: panY
        });
        this.panzoomInstance.pan(panX, panY, { animate: false });
        console.log('[DoubleElimBracket] Pan applied');
      } else {
        console.log('[DoubleElimBracket] Pan skipped - no currentPan or panzoomInstance');
      }
    }
  };

  private handlePanzoomChange = (event: Event): void => {
    const detail = (event as CustomEvent).detail;
    console.log('[DoubleElimBracket] handlePanzoomChange() - panzoom state changed:', {
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
    console.log('[DoubleElimBracket] zoomIn() called, current scale:', this.currentScale());
    this.panzoomInstance?.zoomIn();
  }

  zoomOut(): void {
    console.log('[DoubleElimBracket] zoomOut() called, current scale:', this.currentScale());
    this.panzoomInstance?.zoomOut();
  }

  resetZoom(): void {
    console.log('[DoubleElimBracket] resetZoom() called, current scale:', this.currentScale());
    this.panzoomInstance?.reset({ animate: true });
  }

  fitToView(): void {
    console.log('[DoubleElimBracket] fitToView() called');
    console.log('[DoubleElimBracket] fitToView() - panzoomInstance exists:', !!this.panzoomInstance);
    console.log('[DoubleElimBracket] fitToView() - contentRef exists:', !!this.contentRef);
    console.log('[DoubleElimBracket] fitToView() - containerRef exists:', !!this.containerRef);

    if (!this.panzoomInstance || !this.contentRef || !this.containerRef) {
      console.log('[DoubleElimBracket] fitToView() - EARLY RETURN: missing refs');
      return;
    }

    const content = this.contentRef.nativeElement;
    const container = this.containerRef.nativeElement;

    console.log('[DoubleElimBracket] fitToView() - Content dimensions:', {
      scrollWidth: content.scrollWidth,
      scrollHeight: content.scrollHeight,
      clientWidth: content.clientWidth,
      clientHeight: content.clientHeight,
      offsetWidth: content.offsetWidth,
      offsetHeight: content.offsetHeight,
      boundingRect: content.getBoundingClientRect()
    });

    console.log('[DoubleElimBracket] fitToView() - Container dimensions:', {
      scrollWidth: container.scrollWidth,
      scrollHeight: container.scrollHeight,
      clientWidth: container.clientWidth,
      clientHeight: container.clientHeight,
      offsetWidth: container.offsetWidth,
      offsetHeight: container.offsetHeight,
      boundingRect: container.getBoundingClientRect()
    });

    // Calculate scale to fit entire bracket in view
    const scaleX = container.clientWidth / content.scrollWidth;
    const scaleY = container.clientHeight / content.scrollHeight;
    const fitScale = Math.min(scaleX, scaleY);

    console.log('[DoubleElimBracket] fitToView() - Scale calculations:', {
      scaleX,
      scaleY,
      fitScale,
      minScaleUsed: fitScale === scaleX ? 'scaleX' : 'scaleY'
    });

    // Use width-based scale if height-based would be below minScale (0.25)
    const minScale = 0.25;
    const actualFitScale = fitScale < minScale ? scaleX : fitScale;

    console.log('[DoubleElimBracket] fitToView() - Actual fit scale:', {
      minScale,
      fitScaleBelowMin: fitScale < minScale,
      actualFitScale,
      usedScaleType: fitScale < minScale ? 'scaleX (fallback)' : 'fitScale'
    });

    // Only apply zoom - panzoom automatically centers the view when zooming,
    // which positions the bracket correctly without needing manual pan
    console.log('[DoubleElimBracket] fitToView() - Applying zoom:', actualFitScale);
    this.panzoomInstance.zoom(actualFitScale, { animate: false });
    console.log('[DoubleElimBracket] fitToView() - Complete');
  }

  getZoomPercent(): string {
    return Math.round(this.currentScale() * 100) + '%';
  }
}
