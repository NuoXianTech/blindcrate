import {BaseService} from '../core';
import type {PlayerSettings, UpdatePlayerSettings} from './types';

export class PlayerSettingsService extends BaseService {
  protected static readonly basePath = '/player-settings';

  static read() {
    return this.get<PlayerSettings>('');
  }

  static update(body: UpdatePlayerSettings) {
    return this.patch<PlayerSettings>('', body);
  }
}
