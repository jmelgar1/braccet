import { Component, input, output, signal, inject, effect } from '@angular/core';
import { DatePipe, DecimalPipe } from '@angular/common';
import { MemberPowerRanking, TournamentPlacement, TournamentClass } from '../../models/power-ranking.model';
import { PowerRankingService } from '../../services/power-ranking.service';

@Component({
  selector: 'app-pr-history-modal',
  imports: [DatePipe, DecimalPipe],
  templateUrl: './pr-history-modal.html',
  styleUrl: './pr-history-modal.css'
})
export class PRHistoryModal {
  private prService = inject(PowerRankingService);

  communitySlug = input.required<string>();
  member = input.required<MemberPowerRanking>();
  systemId = input.required<number>();

  close = output<void>();

  placements = signal<TournamentPlacement[]>([]);
  loading = signal(true);
  error = signal('');

  constructor() {
    effect(() => {
      const slug = this.communitySlug();
      const memberId = this.member().member_id;
      const systemId = this.systemId();
      if (slug && memberId && systemId) {
        this.loadPlacements(slug, memberId, systemId);
      }
    });
  }

  private loadPlacements(slug: string, memberId: number, systemId: number): void {
    this.loading.set(true);
    this.error.set('');

    this.prService.getMemberPlacements(slug, memberId, systemId).subscribe({
      next: (placements) => {
        this.placements.set(placements);
        this.loading.set(false);
      },
      error: (err) => {
        this.error.set(err.error?.error || 'Failed to load placements');
        this.loading.set(false);
      }
    });
  }

  getPlacementDisplay(placement: number): string {
    if (placement === 1) return '1st';
    if (placement === 2) return '2nd';
    if (placement === 3) return '3rd';
    return `${placement}th`;
  }

  getTournamentClassLabel(cls: TournamentClass): string {
    switch (cls) {
      case 'major': return 'Major';
      case 'world_lan': return 'World LAN';
      case 'continental': return 'Continental';
      case 'regional': return 'Regional';
      case 'online': return 'Online';
      default: return cls;
    }
  }

  getTournamentClassBadgeClass(cls: TournamentClass): string {
    switch (cls) {
      case 'major': return 'class-badge class-major';
      case 'world_lan': return 'class-badge class-world-lan';
      case 'continental': return 'class-badge class-continental';
      case 'regional': return 'class-badge class-regional';
      default: return 'class-badge class-online';
    }
  }

  onBackdropClick(event: MouseEvent): void {
    if ((event.target as HTMLElement).classList.contains('modal-backdrop')) {
      this.close.emit();
    }
  }
}
