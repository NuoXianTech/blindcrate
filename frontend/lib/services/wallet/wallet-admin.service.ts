import {BaseService} from '../core';
import type {
  AdjustBalanceRequest,
  BalanceItem,
  CreateCurrencyRequest,
  Currency,
  CurrencyRequest,
  ListAdminWalletTransactionsData,
  ListAdminWalletTransactionsParams,
  WalletTransaction,
} from './types';

export class WalletAdminService extends BaseService {
  protected static readonly basePath = '/admin';

  static listCurrencies() {
    return this.get<{ currencies: Currency[] }>('/currencies');
  }

  static createCurrency(body: CreateCurrencyRequest) {
    return this.post<Currency>('/currencies', body);
  }

  /** 已被使用的货币不能修改小数位, 基础货币不能停用 */
  static updateCurrency(code: string, body: CurrencyRequest) {
    return this.put<Currency>(`/currencies/${encodeURIComponent(code)}`, body);
  }

  /** 指定用户的各货币余额, 含已停用货币 */
  static listUserBalances(id: number) {
    return this.get<{ balances: BalanceItem[] }>(`/users/${id}/wallet/balances`);
  }

  static adjustUserBalance(id: number, body: AdjustBalanceRequest) {
    return this.post<WalletTransaction>(`/users/${id}/wallet/adjust`, body);
  }

  static listWalletTransactions(params: ListAdminWalletTransactionsParams) {
    return this.get<ListAdminWalletTransactionsData>('/wallet/transactions', {params});
  }
}
