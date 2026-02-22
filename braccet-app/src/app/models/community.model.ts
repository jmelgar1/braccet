export interface Community {
  id: number;
  slug: string;
  owner_id: number;
  name: string;
  description?: string;
  game?: string;
  avatar_url?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateCommunityRequest {
  name: string;
  description?: string;
  game?: string;
  avatar_url?: string;
}

export type MemberRole = 'owner' | 'admin' | 'member';

export interface CommunityMember {
  id: number;
  community_id: number;
  user_id?: number;
  display_name: string;
  role: MemberRole;
  is_ghost?: boolean;
  elo_rating?: number;
  ranking_points?: number;
  matches_played: number;
  matches_won: number;
  joined_at: string;
  created_at: string;
  updated_at: string;
}

export interface AddMemberRequest {
  display_name: string;
  user_id?: number;
}

export interface UpdateMemberRoleRequest {
  role: MemberRole;
}
