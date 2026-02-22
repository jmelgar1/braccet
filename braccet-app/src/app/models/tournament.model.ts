export interface Tournament {
  id: number;
  slug: string;
  organizer_id: number;
  community_id?: number;
  elo_system_id?: number;
  name: string;
  description?: string;
  game?: string;
  format: 'single_elimination' | 'double_elimination';
  status: 'registration' | 'in_progress' | 'completed' | 'cancelled';
  max_participants?: number;
  registration_open: boolean;
  starts_at?: string;
  starts_at_tentative: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateTournamentRequest {
  name: string;
  description?: string;
  game?: string;
  format: 'single_elimination' | 'double_elimination';
  max_participants?: number;
  starts_at?: string;
  starts_at_tentative?: boolean;
  community_id?: number;
  elo_system_id?: number;
}

export interface Participant {
  id: number;
  tournament_id: number;
  user_id?: number;
  display_name: string;
  seed?: number;
  status: 'registered' | 'checked_in' | 'active' | 'eliminated' | 'disqualified' | 'withdrawn';
  checked_in_at?: string;
  created_at: string;
}

export interface AddParticipantRequest {
  user_id?: number;
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
