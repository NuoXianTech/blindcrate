import {AuctionAdminService, AuctionService} from './auction';
import {AuditAdminService} from './audit';
import {AuthAdminService, AuthService} from './auth';
import {BotAdminService} from './bot';
import {CheckinAdminService, CheckinService} from './checkin';
import {CodexAdminService, CodexService} from './codex';
import {ConditionAdminService} from './condition';
import {InventoryAdminService, InventoryService} from './inventory';
import {ItemAdminService} from './item';
import {LeaderboardAdminService, LeaderboardService} from './leaderboard';
import {LootAdminService} from './loot';
import {MailboxAdminService, MailboxService} from './mailbox';
import {MatchAdminService, MatchService} from './match';
import {MissionAdminService, MissionService} from './mission';
import {PackAdminService} from './pack';
import {RankedAdminService, RankedService} from './ranked';
import {SeasonAdminService, SeasonService} from './season';
import {SettingsAdminService, SettingsService} from './settings';
import {ShopAdminService, ShopService} from './shop';
import {StatsService} from './stats';
import {UploadService} from './upload';
import {VenueAdminService, VenueService} from './venue';
import {WalletAdminService, WalletService} from './wallet';

/**
 * 服务层统一入口, 组件通过 services.xxx 调用接口
 *
 * 目录结构:
 *   /core - 基础工具
 *     - api-client.ts - Axios 实例和统一错误处理
 *     - base.service.ts - 服务基类, 自动剥掉响应外壳
 *     - types.ts - 通用类型
 *     - ws-client.ts - WebSocket 长连接, 订阅推送和发送请求
 *   /<模块> - 每个后端模块一个目录
 *     - xxx.service.ts - 玩家端接口, 继承 BaseService 并设置 basePath
 *     - xxx-admin.service.ts - 管理端接口, 沿用相同的业务归属
 *     - types.ts - 请求和响应类型
 *     - index.ts - 导出服务和类型
 *
 * 使用示例:
 *   import services from '@/lib/services';
 *   const {balances} = await services.wallet.listBalances();
 *
 * 新增模块: 按上面的结构建目录, 再在下面的 services 对象里注册
 */
const services = {
  /** 登录、注册、个人资料 */
  auth: AuthService,
  /** 钱包余额和流水 */
  wallet: WalletService,
  /** 背包物品和记录 */
  inventory: InventoryService,
  /** 地图 */
  venue: VenueService,
  /** 匹配和对局 */
  match: MatchService,
  /** 排行榜 */
  leaderboard: LeaderboardService,
  /** 赛季 */
  season: SeasonService,
  /** 段位分 */
  ranked: RankedService,
  /** 个人数据 */
  stats: StatsService,
  /** 商城 */
  shop: ShopService,
  /** 游戏邮件 */
  mailbox: MailboxService,
  /** 任务 */
  mission: MissionService,
  /** 每日签到 */
  checkin: CheckinService,
  /** 图鉴 */
  codex: CodexService,
  /** 公开设置 */
  settings: SettingsService,
  /** 图片上传 */
  upload: UploadService,
  /** 仓库暗拍 */
  auction: AuctionService,
  /** 后台: 暗拍详情 */
  auctionAdmin: AuctionAdminService,
  /** 后台: 操作日志 */
  auditAdmin: AuditAdminService,
  /** 后台: 用户 */
  authAdmin: AuthAdminService,
  /** 后台: 人机昵称和风格 */
  botAdmin: BotAdminService,
  /** 后台: 签到 */
  checkinAdmin: CheckinAdminService,
  /** 后台: 图鉴 */
  codexAdmin: CodexAdminService,
  /** 后台: 条件类型 */
  conditionAdmin: ConditionAdminService,
  /** 后台: 背包和流水 */
  inventoryAdmin: InventoryAdminService,
  /** 后台: 物品和品质 */
  itemAdmin: ItemAdminService,
  /** 后台: 榜单 */
  leaderboardAdmin: LeaderboardAdminService,
  /** 后台: 掉落池 */
  lootAdmin: LootAdminService,
  /** 后台: 游戏邮件 */
  mailboxAdmin: MailboxAdminService,
  /** 后台: 对局 */
  matchAdmin: MatchAdminService,
  /** 后台: 任务 */
  missionAdmin: MissionAdminService,
  /** 后台: 配置包 */
  packAdmin: PackAdminService,
  /** 后台: 段位和可疑名单 */
  rankedAdmin: RankedAdminService,
  /** 后台: 赛季 */
  seasonAdmin: SeasonAdminService,
  /** 后台: 系统设置 */
  settingsAdmin: SettingsAdminService,
  /** 后台: 商品和订单 */
  shopAdmin: ShopAdminService,
  /** 后台: 地图 */
  venueAdmin: VenueAdminService,
  /** 后台: 货币、余额和流水 */
  walletAdmin: WalletAdminService,
};

export default services;
