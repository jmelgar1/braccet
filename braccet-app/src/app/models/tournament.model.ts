export type TournamentFormat = 'single_elimination' | 'double_elimination' | 'swiss' | 'multi_stage';

export interface Tournament {
  id: number;
  slug: string;
  organizer_id: number;
  community_id?: number;
  elo_system_id?: number;
  name: string;
  description?: string;
  game?: string;
  format: TournamentFormat;
  status: 'registration' | 'in_progress' | 'completed' | 'cancelled';
  max_participants?: number;
  swiss_rounds?: number;
  participant_count?: number;
  registration_open: boolean;
  starts_at?: string;
  starts_at_tentative: boolean;
  created_at: string;
  updated_at: string;
  // Multi-stage tournament fields
  stages?: TournamentStage[];
  current_stage?: TournamentStage;
}

export interface CreateTournamentRequest {
  name: string;
  description?: string;
  game?: string;
  format: 'single_elimination' | 'double_elimination' | 'swiss';
  max_participants?: number;
  swiss_rounds?: number;
  starts_at?: string;
  starts_at_tentative?: boolean;
  community_id?: number;
  elo_system_id?: number;
}

// Multi-stage tournament types

export type StageFormat = 'single_elimination' | 'double_elimination' | 'swiss';

export type RankingCriterion =
  | 'seed'
  | 'match_wins'
  | 'set_wins'
  | 'set_win_pct'
  | 'set_differential'
  | 'points_scored'
  | 'points_differential';

export interface TournamentStage {
  id: number;
  tournament_id: number;
  stage_order: number; // 1, 2, 3 for group stages; 0 for final
  stage_type: 'group' | 'final';
  format: StageFormat;
  participants_per_group?: number;
  advancing_per_group?: number;
  swiss_rounds?: number;
  skip_finals?: boolean;
  is_active: boolean;
  is_complete: boolean;
  ranking_criteria?: RankingCriterion[];
  created_at: string;
  updated_at: string;
}

export interface StageGroup {
  id: number;
  stage_id: number;
  group_name: string;
  group_order: number;
  participant_ids?: number[];
}

export interface StageConfigRequest {
  stage_order: number;
  format: StageFormat;
  participants_per_group?: number;
  advancing_per_group?: number;
  swiss_rounds?: number;
  skip_finals?: boolean;
  ranking_criteria?: RankingCriterion[];
  placement_matches?: boolean;
  placement_depth?: number;
}

export interface CreateMultiStageTournamentRequest {
  name: string;
  description?: string;
  game?: string;
  max_participants?: number;
  starts_at?: string;
  starts_at_tentative?: boolean;
  community_id?: number;
  elo_system_id?: number;
  group_stages: StageConfigRequest[];
  final_stage?: StageConfigRequest;
}

export interface Participant {
  id: number;
  tournament_id: number;
  user_id?: number;
  community_member_id?: number;
  display_name: string;
  icon_url?: string;
  region?: string;
  seed?: number;
  status: 'registered' | 'checked_in' | 'active' | 'eliminated' | 'disqualified' | 'withdrawn';
  checked_in_at?: string;
  elo_rating?: number;
  created_at: string;
}

export interface AddParticipantRequest {
  user_id?: number;
  community_member_id?: number;
  display_name: string;
}

export interface MemberSearchResult {
  id: number;
  display_name: string;
}

export interface UpdateSeedingRequest {
  seeds: Record<number, number>;
}

export interface UpdateTournamentRequest {
  name?: string;
  description?: string;
  game?: string;
  max_participants?: number;
  starts_at?: string;
  starts_at_tentative?: boolean;
  registration_open?: boolean;
  elo_system_id?: number;
}

export interface UpdateStageRequest {
  format?: StageFormat;
  participants_per_group?: number;
  advancing_per_group?: number;
  swiss_rounds?: number;
  skip_finals?: boolean;
  ranking_criteria?: RankingCriterion[];
}

// Stage seed assignment types (for manual seeding before stage starts)
export interface StageSeedAssignment {
  id: number;
  stage_id: number;
  participant_id: number;
  target_group_order: number;
  created_at: string;
}

export interface StageSeedInput {
  participant_id: number;
  target_group_order: number;
}

export interface UpdateStageSeedsRequest {
  seeds: StageSeedInput[];
}
