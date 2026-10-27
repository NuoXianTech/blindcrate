import { BaseService } from "../core";
import type { BalanceItem, ListWalletTransactionsData, ListWalletTransactionsParams } from "./types";

export class WalletService extends BaseService {
  protected static readonly basePath = "/wallet";

  /** 当前用户的各货币余额, 已停用且没有余额的货币不返回 */
  static listBalances() {
    return this.get<{ balances: BalanceItem[] }>("/balances");
  }

  static listTransactions(params: ListWalletTransactionsParams) {
    return this.get<ListWalletTransactionsData>("/transactions", { params });
  }
}
