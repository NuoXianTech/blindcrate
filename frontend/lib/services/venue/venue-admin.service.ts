import {BaseService} from '../core';
import type {AdminVenue, ListVenuesData, ListVenuesParams, RecalibrateData, SimulateResult, VenueRequest} from './types';

export class VenueAdminService extends BaseService {
  protected static readonly basePath = '/admin';

  static listVenues(params: ListVenuesParams) {
    return this.get<ListVenuesData>('/venues', {params});
  }

  /** 保存时会模拟仓库, 摆放失败率过高时拒绝 */
  static createVenue(body: VenueRequest) {
    return this.post<AdminVenue>('/venues', body);
  }

  static updateVenue(id: number, body: VenueRequest) {
    return this.put<AdminVenue>(`/venues/${id}`, body);
  }

  static deleteVenue(id: number) {
    return this.delete<null>(`/venues/${id}`);
  }

  /** 按未保存的配置模拟仓库、推导拍卖参数并验算对局, 不写入数据; 验算未通过时不报错 */
  static simulateVenue(body: VenueRequest) {
    return this.post<SimulateResult>('/venues/simulate', body);
  }

  /** 按当前的玩法平衡设置重新推导全部地图; 启用的地图验算未通过时不保存 */
  static recalibrateVenues() {
    return this.post<RecalibrateData>('/venues/recalibrate');
  }
}
