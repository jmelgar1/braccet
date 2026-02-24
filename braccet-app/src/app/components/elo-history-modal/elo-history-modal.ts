import { Component, input, output, signal, inject, effect } from '@angular/core';
import { DatePipe } from '@angular/common';
import { MemberEloRating, EloHistory } from '../../models/elo.model';
import { EloService } from '../../services/elo.service';

@Component({
  selector: 'app-elo-history-modal',
  imports: [DatePipe],
  templateUrl: './elo-history-modal.html',
  styleUrl: './elo-history-modal.css'
})
export class EloHistoryModal {
  private eloService = inject(EloService);

  communitySlug = input.required<string>();
  member = input.required<MemberEloRating>();
  systemId = input.required<number>();
  isOwnerOrAdmin = input<boolean>(false);

  close = output<void>();
  reverted = output<void>();

  history = signal<EloHistory[]>([]);
  loading = signal(true);
  error = signal('');
  reverting = signal(false);
  confirmRevertEntry = signal<EloHistory | null>(null);

  constructor() {
    effect(() => {
      const slug = this.communitySlug();
      const memberId = this.member().member_id;
      const systemId = this.systemId();
      if (slug && memberId && systemId) {
        this.loadHistory(slug, memberId, systemId);
      }
    });
  }

  private loadHistory(slug: string, memberId: number, systemId: number): void {
    this.loading.set(true);
    this.error.set('');

    this.eloService.getMemberHistory(slug, memberId, systemId, 100).subscribe({
      next: (history) => {
        this.history.set(history);
        this.loading.set(false);
      },
      error: (err) => {
        this.error.set(err.error?.error || 'Failed to load history');
        this.loading.set(false);
      }
    });
  }

  onRevertClick(entry: EloHistory): void {
    this.confirmRevertEntry.set(entry);
  }

  cancelRevert(): void {
    this.confirmRevertEntry.set(null);
  }

  confirmRevert(): void {
    const entry = this.confirmRevertEntry();
    if (!entry) return;

    this.reverting.set(true);
    this.error.set('');

    this.eloService.revertMemberElo(
      this.communitySlug(),
      this.member().member_id,
      this.systemId(),
      entry.id
    ).subscribe({
      next: () => {
        this.reverting.set(false);
        this.confirmRevertEntry.set(null);
        this.reverted.emit();
        this.close.emit();
      },
      error: (err) => {
        this.reverting.set(false);
        this.error.set(err.error?.error || 'Failed to revert ELO');
      }
    });
  }

  // Check if this is the oldest entry (cannot revert to oldest)
  isOldestEntry(index: number): boolean {
    return index === this.history().length - 1;
  }

  onBackdropClick(event: MouseEvent): void {
    if ((event.target as HTMLElement).classList.contains('modal-backdrop')) {
      this.close.emit();
    }
  }
}
