import {BaseService} from '../core';
import type {CrateOpenResult, CrateView} from './types';

export class CrateService extends BaseService {
  protected static readonly basePath = '/crates';

  /** 箱子、自己持有的数量和箱子中每一项的概率; 物品不是箱子或箱子暂时无法开启时失败 */
  static getCrate(itemID: number) {
    return this.get<CrateView>(`/${itemID}`);
  }

  /** 开启一个箱子, 结果在返回前已经结算; 幂等键相同的重复提交只开启一次, 返回原来的结果 */
  static open(itemID: number, idempotencyKey: string) {
    return this.post<CrateOpenResult>(`/${itemID}/open`, {idempotency_key: idempotencyKey});
  }
}
