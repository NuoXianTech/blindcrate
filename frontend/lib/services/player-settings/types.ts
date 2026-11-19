export interface PlayerSettings {
  keep_one_collectible: boolean;
  version: number;
  updated_at: string | null;
}

export interface UpdatePlayerSettings {
  keep_one_collectible: boolean;
  version: number;
}

export const PLAYER_SETTINGS_CHANGED = 'player_settings:changed';
