export interface Tournament {
  id: number;
  organizer_id: number;
  name: string;
  description?: string;
  game?: string;
  format: 'single_elimination' | 'double_elimination';
  status: 'draft' | 'registration' | 'in_progress' | 'completed' | 'cancelled';
  max_participants?: number;
  registration_open: boolean;
  starts_at?: string;
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
}
