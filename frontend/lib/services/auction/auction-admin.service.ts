import {BaseService} from '../core';
import type {AuctionAdminView} from './types';

export class AuctionAdminService extends BaseService {
  protected static readonly basePath = '/admin';

  /** 一局暗拍的公开部分和每个座位的暗拍数据 */
  static getAuction(matchID: number) {
    return this.get<AuctionAdminView>(`/auctions/${matchID}`);
  }
}
