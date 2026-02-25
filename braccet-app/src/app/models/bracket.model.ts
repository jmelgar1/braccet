export interface Participant {
  id: number;
  name: string;
  seed: number;
  icon_url?: string;
}

export interface SetScore {
  set_number: number;
  participant1_score: number;
  participant2_score: number;
}

export type BracketType = 'winners' | 'losers' | 'grand_final' | 'swiss';

export interface Match {
  id: number;
  round: number;
  position: number;
  bracket_type: BracketType;
  stage_id?: number; // For multi-stage tournaments
  group_id?: number; // For group brackets
  participant1_id?: number;
  participant2_id?: number;
  participant1_name?: string;
  participant2_name?: string;
  participant1_icon_url?: string;
  participant2_icon_url?: string;
  seed1?: number;
  seed2?: number;
  sets: SetScore[];
  participant1_sets: number;
  participant2_sets: number;
  winner_id?: number;
  forfeit_winner_id?: number;
  status: string;
  next_match_id?: number;
  loser_match_id?: number;
}

export interface BracketStage {
  tournament_id: number;
  bracket_type: string;
  round: number;
  stage_name: string;
  best_of: number;
}

export type BracketFormat = 'single_elimination' | 'double_elimination' | 'swiss';

export interface BracketState {
  tournament_id: number;
  format: BracketFormat;
  total_rounds: number;
  winners_rounds?: number;
  losers_rounds?: number;
  current_round: number;
  is_complete: boolean;
  champion_id?: number;
  matches: Match[];
  stages: BracketStage[];
}

export interface UpdateStageRequest {
  stage_name?: string;
  best_of?: number;
  bracket_type?: BracketType;
}

export interface CreateBracketRequest {
  tournament_id: number;
  format: string;
  participants: Participant[];
  swiss_rounds?: number;
}

export interface ReportResultRequest {
  sets: SetScore[];
}

export interface EditResultResponse {
  match: Match;
  cascade_matches?: Match[];
  winner_changed: boolean;
}

// Swiss-specific types
export type SwissParticipantStatus = 'active' | 'advanced' | 'eliminated';

export interface SwissStanding {
  rank: number;
  participant_id: number;
  participant_name: string;
  icon_url?: string;
  seed: number;
  wins: number;
  losses: number;
  match_wins?: number; // Real wins excluding BYEs (for threshold tracking)
  game_wins: number;
  game_losses: number;
  opponent_wins: number;
  status?: SwissParticipantStatus;
}

export interface SwissBracketState {
  tournament_id: number;
  format: 'swiss';
  total_rounds: number;
  current_round: number;
  is_complete: boolean;
  wins_to_advance?: number;
  losses_to_eliminate?: number;
  standings: SwissStanding[];
  matches: Match[];
  stages: BracketStage[];
}

// Elimination standings types
export interface EliminationStanding {
  rank: number;
  participant_id: number;
  participant_name: string;
  icon_url?: string;
  seed: number;
  wins: number;
  losses: number;
  game_wins: number;
  game_losses: number;
  placement: string;
}

export interface EliminationStandingsResponse {
  tournament_id: number;
  format: 'single_elimination' | 'double_elimination';
  is_complete: boolean;
  standings: EliminationStanding[];
}

// Group bracket types for multi-stage tournaments

export interface GroupStanding {
  rank: number;
  participant_id: number;
  participant_name: string;
  icon_url?: string;
  seed: number;
  match_wins: number;
  match_losses: number;
  set_wins: number;
  set_losses: number;
  set_differential: number;
  points_scored: number;
  points_against: number;
  point_differential: number;
  opponent_match_wins: number;
}

export interface StageStanding {
  rank: number;
  participant_id: number;
  participant_name?: string;
  group_id: number;
  group_name?: string;
  group_rank: number;
  match_wins: number;
  match_losses: number;
  set_wins: number;
  set_losses: number;
  advances: boolean;
}

export interface GroupBracketState extends BracketState {
  stage_id: number;
  group_id: number;
  group_name: string;
  standings: GroupStanding[];
}

export interface MultiStageState {
  tournament_id: number;
  stages: StageState[];
  current_stage_id?: number;
}

export interface StageState {
  stage_id: number;
  stage_order: number;
  stage_type: 'group' | 'final';
  format: BracketFormat;
  is_active: boolean;
  is_complete: boolean;
  groups?: GroupState[];
  standings?: StageStanding[];
}

export interface GroupState {
  group_id: number;
  group_name: string;
  group_order: number;
  bracket?: GroupBracketState;
  standings?: GroupStanding[];
}
