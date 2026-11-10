import {BaseService} from '../core';
import type {ConditionType} from './types';

export class ConditionAdminService extends BaseService {
  protected static readonly basePath = '/admin';

  static listConditionTypes() {
    return this.get<{ types: ConditionType[] }>('/conditions/types');
  }
}
