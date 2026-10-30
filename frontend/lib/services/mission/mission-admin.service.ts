import {BaseService} from '../core';
import type {AdminMission, ListMissionsData, ListMissionsParams, MissionOptions, MissionRequest} from './types';

/** 任务的后台接口 */
export class MissionAdminService extends BaseService {
  protected static readonly basePath = '/admin';

  /** 任务列表, 按周期、排序值和 ID 排列 */
  static listMissions(params: ListMissionsParams) {
    return this.get<ListMissionsData>('/missions', {params});
  }

  /** 可以选择的周期和指标 */
  static listOptions() {
    return this.get<MissionOptions>('/missions/options');
  }

  static createMission(body: MissionRequest) {
    return this.post<AdminMission>('/missions', body);
  }

  /** 修改任务, 需提交全部字段, 代码不能修改 */
  static updateMission(id: number, body: MissionRequest) {
    return this.put<AdminMission>(`/missions/${id}`, body);
  }

  /** 软删除, 已领取的奖励不收回 */
  static deleteMission(id: number) {
    return this.delete<null>(`/missions/${id}`);
  }
}
