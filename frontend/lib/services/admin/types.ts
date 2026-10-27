import type {PageParams} from '../core';
import type {InventoryTransaction} from '../inventory';
import type {WalletTransaction} from '../wallet';

/** 管理端看到的完整用户, 对应后端 auth.User */
export interface AdminUser {
  id: number;
  username: string;
  email: string;
  email_verified_at: string | null;
  nickname: string;
  avatar_url: string;
  is_disabled: boolean;
  is_admin: boolean;
  banned_until: string | null;
  ban_reason: string;
  last_login_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface ListUsersParams extends PageParams {
  /** 按用户名或邮箱前缀搜索 */
  keyword?: string;
  is_admin?: boolean;
  is_disabled?: boolean;
  is_banned?: boolean;
}

export interface ListUsersData {
  total: number;
  users: AdminUser[];
}

export interface UpdateUserRequest {
  username?: string;
  nickname?: string;
  avatar_url?: string;
}

export type AuditScope = 'user' | 'admin';

export interface AuditLog {
  id: number;
  actor_id: number;
  /** 未登录或用户已不存在时为空 */
  actor_username: string;
  scope: AuditScope;
  action: string;
  target_type: string;
  target_id: number;
  detail: Record<string, unknown> | null;
  ip: string;
  user_agent: string;
  created_at: string;
}

export interface ListAuditLogsParams extends PageParams {
  actor_id?: number;
  scope?: AuditScope;
  action?: string;
  target_type?: string;
  target_id?: number;
  /** RFC3339, 包含 */
  start_time?: string;
  /** RFC3339, 不包含 */
  end_time?: string;
}

export interface ListAuditLogsData {
  total: number;
  logs: AuditLog[];
}

/** 货币定义, 对应后端 wallet.Currency */
export interface Currency {
  code: string;
  name: string;
  /** 展示用小数位, 金额统一按最小单位的整数存储 */
  decimals: number;
  sort_order: number;
  is_enabled: boolean;
  created_at: string;
  updated_at: string;
}

/** 物品定义, 对应后端 item.Item */
export interface Item {
  id: number;
  name: string;
  description: string;
  /** 1-7, 见 lib/quality.ts */
  quality: number;
  image_url: string;
  sell_currency: string;
  /** 官方回收价, 最小单位; 0 表示不可卖出 */
  sell_price: number;
  /** 掉落权重, 0 表示不会掉落 */
  weight: number;
  /** 在保险柜中占的格数 */
  width: number;
  height: number;
  /** 下架后不会掉落, 也不能再发放, 已有的保留 */
  is_enabled: boolean;
  created_at: string;
  updated_at: string;
}

/** 管理端物品列表项, 附带持有人数 */
export interface AdminItem extends Item {
  holder_count: number;
}

export interface ListItemsParams extends PageParams {
  /** 按名称搜索, 不区分大小写 */
  keyword?: string;
  quality?: number;
  is_enabled?: boolean;
}

export interface ListItemsData {
  total: number;
  items: AdminItem[];
}

/** 新建或修改物品, 修改时也需提交全部字段 */
export type ItemRequest = Pick<
  Item,
  'name' | 'description' | 'quality' | 'image_url' | 'sell_currency' | 'sell_price' | 'weight' | 'width' | 'height' | 'is_enabled'
>;

/** 设置项的描述, 后台据此生成表单, 对应后端 settings.Field */
export interface SettingField {
  key: string;
  label: string;
  help: string;
  type: 'string' | 'bool' | 'int';
  input: 'text' | 'textarea' | 'password' | 'switch' | 'number';
  /** 密钥, 读取时不返回明文 */
  secret: boolean;
  /** 会通过公开接口返回给前端 */
  public: boolean;
}

/** 一个设置组, 对应后端 settings.GroupView */
export interface SettingGroup {
  key: string;
  label: string;
  description: string;
  fields: SettingField[];
  /** 密钥字段固定为空字符串 */
  value: Record<string, string | number | boolean>;
  /** 密钥字段是否已设置 */
  secrets_set: Record<string, boolean>;
  /** 无法解密的密钥字段, 需要重新填写或清空才能保存 */
  secrets_broken: string[] | null;
  updated_by: number;
  /** 未保存过为空 */
  updated_at: string | null;
}

export interface UpdateSettingRequest {
  /** 只需提交改动的字段, 密钥字段留空表示不修改 */
  value: Record<string, string | number | boolean>;
  /** 需要清空的密钥字段 */
  clear_secrets?: string[];
}

export interface AdjustBalanceRequest {
  currency: string;
  /** 最小单位, 正数加款, 负数扣款, 扣款后余额不能为负 */
  amount: number;
  /** 调账原因, 用户可在流水中看到 */
  remark: string;
  /** 每次打开表单生成一个, 重复提交只生效一次 */
  idempotency_key: string;
}

export interface AdjustInventoryRequest {
  item_id: number;
  /** 正数发放, 负数扣除, 扣除后数量不能为负 */
  quantity: number;
  remark: string;
  idempotency_key: string;
}

/** 流水查询共用的筛选条件 */
interface TransactionFilters extends PageParams {
  user_id?: number;
  operator_id?: number;
  biz_type?: string;
  biz_id?: string;
  /** RFC3339, 包含 */
  start_time?: string;
  /** RFC3339, 不包含 */
  end_time?: string;
}

/** 流水的用户和操作人, 用户已不存在或系统操作时用户名为空 */
interface TransactionActors {
  user_id: number;
  username: string;
  operator_id: number;
  operator_username: string;
  idempotency_key: string;
}

/** 全部用户的余额流水, 对应后端 admin.WalletTransactionItem */
export type AdminWalletTransaction = WalletTransaction & TransactionActors;

export interface ListAdminWalletTransactionsParams extends TransactionFilters {
  currency?: string;
}

export interface ListAdminWalletTransactionsData {
  total: number;
  transactions: AdminWalletTransaction[];
}

/** 全部用户的背包流水, 对应后端 admin.InventoryTransactionItem */
export type AdminInventoryTransaction = InventoryTransaction & TransactionActors;

export interface ListAdminInventoryTransactionsParams extends TransactionFilters {
  item_id?: number;
}

export interface ListAdminInventoryTransactionsData {
  total: number;
  transactions: AdminInventoryTransaction[];
}
