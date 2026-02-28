export type TournamentFormat = 'single_elimination' | 'double_elimination' | 'swiss' | 'multi_stage';

export type TournamentClass = 'major' | 'world_lan' | 'continental' | 'regional' | 'online';

export type VenueType = 'lan' | 'online' | 'hybrid';

export type EventTournamentRole = 'qualifier' | 'main';

export type PrizeDistributionMode = 'percentage' | 'amount';

export interface PlacementTier {
  placement: string;  // e.g., "1st", "3rd-4th"
  low: number;        // Lower placement bound
  high: number;       // Upper placement bound
  value: number;      // Percentage or dollar amount based on mode
}

export interface PrizeDistribution {
  mode: PrizeDistributionMode;
  tiers: PlacementTier[];
  computed_amounts?: Record<string, number>; // Placement -> USD amount (for percentage mode)
}

export interface SuggestedPrizeTiersResponse {
  participant_count: number;
  tiers: PlacementTier[];
}

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
  participant_icons?: string[]; // First N participant icon URLs for preview
  registration_open: boolean;
  logo_url?: string;
  starts_at?: string;
  starts_at_tentative: boolean;
  tournament_class?: TournamentClass;
  prize_pool_usd?: number;
  prize_distribution?: PrizeDistribution;
  event_id?: number;
  event_role?: EventTournamentRole;
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
  tournament_class?: TournamentClass;
  prize_pool_usd?: number;
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
  expected_participants?: number; // Expected total participants in this stage (for prize tier calculation)
  swiss_rounds?: number;
  wins_to_advance?: number;
  losses_to_eliminate?: number;
  skip_finals?: boolean;
  is_active: boolean;
  is_complete: boolean;
  ranking_criteria?: RankingCriterion[];
  venue_type?: VenueType;
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
  expected_participants?: number;
  swiss_rounds?: number;
  wins_to_advance?: number;
  losses_to_eliminate?: number;
  skip_finals?: boolean;
  ranking_criteria?: RankingCriterion[];
  placement_matches?: boolean;
  placement_depth?: number;
  venue_type?: VenueType;
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
  tournament_class?: TournamentClass;
  prize_pool_usd?: number;
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

export interface RegionInviteMember {
  id: number;
  display_name: string;
  elo_rating?: number;
  icon_url?: string;
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
  tournament_class?: TournamentClass;
  prize_pool_usd?: number;
  prize_distribution?: {
    mode: PrizeDistributionMode;
    tiers: PlacementTier[];
  };
}

export interface UpdateStageRequest {
  format?: StageFormat;
  participants_per_group?: number;
  advancing_per_group?: number;
  expected_participants?: number;
  swiss_rounds?: number;
  wins_to_advance?: number;
  losses_to_eliminate?: number;
  skip_finals?: boolean;
  ranking_criteria?: RankingCriterion[];
  venue_type?: VenueType;
}

// Stage seed assignment types (for manual seeding before stage starts)
export interface StageSeedAssignment {
  id: number;
  stage_id: number;
  participant_id: number;
  target_group_order: number; // -1 means auto-assign via serpentine
  created_at: string;
}

export interface StageSeedInput {
  participant_id: number;
  target_group_order: number; // -1 means auto-assign via serpentine
}

export interface UpdateStageSeedsRequest {
  seeds: StageSeedInput[];
}

// Stage participant pool types (for assigning participants to start in specific stages)
export interface StagePoolEntry {
  participant_id: number;
  stage_id: number;
  stage_order: number;
}

export interface StagePoolResponse {
  entries: StagePoolEntry[];
}

export interface StagePoolConfigInput {
  stage_id: number;
  participant_ids: number[];
}

export interface UpdateStagePoolRequest {
  config: StagePoolConfigInput[];
}
