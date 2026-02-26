export interface PRSystem {
  id: number;
  community_id: number;
  name: string;
  description?: string;
  achievements_weight: number;
  form_weight: number;
  lan_weight: number;
  achievement_decay_months: number;
  achievement_window_months: number;
  form_window_months: number;
  lan_results_count: number;
  is_default: boolean;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreatePRSystemRequest {
  name: string;
  description?: string;
  achievements_weight?: number;
  form_weight?: number;
  lan_weight?: number;
  achievement_decay_months?: number;
  achievement_window_months?: number;
  form_window_months?: number;
  lan_results_count?: number;
  is_default?: boolean;
}

export interface UpdatePRSystemRequest {
  name?: string;
  description?: string;
  achievements_weight?: number;
  form_weight?: number;
  lan_weight?: number;
  achievement_decay_months?: number;
  achievement_window_months?: number;
  form_window_months?: number;
  lan_results_count?: number;
  is_active?: boolean;
}

export interface MemberPowerRanking {
  id: number;
  member_id: number;
  member_name?: string;
  member_region?: string;
  member_icon_url?: string;
  pr_system_id: number;
  total_points: number;
  rank?: number;
  achievements_score: number;
  form_score: number;
  lan_score: number;
  form_wins: number;
  form_set_wins: number;
  form_set_losses: number;
  form_avg_opponent_rating?: number;
  form_best_win_rating?: number;
  lan_placements_count: number;
  last_calculated_at?: string;
  created_at: string;
  updated_at: string;
}

export type TournamentClass = 'major' | 'world_lan' | 'continental' | 'regional' | 'online';

export interface TournamentPlacement {
  id: number;
  member_id: number;
  pr_system_id: number;
  tournament_id: number;
  placement: number;
  participant_count: number;
  tournament_class: TournamentClass;
  prize_pool_usd?: number;
  is_bo3_playoffs: boolean;
  avg_opponent_rating?: number;
  is_lan: boolean;
  raw_points: number;
  completed_at: string;
  created_at: string;
}
