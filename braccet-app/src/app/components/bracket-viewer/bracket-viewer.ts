import { Component, input, computed } from '@angular/core';
import { PreviewMatch, BracketPreview } from '../../services/bracket-generator.service';
import { Match } from '../../models/bracket.model';

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
export class BracketViewer {
  preview = input<BracketData | null>(null);
  isPreview = input(true);

  rounds = computed(() => {
    const p = this.preview();
    if (!p) return [];

    const roundsArray: { round: number; matches: DisplayMatch[] }[] = [];

    for (let r = 1; r <= p.totalRounds; r++) {
      const roundMatches = p.matches.filter(m => m.round === r);
      roundsArray.push({
        round: r,
        matches: roundMatches
      });
    }

    return roundsArray;
  });

  getRoundLabel(round: number): string {
    const total = this.preview()?.totalRounds ?? 0;
    if (round === total) return 'Final';
    if (round === total - 1) return 'Semifinals';
    if (round === total - 2) return 'Quarterfinals';
    return `Round ${round}`;
  }

  getParticipant1Display(match: DisplayMatch): string {
    if ('participant1Name' in match && match.participant1Name) {
      return match.participant1Name;
    }
    if ('participant1_name' in match && match.participant1_name) {
      return match.participant1_name;
    }
    if ('seed1' in match && match.seed1 > 0) {
      return `Seed ${match.seed1}`;
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
    if ('seed2' in match && match.seed2 > 0) {
      return `Seed ${match.seed2}`;
    }
    return 'TBD';
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

  isBye(match: DisplayMatch): boolean {
    if ('isBye' in match) {
      return match.isBye;
    }
    return false;
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
}
