export interface EloSystem {
  id: number;
  community_id: number;
  name: string;
  description?: string;
  starting_rating: number;
  k_factor: number;
  floor_rating: number;
  provisional_games: number;
  provisional_k_factor: number;
  win_streak_enabled: boolean;
  win_streak_threshold: number;
  win_streak_bonus: number;
  decay_enabled: boolean;
  decay_days: number;
  decay_amount: number;
  decay_floor: number;
  is_default: boolean;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateEloSystemRequest {
  name: string;
  description?: string;
  starting_rating?: number;
  k_factor?: number;
  floor_rating?: number;
  provisional_games?: number;
  provisional_k_factor?: number;
  win_streak_enabled?: boolean;
  win_streak_threshold?: number;
  win_streak_bonus?: number;
  decay_enabled?: boolean;
  decay_days?: number;
  decay_amount?: number;
  decay_floor?: number;
  is_default?: boolean;
}

export interface UpdateEloSystemRequest {
  name?: string;
  description?: string;
  starting_rating?: number;
  k_factor?: number;
  floor_rating?: number;
  provisional_games?: number;
  provisional_k_factor?: number;
  win_streak_enabled?: boolean;
  win_streak_threshold?: number;
  win_streak_bonus?: number;
  decay_enabled?: boolean;
  decay_days?: number;
  decay_amount?: number;
  decay_floor?: number;
  is_active?: boolean;
}

export interface MemberEloRating {
  id: number;
  member_id: number;
  member_name?: string;
  elo_system_id: number;
  rating: number;
  games_played: number;
  games_won: number;
  current_win_streak: number;
  highest_rating: number;
  lowest_rating: number;
  last_game_at?: string;
  created_at: string;
  updated_at: string;
}

export type EloChangeType = 'match' | 'decay' | 'adjustment' | 'initial';

export interface EloHistory {
  id: number;
  member_id: number;
  elo_system_id: number;
  change_type: EloChangeType;
  rating_before: number;
  rating_change: number;
  rating_after: number;
  match_id?: number;
  tournament_id?: number;
  opponent_member_id?: number;
  opponent_display_name?: string;
  opponent_rating_before?: number;
  is_winner?: boolean;
  k_factor_used?: number;
  expected_score?: number;
  win_streak_bonus: number;
  notes?: string;
  created_at: string;
}
