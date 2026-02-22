import { Injectable } from '@angular/core';
import { Participant } from '../models/tournament.model';

export interface PreviewMatch {
  round: number;
  position: number;
  seed1: number;
  seed2: number;
  participant1Name?: string;
  participant2Name?: string;
  isBye: boolean;
}

export interface BracketPreview {
  totalRounds: number;
  bracketSize: number;
  matches: PreviewMatch[];
}

@Injectable({
  providedIn: 'root'
})
export class BracketGeneratorService {

  /**
   * Returns the smallest power of 2 >= participants.
   * Ported from bracket/internal/engine/seeding.go
   */
  calculateBracketSize(participants: number): number {
    if (participants <= 0) {
      return 0;
    }
    let size = 1;
    while (size < participants) {
      size *= 2;
    }
    return size;
  }

  /**
   * Returns the number of rounds needed for a bracket of given size.
   * Ported from bracket/internal/engine/seeding.go
   */
  totalRounds(bracketSize: number): number {
    if (bracketSize <= 1) {
      return 0;
    }
    let rounds = 0;
    let size = bracketSize;
    while (size > 1) {
      rounds++;
      size = Math.floor(size / 2);
    }
    return rounds;
  }

  /**
   * Returns seed pairings for a bracket.
   * Uses standard tournament seeding where top seeds meet in later rounds.
   * For an 8-player bracket: [[1,8], [4,5], [2,7], [3,6]]
   * This ensures 1 vs 2 can only happen in the finals.
   * Ported from bracket/internal/engine/seeding.go
   */
  generateSeedPairings(bracketSize: number): [number, number][] {
    if (bracketSize < 2) {
      return [];
    }
    return this.buildPairings(bracketSize);
  }

  private buildPairings(size: number): [number, number][] {
    if (size === 2) {
      return [[1, 2]];
    }

    // Get pairings for half-size bracket
    const smaller = this.buildPairings(size / 2);

    // For each match in smaller bracket, create two matches
    // Top half keeps the original seed, pairs with (size+1 - original opponent)
    // This maintains proper seeding where seed N faces seed (size+1-N)
    const result: [number, number][] = [];
    for (const pair of smaller) {
      // First match: original seed vs its proper opponent for this bracket size
      result.push([pair[0], size + 1 - pair[0]]);
      // Second match: original opponent vs its proper opponent
      result.push([pair[1], size + 1 - pair[1]]);
    }

    return result;
  }

  /**
   * Returns the number of matches in a given round (1-indexed).
   * Round 1 has bracketSize/2 matches, each subsequent round has half.
   */
  matchesInRound(bracketSize: number, round: number): number {
    if (round < 1 || bracketSize < 2) {
      return 0;
    }
    let matches = bracketSize / 2;
    for (let i = 1; i < round; i++) {
      matches = Math.floor(matches / 2);
    }
    return matches;
  }

  /**
   * Generates a bracket preview from a list of participants.
   * Participants should be ordered by seed (index 0 = seed 1).
   */
  generatePreview(participants: Participant[]): BracketPreview {
    const count = participants.length;
    if (count < 2) {
      return {
        totalRounds: 0,
        bracketSize: 0,
        matches: []
      };
    }

    const bracketSize = this.calculateBracketSize(count);
    const rounds = this.totalRounds(bracketSize);
    const pairings = this.generateSeedPairings(bracketSize);

    const matches: PreviewMatch[] = [];

    // Generate round 1 matches from seed pairings
    pairings.forEach((pair, index) => {
      const seed1 = pair[0];
      const seed2 = pair[1];

      // Check if seeds are within participant count (byes for higher seeds)
      const participant1 = seed1 <= count ? participants[seed1 - 1] : null;
      const participant2 = seed2 <= count ? participants[seed2 - 1] : null;

      const isBye = !participant1 || !participant2;

      matches.push({
        round: 1,
        position: index + 1,
        // Only show seed if participant exists (bye slots get 0)
        seed1: participant1 ? seed1 : 0,
        seed2: participant2 ? seed2 : 0,
        participant1Name: participant1?.display_name,
        participant2Name: participant2?.display_name,
        isBye
      });
    });

    // Generate placeholder matches for subsequent rounds
    for (let round = 2; round <= rounds; round++) {
      const matchCount = this.matchesInRound(bracketSize, round);
      for (let pos = 1; pos <= matchCount; pos++) {
        matches.push({
          round,
          position: pos,
          seed1: 0, // TBD
          seed2: 0, // TBD
          isBye: false
        });
      }
    }

    // Advance BYE winners to their next round matches
    this.advanceByeWinners(matches);

    return {
      totalRounds: rounds,
      bracketSize,
      matches
    };
  }

  /**
   * Advances participants who have BYEs to their next round matches.
   * Modifies matches array in place.
   */
  private advanceByeWinners(matches: PreviewMatch[]): void {
    // Find all BYE matches in round 1
    const byeMatches = matches.filter(m => m.round === 1 && m.isBye);

    for (const byeMatch of byeMatches) {
      // Determine the winner (the participant who exists)
      const winnerName = byeMatch.participant1Name || byeMatch.participant2Name;
      const winnerSeed = byeMatch.seed1 || byeMatch.seed2;

      if (!winnerName) continue;

      // Find the next round match
      // Position in next round: Math.ceil(position / 2)
      const nextPosition = Math.ceil(byeMatch.position / 2);
      const nextMatch = matches.find(m => m.round === 2 && m.position === nextPosition);

      if (!nextMatch) continue;

      // Determine which slot: odd positions -> slot 1, even positions -> slot 2
      if (byeMatch.position % 2 === 1) {
        nextMatch.participant1Name = winnerName;
        nextMatch.seed1 = winnerSeed;
      } else {
        nextMatch.participant2Name = winnerName;
        nextMatch.seed2 = winnerSeed;
      }
    }
  }
}
