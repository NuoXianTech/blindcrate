import {BaseService} from '../core';
import type {CreateSeasonRequest, ListSeasonsData, ListSeasonsParams, Season, SeasonRequest} from './types';

export class SeasonAdminService extends BaseService {
  protected static readonly basePath = '/admin';

  static listSeasons(params: ListSeasonsParams) {
    return this.get<ListSeasonsData>('/seasons', {params});
  }

  /** 时间不能与其他赛季重叠 */
  static createSeason(body: CreateSeasonRequest) {
    return this.post<Season>('/seasons', body);
  }

  /** 开始后不能修改开始时间, 结束后只能修改名称 */
  static updateSeason(id: number, body: SeasonRequest) {
    return this.put<Season>(`/seasons/${id}`, body);
  }

  /** 只能删除未开始的赛季 */
  static deleteSeason(id: number) {
    return this.delete<null>(`/seasons/${id}`);
  }

  /** 把进行中的赛季的结束时间改为当前时间 */
  static endSeason(id: number) {
    return this.post<Season>(`/seasons/${id}/end`);
  }
}
