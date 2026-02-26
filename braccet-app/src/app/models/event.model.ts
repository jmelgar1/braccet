export type EventStatus = 'draft' | 'registration' | 'in_progress' | 'completed' | 'cancelled';
export type EventTournamentRole = 'qualifier' | 'main';

export interface Event {
  id: number;
  slug: string;
  community_id: number;
  organizer_id: number;
  name: string;
  description?: string;
  game?: string;
  status: EventStatus;
  starts_at?: string;
  logo_url?: string;
  tournaments?: EventTournament[];
  created_at: string;
  updated_at: string;
}

export interface EventTournament {
  id: number;
  tournament_id: number;
  tournament_slug: string;
  tournament_name: string;
  tournament_status: string;
  role: EventTournamentRole;
  display_order: number;
  advancing_count?: number;
  target_tournament_id?: number;
  participant_count: number;
  advanced_count?: number;
}

export interface CreateEventRequest {
  community_id: number;
  name: string;
  description?: string;
  game?: string;
  starts_at?: string;
}

export interface UpdateEventRequest {
  name?: string;
  description?: string;
  game?: string;
  status?: EventStatus;
  starts_at?: string;
}

export interface AddTournamentToEventRequest {
  tournament_id: number;
  role: EventTournamentRole;
  advancing_count?: number;
  target_tournament_id?: number;
}

export interface UpdateEventTournamentRequest {
  role?: EventTournamentRole;
  advancing_count?: number;
  target_tournament_id?: number;
  display_order?: number;
}

export interface QualifierStatus {
  tournament_id: number;
  tournament_name: string;
  tournament_slug: string;
  status: string;
  advancing_count?: number;
  advanced_ids?: number[];
}

export interface MainEventStatus {
  tournament_id: number;
  tournament_name: string;
  tournament_slug: string;
  status: string;
  registered_count: number;
  expected_from_quals: number;
  actual_from_quals: number;
}

export interface QualificationStatusResponse {
  qualifiers: QualifierStatus[];
  main_event?: MainEventStatus;
}

export interface AdvancementResponse {
  advanced_count: number;
  participants: AdvancedParticipant[];
}

export interface AdvancedParticipant {
  source_participant_id: number;
  target_participant_id: number;
  display_name: string;
  placement: number;
}

// DAG Editor types
export interface GraphNode {
  tournament_id: number;
  tournament_slug: string;
  tournament_name: string;
  role: EventTournamentRole;
  status: string;
  participant_count: number;
  advanced_count: number;
  advancing_count?: number;
  position_x: number;
  position_y: number;
}

export interface GraphEdge {
  source_tournament_id: number;
  target_tournament_id: number;
  advancing_count: number;
}

export interface EventGraphResponse {
  nodes: GraphNode[];
  edges: GraphEdge[];
}

export interface ConnectionUpdate {
  tournament_id: number;
  target_tournament_id?: number;
  advancing_count?: number;
  position_x: number;
  position_y: number;
}

export interface UpdateConnectionsRequest {
  connections: ConnectionUpdate[];
}
