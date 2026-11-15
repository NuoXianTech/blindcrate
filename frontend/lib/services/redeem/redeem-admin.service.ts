import {BaseService} from '../core';
import type {
  CreateRedeemBatchRequest,
  ListRedeemBatchesData,
  ListRedeemBatchesParams,
  ListRedeemCodesData,
  ListRedeemRecordsData,
  ListRedeemRecordsParams,
  RedeemBatch,
  RedeemBatchRequest,
} from './types';

/** 兑换码的后台接口 */
export class RedeemAdminService extends BaseService {
  protected static readonly basePath = '/admin';

  /** 批次列表, 从新到旧, 不含已删除的 */
  static listBatches(params: ListRedeemBatchesParams) {
    return this.get<ListRedeemBatchesData>('/redeem/batches', {params});
  }

  /** 生成一个批次的兑换码, 幂等键相同的重复提交只生成一次 */
  static createBatch(body: CreateRedeemBatchRequest) {
    return this.post<RedeemBatch>('/redeem/batches', body);
  }

  /** 修改后只影响之后的兑换 */
  static updateBatch(id: number, body: RedeemBatchRequest) {
    return this.put<RedeemBatch>(`/redeem/batches/${id}`, body);
  }

  /** 软删除, 批次中的兑换码不能再兑换, 也不能再被新的批次使用 */
  static deleteBatch(id: number) {
    return this.delete<null>(`/redeem/batches/${id}`);
  }

  /** 批次的全部兑换码和各自已兑换的次数, 按生成顺序 */
  static listCodes(batchID: number) {
    return this.get<ListRedeemCodesData>(`/redeem/batches/${batchID}/codes`);
  }

  /** 兑换记录, 从新到旧, 包括已删除的批次的记录 */
  static listRecords(params: ListRedeemRecordsParams) {
    return this.get<ListRedeemRecordsData>('/redeem/records', {params});
  }
}
