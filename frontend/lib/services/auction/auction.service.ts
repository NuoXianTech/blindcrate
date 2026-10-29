import {BaseService, wsClient} from '../core';
import {AUCTION_MESSAGES, type AuctionView, type BidRequest, type UsableItem, type UseRequest} from './types';

export class AuctionService extends BaseService {
  protected static readonly basePath = '/auctions';

  /** 自己视角的一局暗拍, 断线重连后用它恢复 */
  static getAuction(matchId: number) {
    return this.get<AuctionView>(`/${matchId}`);
  }

  /** 通过 WebSocket 出价, 返回出价后自己的视角; 被拒绝时抛出 ApiError, message 可以直接展示 */
  static bid(body: BidRequest) {
    return wsClient.request<AuctionView>(AUCTION_MESSAGES.bid, body);
  }

  /** 背包中这局可以用的道具, 只看地图规则和背包, 不检查阶段和剩余次数 */
  static listUsableItems(matchId: number) {
    return this.get<{ items: UsableItem[] }>(`/${matchId}/items`);
  }

  /** 通过 WebSocket 使用道具, 返回使用后自己的视角; 被拒绝时抛出 ApiError, message 可以直接展示 */
  static use(body: UseRequest) {
    return wsClient.request<AuctionView>(AUCTION_MESSAGES.use, body);
  }
}
