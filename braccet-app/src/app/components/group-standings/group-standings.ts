import { Component, input, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { GroupStanding } from '../../models/bracket.model';

@Component({
  selector: 'app-group-standings',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './group-standings.html',
  styleUrls: ['./group-standings.css']
})
export class GroupStandingsComponent {
  standings = input.required<GroupStanding[]>();
  advancingCount = input<number>();
  showRanking = input(true);

  isAdvancing(rank: number): boolean {
    const count = this.advancingCount();
    return count ? rank <= count : false;
  }

  formatRecord(wins: number, losses: number): string {
    return `${wins}-${losses}`;
  }

  formatDifferential(value: number): string {
    if (value > 0) return `+${value}`;
    return value.toString();
  }
}
