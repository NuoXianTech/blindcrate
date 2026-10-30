import type {AuctionAdminView} from '../auction';
import {BaseService, apiClient, type CodeLabel} from '../core';
import type {InventoryTransaction, ListUserItemsData, ListUserItemsParams} from '../inventory';
import type {ListTiersData, Tier} from '../ranked';
import type {Season} from '../season';
import type {BalanceItem, WalletTransaction} from '../wallet';
import type {
  AdjustBalanceRequest,
  AdjustInventoryRequest,
  AdminUser,
  AdminMatch,
  AdminQuality,
  AdminShopItem,
  AdminShopOrder,
  AdminVenue,
  ConditionType,
  CreateCurrencyRequest,
  ExportPackRequest,
  CreateQualityRequest,
  CreateSeasonRequest,
  Currency,
  CurrencyRequest,
  Item,
  ItemCategory,
  ItemEffect,
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
  ListPackImportsData,
  ListPackImportsParams,
  ListSeasonsData,
  ListSeasonsParams,
  ListShopItemsData,
  ListShopItemsParams,
  ListShopOrdersData,
  ListShopOrdersParams,
  ListSuspectsData,
  ListSuspectsParams,
  ListUsersData,
  ListUsersParams,
  ListVenuesData,
  ListVenuesParams,
  QualityRequest,
  ReviewSuspectRequest,
  SeasonRequest,
  SuspectReview,
  TierRequest,
  LootEntryRequest,
  LootPool,
  LootPoolEntriesData,
  LootPoolRequest,
  PackAction,
  PackImport,
  PackPreview,
  PackSectionInfo,
  ListSettingRevisionsData,
  ListSettingRevisionsParams,
  ListSettingsData,
  SettingGroup,
  SettingRevisionDetail,
  ShopItemRequest,
  Simulation,
  StatMetric,
  UpdateSettingRequest,
  UpdateUserRequest,
  VenueRequest,
} from './types';

/** 上传配置包的请求配置: 指定为 multipart 避免按默认的 JSON 序列化, 浏览器会补上分隔符; 配置包较大时较慢, 放宽超时 */
const PACK_UPLOAD = {headers: {'Content-Type': 'multipart/form-data'}, timeout: 120000};

/** 配置包和处理方式的上传表单 */
function packForm(file: File, actions: PackAction[]) {
  const form = new FormData();
  form.append('file', file);
  form.append('actions', JSON.stringify(actions));
  return form;
}

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

  /** 操作日志的操作类型和名称, 按后端登记的顺序 */
  static listAuditActions() {
    return this.get<{ actions: CodeLabel[] }>('/audit-logs/actions');
  }

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

  static listItems(params: ListItemsParams) {
    return this.get<ListItemsData>('/items', {params});
  }

  static createItem(body: ItemRequest) {
    return this.post<Item>('/items', body);
  }

  static updateItem(id: number, body: ItemRequest) {
    return this.put<Item>(`/items/${id}`, body);
  }

  /** 由后端代码注册的物品类别, 按注册顺序 */
  static listItemCategories() {
    return this.get<{ categories: ItemCategory[] }>('/items/categories');
  }

  /** 由后端代码注册的使用效果和参数的字段描述, 按注册顺序 */
  static listItemEffects() {
    return this.get<{ effects: ItemEffect[] }>('/items/effects');
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

  /** 全部段位, 按分数线从低到高 */
  static listTiers() {
    return this.get<ListTiersData>('/ranked/tiers');
  }

  /** 分数线不能与其他段位相同 */
  static createTier(body: TierRequest) {
    return this.post<Tier>('/ranked/tiers', body);
  }

  /** 段位按段位分实时计算, 修改分数线后所有人的段位随即变化 */
  static updateTier(id: number, body: TierRequest) {
    return this.put<Tier>(`/ranked/tiers/${id}`, body);
  }

  /** 段位分不受影响, 原来在这个段位的人按剩下的分数线重新计算段位 */
  static deleteTier(id: number) {
    return this.delete<null>(`/ranked/tiers/${id}`);
  }

  /** 排位的可疑名单, 其中一人低价成交多的排在前面; 统计天数和次数见排位设置 */
  static listSuspects(params: ListSuspectsParams) {
    return this.get<ListSuspectsData>('/ranked/suspects', {params});
  }

  /** 标记为已处理, 之后两人又一起打了排位时重新进入待处理 */
  static reviewSuspect(body: ReviewSuspectRequest) {
    return this.post<SuspectReview>('/ranked/suspects/review', body);
  }

  static listShopItems(params: ListShopItemsParams) {
    return this.get<ListShopItemsData>('/shop/items', {params});
  }

  static createShopItem(body: ShopItemRequest) {
    return this.post<AdminShopItem>('/shop/items', body);
  }

  /** 已有订单不受影响 */
  static updateShopItem(id: number, body: ShopItemRequest) {
    return this.put<AdminShopItem>(`/shop/items/${id}`, body);
  }

  /** 软删除, 代码不能再被新商品使用; 已有订单仍可退款 */
  static deleteShopItem(id: number) {
    return this.delete<null>(`/shop/items/${id}`);
  }

  /** 从新到旧 */
  static listShopOrders(params: ListShopOrdersParams) {
    return this.get<ListShopOrdersData>('/shop/orders', {params});
  }

  /** 退还价格并收回商品, 恢复库存, 不恢复限购数量; 用户的货币或物品不足以收回商品时失败 */
  static refundShopOrder(id: number) {
    return this.post<AdminShopOrder>(`/shop/orders/${id}/refund`);
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

  /** 全部配置种类, 按依赖排列 */
  static listPackSections() {
    return this.get<PackSectionInfo[]>('/packs/sections');
  }

  /** 导出配置包, 返回 zip 文件和下载时的文件名 */
  static async exportPack(body: ExportPackRequest) {
    // 图片较多时打包较慢, 放宽超时
    const res = await apiClient.post<Blob>(this.url('/packs/export'), body, {responseType: 'blob', timeout: 120000});
    const match = /filename="([^"]+)"/.exec(String(res.headers['content-disposition'] ?? ''));
    return {blob: res.data, filename: match?.[1] ?? 'blindcrate-pack.zip'};
  }

  /** 按处理方式试着导入, 返回每一项的比较结果, 不写入任何内容 */
  static previewPack(file: File, actions: PackAction[]) {
    return this.post<PackPreview>('/packs/preview', packForm(file, actions), PACK_UPLOAD);
  }

  /** 在一个事务中全部导入; 有未跳过的冲突项时返回 409, 错误的 data 为每一项的结果 */
  static importPack(file: File, actions: PackAction[]) {
    return this.post<PackImport>('/packs/import', packForm(file, actions), PACK_UPLOAD);
  }

  /** 从新到旧, 不含每一项的结果 */
  static listPackImports(params: ListPackImportsParams) {
    return this.get<ListPackImportsData>('/packs/imports', {params});
  }

  static getPackImport(id: number) {
    return this.get<PackImport>(`/packs/imports/${id}`);
  }

  /** 指定用户的各货币余额, 含已停用货币 */
  static listUserBalances(id: number) {
    return this.get<{ balances: BalanceItem[] }>(`/users/${id}/wallet/balances`);
  }

  static adjustUserBalance(id: number, body: AdjustBalanceRequest) {
    return this.post<WalletTransaction>(`/users/${id}/wallet/adjust`, body);
  }

  static listUserInventory(id: number, params: ListUserItemsParams) {
    return this.get<ListUserItemsData>(`/users/${id}/inventory/items`, {params});
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
  static getAuction(matchID: number) {
    return this.get<AuctionAdminView>(`/auctions/${matchID}`);
  }

  /** 取消进行中的对局, 玩法会退还入场代价并解冻资金; 原因记录在操作日志中 */
  static cancelMatch(id: number, reason: string) {
    return this.post<Omit<AdminMatch, 'players'>>(`/matches/${id}/cancel`, {reason});
  }
}
