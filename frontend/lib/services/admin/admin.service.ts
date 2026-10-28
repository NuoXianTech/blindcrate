import type {AuctionAdminView} from '../auction';
import {BaseService} from '../core';
import type {InventoryTransaction, ListUserItemsData, ListUserItemsParams} from '../inventory';
import type {BalanceItem, WalletTransaction} from '../wallet';
import type {
  AdjustBalanceRequest,
  AdjustInventoryRequest,
  AdminUser,
  AdminMatch,
  AdminQuality,
  AdminVenue,
  ConditionType,
  CreateQualityRequest,
  Currency,
  Item,
  ItemRequest,
  LeaderboardRequest,
  AdminLeaderboard,
  ListLeaderboardsData,
  ListLeaderboardsParams,
  ListAdminInventoryTransactionsData,
  ListAdminInventoryTransactionsParams,
  ListAdminWalletTransactionsData,
  ListAdminWalletTransactionsParams,
  ListAdminMatchesData,
  ListAdminMatchesParams,
  ListAuditLogsData,
  ListAuditLogsParams,
  ListItemsData,
  ListItemsParams,
  ListLootPoolsData,
  ListLootPoolsParams,
  ListUsersData,
  ListUsersParams,
  ListVenuesData,
  ListVenuesParams,
  QualityRequest,
  LootEntryRequest,
  LootPool,
  LootPoolEntriesData,
  LootPoolRequest,
  ListSettingRevisionsData,
  ListSettingRevisionsParams,
  ListSettingsData,
  SettingGroup,
  SettingRevisionDetail,
  Simulation,
  StatMetric,
  UpdateSettingRequest,
  UpdateUserRequest,
  VenueRequest,
} from './types';

export class AdminService extends BaseService {
  protected static readonly basePath = '/admin';

  static listUsers(params: ListUsersParams) {
    return this.get<ListUsersData>('/users', {params});
  }

  static updateUser(id: number, body: UpdateUserRequest) {
    return this.patch<Pick<AdminUser, 'id' | 'username' | 'nickname' | 'email' | 'avatar_url' | 'is_admin'>>(
        `/users/${id}`,
        body,
    );
  }

  /** 封禁到指定时间(RFC3339), 并让其所有设备下线 */
  static banUser(id: number, bannedUntil: string, reason: string) {
    return this.put<null>(`/users/${id}/ban`, {banned_until: bannedUntil, reason});
  }

  static unbanUser(id: number) {
    return this.delete<null>(`/users/${id}/ban`);
  }

  /** 停用或启用, 停用时让其所有设备下线 */
  static setUserDisabled(id: number, disabled: boolean) {
    return this.put<null>(`/users/${id}/disabled`, {disabled});
  }

  static listAuditLogs(params: ListAuditLogsParams) {
    return this.get<ListAuditLogsData>('/audit-logs', {params});
  }

  static listCurrencies() {
    return this.get<{ currencies: Currency[] }>('/wallet/currencies');
  }

  static listItems(params: ListItemsParams) {
    return this.get<ListItemsData>('/items', {params});
  }

  static createItem(body: ItemRequest) {
    return this.post<Item>('/items', body);
  }

  static updateItem(id: number, body: ItemRequest) {
    return this.put<Item>(`/items/${id}`, body);
  }

  /** 删除后已持有的玩家背包保留为占位物品, 返回删除时的持有人数 */
  static deleteItem(id: number) {
    return this.delete<{ holder_count: number }>(`/items/${id}`);
  }

  static listLootPools(params: ListLootPoolsParams) {
    return this.get<ListLootPoolsData>('/loot-pools', {params});
  }

  static createLootPool(body: LootPoolRequest) {
    return this.post<LootPool>('/loot-pools', body);
  }

  static updateLootPool(id: number, body: LootPoolRequest) {
    return this.put<LootPool>(`/loot-pools/${id}`, body);
  }

  /** 一同删除全部条目, 返回删除的条目数 */
  static deleteLootPool(id: number) {
    return this.delete<{ entry_count: number }>(`/loot-pools/${id}`);
  }

  static listLootPoolEntries(id: number) {
    return this.get<LootPoolEntriesData>(`/loot-pools/${id}/entries`);
  }

  /** 用提交的条目整体替换, 提交空数组清空 */
  static replaceLootPoolEntries(id: number, entries: LootEntryRequest[]) {
    return this.put<LootPoolEntriesData>(`/loot-pools/${id}/entries`, {entries});
  }

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

  /** 按未保存的配置模拟仓库, 不写入数据 */
  static simulateVenue(body: VenueRequest) {
    return this.post<Simulation>('/venues/simulate', body);
  }

  /** 全部品质, 附带使用每个品质的物品数 */
  static listQualities() {
    return this.get<{ qualities: AdminQuality[] }>('/qualities');
  }

  static createQuality(body: CreateQualityRequest) {
    return this.post<AdminQuality>('/qualities', body);
  }

  /** 新的扫描时长只影响之后开局的对局 */
  static updateQuality(level: number, body: QualityRequest) {
    return this.put<AdminQuality>(`/qualities/${level}`, body);
  }

  /** 还有物品使用该品质时不能删除 */
  static deleteQuality(level: number) {
    return this.delete<null>(`/qualities/${level}`);
  }

  static listLeaderboards(params: ListLeaderboardsParams) {
    return this.get<ListLeaderboardsData>('/leaderboards', {params});
  }

  /** 可以排名的指标: 计数指标和每种货币的净收益 */
  static listLeaderboardMetrics() {
    return this.get<{ metrics: StatMetric[] }>('/leaderboards/metrics');
  }

  static createLeaderboard(body: LeaderboardRequest) {
    return this.post<AdminLeaderboard>('/leaderboards', body);
  }

  /** 修改后缓存的前 N 名随即失效 */
  static updateLeaderboard(id: number, body: LeaderboardRequest) {
    return this.put<AdminLeaderboard>(`/leaderboards/${id}`, body);
  }

  /** 只删除榜单配置, 统计数据不受影响 */
  static deleteLeaderboard(id: number) {
    return this.delete<null>(`/leaderboards/${id}`);
  }

  static listConditionTypes() {
    return this.get<{ types: ConditionType[] }>('/conditions/types');
  }

  static listSettings(category?: string) {
    return this.get<ListSettingsData>('/settings', {params: {category}});
  }

  static getSetting(key: string) {
    return this.get<SettingGroup>(`/settings/${key}`);
  }

  /** 保存一个设置组, 立即生效, 返回保存后的设置组; 已被别人修改时返回 409 */
  static updateSetting(key: string, body: UpdateSettingRequest) {
    return this.put<SettingGroup>(`/settings/${key}`, body);
  }

  /** 设置组的历史版本, 从新到旧 */
  static listSettingRevisions(key: string, params: ListSettingRevisionsParams) {
    return this.get<ListSettingRevisionsData>(`/settings/${key}/revisions`, {params});
  }

  static getSettingRevision(key: string, version: number) {
    return this.get<SettingRevisionDetail>(`/settings/${key}/revisions/${version}`);
  }

  /** 回滚到 target 版本, 生成一个新版本 */
  static rollbackSetting(key: string, body: { version: number; target: number }) {
    return this.post<SettingGroup>(`/settings/${key}/rollback`, body);
  }

  /** 恢复默认值, 密钥字段会被清空 */
  static resetSetting(key: string, body: { version: number }) {
    return this.post<SettingGroup>(`/settings/${key}/reset`, body);
  }

  /** 用已保存的邮件设置发送测试邮件 */
  static testMail(to: string) {
    return this.post<null>('/settings/mail/test', {to});
  }

  /** 指定用户的各货币余额, 含已停用货币 */
  static listUserBalances(id: number) {
    return this.get<{ balances: BalanceItem[] }>(`/users/${id}/wallet/balances`);
  }

  static adjustUserBalance(id: number, body: AdjustBalanceRequest) {
    return this.post<WalletTransaction>(`/users/${id}/wallet/adjust`, body);
  }

  static listUserInventory(id: number, params: ListUserItemsParams) {
    return this.get<ListUserItemsData>(`/users/${id}/inventory`, {params});
  }

  static adjustUserInventory(id: number, body: AdjustInventoryRequest) {
    return this.post<InventoryTransaction>(`/users/${id}/inventory/adjust`, body);
  }

  static listWalletTransactions(params: ListAdminWalletTransactionsParams) {
    return this.get<ListAdminWalletTransactionsData>('/wallet/transactions', {params});
  }

  static listInventoryTransactions(params: ListAdminInventoryTransactionsParams) {
    return this.get<ListAdminInventoryTransactionsData>('/inventory/transactions', {params});
  }

  static listMatches(params: ListAdminMatchesParams) {
    return this.get<ListAdminMatchesData>('/matches', {params});
  }

  /** 一局暗拍的公开部分和每个座位的暗拍数据 */
  static getMatchAuction(id: number) {
    return this.get<AuctionAdminView>(`/matches/${id}/auction`);
  }

  /** 取消进行中的对局, 玩法会退还入场代价并解冻资金; 原因记录在操作日志中 */
  static cancelMatch(id: number, reason: string) {
    return this.post<Omit<AdminMatch, 'players'>>(`/matches/${id}/cancel`, {reason});
  }
}
