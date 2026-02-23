import { Component, input, computed, output, AfterViewInit, OnDestroy, ElementRef, ViewChild, signal } from '@angular/core';
import { BracketPreview, BracketMatch } from '../../services/bracket-generator.service';
import { Match, BracketStage, BracketType } from '../../models/bracket.model';
import { BracketViewer } from '../bracket-viewer/bracket-viewer';
import Panzoom, { PanzoomObject } from '@panzoom/panzoom';

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
  private fitToViewApplied = false;
  currentScale = signal(1);

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

  // Lifecycle hooks for panzoom
  ngAfterViewInit(): void {
    if (this.contentRef?.nativeElement) {
      this.initPanzoom();
    }
  }

  ngOnDestroy(): void {
    this.destroyPanzoom();
  }

  private initPanzoom(): void {
    const element = this.contentRef.nativeElement;

    this.panzoomInstance = Panzoom(element, {
      minScale: 0.25,
      maxScale: 3,
      contain: 'outside',
      excludeClass: 'panzoom-exclude',
      cursor: 'grab',
    });

    // Bind mouse wheel zoom (Ctrl+wheel or Shift+wheel)
    this.containerRef.nativeElement.addEventListener('wheel', this.handleWheel);

    // Track scale changes
    element.addEventListener('panzoomchange', this.handlePanzoomChange);

    // Auto-fit to view on load
    setTimeout(() => this.fitToView(), 0);
  }

  private destroyPanzoom(): void {
    if (this.panzoomInstance) {
      this.containerRef?.nativeElement.removeEventListener('wheel', this.handleWheel);
      this.contentRef?.nativeElement.removeEventListener('panzoomchange', this.handlePanzoomChange);
      this.panzoomInstance.destroy();
      this.panzoomInstance = null;
    }
  }

  private handleWheel = (event: WheelEvent): void => {
    if (event.ctrlKey || event.shiftKey) {
      event.preventDefault();
      this.panzoomInstance?.zoomWithWheel(event);
    }
  };

  private handlePanzoomChange = (event: Event): void => {
    const detail = (event as CustomEvent).detail;
    this.currentScale.set(detail.scale);
    console.log('[panzoom] Position:', { x: detail.x, y: detail.y, scale: detail.scale });
    // Log stack trace to find what's calling this
    console.trace('[panzoom] Change triggered by:');
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
    console.log('[fitToView] Called, fitToViewApplied:', this.fitToViewApplied);
    if (!this.panzoomInstance || !this.contentRef || !this.containerRef) return;

    const content = this.contentRef.nativeElement;
    const container = this.containerRef.nativeElement;

    // Calculate scale to fit entire bracket in view
    const scaleX = container.clientWidth / content.scrollWidth;
    const scaleY = container.clientHeight / content.scrollHeight;
    const fitScale = Math.min(scaleX, scaleY);

    // Use width-based scale if height-based would be below minScale (0.25)
    const minScale = 0.25;
    const actualFitScale = fitScale < minScale ? scaleX : fitScale;

    // Calculate scaled dimensions
    const scaledWidth = content.scrollWidth * actualFitScale;
    const scaledHeight = content.scrollHeight * actualFitScale;

    // The bracket content starts at approximately 39% into the content area
    // due to internal layout structure (headers, padding, flex positioning)
    const contentOffsetRatio = 0.39;
    const panX = -scaledWidth * contentOffsetRatio;
    const panY = -scaledHeight * contentOffsetRatio;

    console.log('[fitToView] Container:', {
      width: container.clientWidth,
      height: container.clientHeight
    });
    console.log('[fitToView] Content:', {
      scrollWidth: content.scrollWidth,
      scrollHeight: content.scrollHeight
    });
    console.log('[fitToView] Scale:', {
      scaleX,
      scaleY,
      fitScale,
      actualFitScale
    });
    console.log('[fitToView] Pan calculation:', {
      scaledWidth,
      scaledHeight,
      panX,
      panY
    });

    // Only apply zoom - panzoom automatically centers the view when zooming,
    // which positions the bracket correctly without needing manual pan
    this.panzoomInstance.zoom(actualFitScale, { animate: false });

    // Log final position
    const finalPan = this.panzoomInstance.getPan();
    const finalScale = this.panzoomInstance.getScale();
    console.log('[fitToView] After apply:', { finalPan, finalScale });
  }

  getZoomPercent(): string {
    return Math.round(this.currentScale() * 100) + '%';
  }
}
