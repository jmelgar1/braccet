export interface Participant {
  id: number;
  name: string;
  seed: number;
}

export interface SetScore {
  set_number: number;
  participant1_score: number;
  participant2_score: number;
}

export interface Match {
  id: number;
  round: number;
  position: number;
  participant1_id?: number;
  participant2_id?: number;
  participant1_name?: string;
  participant2_name?: string;
  sets: SetScore[];
  participant1_sets: number;
  participant2_sets: number;
  winner_id?: number;
  forfeit_winner_id?: number;
  status: string;
  next_match_id?: number;
}

export interface BracketState {
  tournament_id: number;
  total_rounds: number;
  current_round: number;
  is_complete: boolean;
  champion_id?: number;
  matches: Match[];
}

export interface CreateBracketRequest {
  tournament_id: number;
  format: string;
  participants: Participant[];
}

export interface ReportResultRequest {
  sets: SetScore[];
}
