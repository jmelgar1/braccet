import {
  Component,
  input,
  output,
  signal,
  computed,
  AfterViewInit,
  OnDestroy,
  ElementRef,
  ViewChild,
  effect
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import Panzoom, { PanzoomObject } from '@panzoom/panzoom';
import { GraphNode, GraphEdge, EventGraphResponse } from '../../models/event.model';

interface DragState {
  sourceId: number;
  startX: number;
  startY: number;
  currentX: number;
  currentY: number;
}

interface NodePosition {
  tournamentId: number;
  x: number;
  y: number;
  width: number;
  height: number;
}

@Component({
  selector: 'app-qualifier-dag-editor',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './qualifier-dag-editor.html',
  styleUrls: ['./qualifier-dag-editor.css']
})
export class QualifierDagEditor implements AfterViewInit, OnDestroy {
  @ViewChild('panzoomContainer') containerRef!: ElementRef<HTMLElement>;
  @ViewChild('dagCanvas') canvasRef!: ElementRef<HTMLElement>;
  @ViewChild('svgLayer') svgRef!: ElementRef<SVGElement>;

  // Inputs
  graph = input<EventGraphResponse | null>(null);
  isOrganizer = input(false);

  // Outputs
  connectionCreated = output<{ sourceId: number; targetId: number }>();
  connectionRemoved = output<{ sourceId: number; targetId: number }>();
  advancingCountChanged = output<{ sourceId: number; count: number }>();
  positionsChanged = output<{ tournamentId: number; x: number; y: number }[]>();

  // Internal state
  nodes = signal<GraphNode[]>([]);
  edges = signal<GraphEdge[]>([]);
  selectedEdge = signal<GraphEdge | null>(null);
  dragState = signal<DragState | null>(null);
  nodePositions = signal<Map<number, NodePosition>>(new Map());
  editingEdge = signal<GraphEdge | null>(null);
  editingAdvancingCount = signal<number>(0);

  // Panzoom
  private panzoomInstance: PanzoomObject | null = null;
  currentScale = signal(1);
  isPanning = signal(false);

  // Node dimensions
  readonly NODE_WIDTH = 200;
  readonly NODE_HEIGHT = 80;
  readonly NODE_PADDING = 40;

  constructor() {
    // React to graph input changes
    effect(() => {
      const graphData = this.graph();
      if (graphData) {
        this.nodes.set(graphData.nodes);
        this.edges.set(graphData.edges);
        this.initializeNodePositions();
      }
    });
  }

  ngAfterViewInit(): void {
    this.initPanzoom();
  }

  ngOnDestroy(): void {
    this.destroyPanzoom();
  }

  private initPanzoom(): void {
    if (!this.canvasRef?.nativeElement) return;

    const element = this.canvasRef.nativeElement;

    this.panzoomInstance = Panzoom(element, {
      maxScale: 2,
      minScale: 0.3,
      startScale: 0.8,
      contain: 'outside',
      cursor: 'grab',
      panOnlyWhenZoomed: false
    });

    // Listen for zoom changes
    element.addEventListener('panzoomchange', (e: any) => {
      this.currentScale.set(e.detail.scale);
    });

    // Handle wheel zoom
    this.containerRef.nativeElement.addEventListener('wheel', (e) => {
      this.panzoomInstance?.zoomWithWheel(e);
    });
  }

  private destroyPanzoom(): void {
    this.panzoomInstance?.destroy();
    this.panzoomInstance = null;
  }

  private initializeNodePositions(): void {
    const nodes = this.nodes();
    const positions = new Map<number, NodePosition>();

    // Group nodes by role
    const mainNodes = nodes.filter(n => n.role === 'main');
    const qualifierNodes = nodes.filter(n => n.role === 'qualifier');

    // Position main event on the right
    mainNodes.forEach((node, i) => {
      const x = node.position_x !== 0 ? node.position_x : 600;
      const y = node.position_y !== 0 ? node.position_y : 200 + i * (this.NODE_HEIGHT + this.NODE_PADDING);
      positions.set(node.tournament_id, {
        tournamentId: node.tournament_id,
        x,
        y,
        width: this.NODE_WIDTH,
        height: this.NODE_HEIGHT
      });
    });

    // Position qualifiers on the left
    qualifierNodes.forEach((node, i) => {
      const x = node.position_x !== 0 ? node.position_x : 100;
      const y = node.position_y !== 0 ? node.position_y : 100 + i * (this.NODE_HEIGHT + this.NODE_PADDING);
      positions.set(node.tournament_id, {
        tournamentId: node.tournament_id,
        x,
        y,
        width: this.NODE_WIDTH,
        height: this.NODE_HEIGHT
      });
    });

    this.nodePositions.set(positions);
  }

  // Get position for a node
  getNodeStyle(tournamentId: number): { [key: string]: string } {
    const pos = this.nodePositions().get(tournamentId);
    if (!pos) return {};
    return {
      left: `${pos.x}px`,
      top: `${pos.y}px`,
      width: `${this.NODE_WIDTH}px`,
      height: `${this.NODE_HEIGHT}px`
    };
  }

  // Calculate SVG path for an edge
  getEdgePath(edge: GraphEdge): string {
    const positions = this.nodePositions();
    const sourcePos = positions.get(edge.source_tournament_id);
    const targetPos = positions.get(edge.target_tournament_id);

    if (!sourcePos || !targetPos) return '';

    // Start from right edge of source, end at left edge of target
    const startX = sourcePos.x + sourcePos.width;
    const startY = sourcePos.y + sourcePos.height / 2;
    const endX = targetPos.x;
    const endY = targetPos.y + targetPos.height / 2;

    // Bezier curve control points
    const midX = (startX + endX) / 2;

    return `M ${startX} ${startY} C ${midX} ${startY}, ${midX} ${endY}, ${endX} ${endY}`;
  }

  // Get label position for an edge
  getEdgeLabelPosition(edge: GraphEdge): { x: number; y: number } {
    const positions = this.nodePositions();
    const sourcePos = positions.get(edge.source_tournament_id);
    const targetPos = positions.get(edge.target_tournament_id);

    if (!sourcePos || !targetPos) return { x: 0, y: 0 };

    const startX = sourcePos.x + sourcePos.width;
    const startY = sourcePos.y + sourcePos.height / 2;
    const endX = targetPos.x;
    const endY = targetPos.y + targetPos.height / 2;

    return {
      x: (startX + endX) / 2,
      y: (startY + endY) / 2 - 10
    };
  }

  // Dragging path for creating new connection
  getDraggingPath(): string {
    const state = this.dragState();
    if (!state) return '';

    const positions = this.nodePositions();
    const sourcePos = positions.get(state.sourceId);
    if (!sourcePos) return '';

    const startX = sourcePos.x + sourcePos.width;
    const startY = sourcePos.y + sourcePos.height / 2;
    const endX = state.currentX;
    const endY = state.currentY;
    const midX = (startX + endX) / 2;

    return `M ${startX} ${startY} C ${midX} ${startY}, ${midX} ${endY}, ${endX} ${endY}`;
  }

  // Node dragging
  onNodeDragStart(event: MouseEvent, tournamentId: number): void {
    if (!this.isOrganizer()) return;
    event.stopPropagation();

    const node = this.nodes().find(n => n.tournament_id === tournamentId);
    if (!node) return;

    // If it's a main event, don't allow connecting (they can't have outgoing connections)
    if (node.role === 'main') return;

    const rect = this.canvasRef.nativeElement.getBoundingClientRect();
    const scale = this.currentScale();

    this.dragState.set({
      sourceId: tournamentId,
      startX: event.clientX,
      startY: event.clientY,
      currentX: (event.clientX - rect.left) / scale,
      currentY: (event.clientY - rect.top) / scale
    });

    // Temporarily disable panzoom while dragging connection
    this.panzoomInstance?.setOptions({ disablePan: true });
  }

  onCanvasMouseMove(event: MouseEvent): void {
    const state = this.dragState();
    if (!state) return;

    const rect = this.canvasRef.nativeElement.getBoundingClientRect();
    const scale = this.currentScale();

    this.dragState.set({
      ...state,
      currentX: (event.clientX - rect.left) / scale,
      currentY: (event.clientY - rect.top) / scale
    });
  }

  onCanvasMouseUp(event: MouseEvent): void {
    this.panzoomInstance?.setOptions({ disablePan: false });
    this.dragState.set(null);
  }

  onNodeDrop(targetId: number): void {
    const state = this.dragState();
    if (!state || state.sourceId === targetId) {
      this.dragState.set(null);
      this.panzoomInstance?.setOptions({ disablePan: false });
      return;
    }

    // Check if connection already exists
    const existingEdge = this.edges().find(
      e => e.source_tournament_id === state.sourceId && e.target_tournament_id === targetId
    );

    if (!existingEdge) {
      this.connectionCreated.emit({ sourceId: state.sourceId, targetId });
    }

    this.dragState.set(null);
    this.panzoomInstance?.setOptions({ disablePan: false });
  }

  // Edge selection and editing
  onEdgeClick(edge: GraphEdge, event: MouseEvent): void {
    event.stopPropagation();
    if (this.selectedEdge()?.source_tournament_id === edge.source_tournament_id &&
        this.selectedEdge()?.target_tournament_id === edge.target_tournament_id) {
      this.selectedEdge.set(null);
    } else {
      this.selectedEdge.set(edge);
    }
  }

  onEdgeDoubleClick(edge: GraphEdge, event: MouseEvent): void {
    event.stopPropagation();
    if (!this.isOrganizer()) return;
    this.editingEdge.set(edge);
    this.editingAdvancingCount.set(edge.advancing_count);
  }

  saveAdvancingCount(): void {
    const edge = this.editingEdge();
    if (!edge) return;

    this.advancingCountChanged.emit({
      sourceId: edge.source_tournament_id,
      count: this.editingAdvancingCount()
    });
    this.editingEdge.set(null);
  }

  cancelEditAdvancingCount(): void {
    this.editingEdge.set(null);
  }

  deleteSelectedEdge(): void {
    const edge = this.selectedEdge();
    if (!edge) return;

    this.connectionRemoved.emit({
      sourceId: edge.source_tournament_id,
      targetId: edge.target_tournament_id
    });
    this.selectedEdge.set(null);
  }

  // Zoom controls
  zoomIn(): void {
    this.panzoomInstance?.zoomIn();
  }

  zoomOut(): void {
    this.panzoomInstance?.zoomOut();
  }

  resetView(): void {
    this.panzoomInstance?.reset();
  }

  // Get status badge class
  getStatusClass(status: string): string {
    switch (status) {
      case 'completed': return 'badge-success';
      case 'in_progress': return 'badge-warning';
      case 'registration': return 'badge-info';
      default: return 'badge-secondary';
    }
  }

  // Check if edge is selected
  isEdgeSelected(edge: GraphEdge): boolean {
    const selected = this.selectedEdge();
    return selected?.source_tournament_id === edge.source_tournament_id &&
           selected?.target_tournament_id === edge.target_tournament_id;
  }

  // Track by functions
  trackByTournamentId(index: number, node: GraphNode): number {
    return node.tournament_id;
  }

  trackByEdge(index: number, edge: GraphEdge): string {
    return `${edge.source_tournament_id}-${edge.target_tournament_id}`;
  }
}
