import {
  ArrowDownUp,
  Backpack,
  CalendarRange,
  ChartColumn,
  ClipboardList,
  Coins,
  Dices,
  Gem,
  Gamepad2,
  History,
  Mail,
  Mails,
  Medal,
  Mountain,
  Package,
  ReceiptText,
  ScrollText,
  Settings,
  ShieldAlert,
  ShoppingBag,
  Store,
  Swords,
  Tags,
  Trophy,
  Users,
  Wallet,
  Warehouse,
  type LucideIcon,
} from 'lucide-react';
import {ROUTES} from '@/lib/access';

/**
 * 导航入口: 玩家页面在底部 dock 中, 管理页面在管理区顶部按分组展示
 * 新增页面时在这里加一项, 访问规则见 lib/access.ts
 */
export interface NavEntry {
  title: string;
  href: string;
  icon: LucideIcon;
  /** 只在路径完全相同时高亮, 用于有子页面的入口 */
  exact?: boolean;
  /** 角标显示的数字, 取值见 ManagementBar */
  badge?: NavBadge;
}

/** 入口角标的种类 */
export type NavBadge = 'mail';

export interface NavGroup {
  label: string;
  entries: NavEntry[];
}

/** 玩家页面, 按 dock 中的顺序 */
export const PLAYER_NAV: NavEntry[] = [
  {title: '大厅', icon: Gamepad2, href: ROUTES.gameHome, exact: true},
  {title: '任务', icon: ClipboardList, href: ROUTES.missions},
  {title: '战绩', icon: History, href: ROUTES.matches},
  {title: '背包', icon: Backpack, href: ROUTES.inventory},
  {title: '商城', icon: Store, href: ROUTES.shop},
  {title: '钱包', icon: Wallet, href: ROUTES.wallet},
  {title: '邮件', icon: Mail, href: ROUTES.mail, badge: 'mail'},
  {title: '排行榜', icon: Trophy, href: ROUTES.leaderboard},
  {title: '数据', icon: ChartColumn, href: ROUTES.stats},
];

/** 管理页面, 按分组; dock 中只有一个入口, 指向第一项 */
export const ADMIN_NAV: NavGroup[] = [
  {
    label: '内容',
    entries: [
      {title: '物品管理', icon: Package, href: ROUTES.adminItems},
      {title: '品质管理', icon: Gem, href: ROUTES.adminQualities},
      {title: '货币管理', icon: Coins, href: ROUTES.adminCurrencies},
      {title: '掉落池', icon: Dices, href: ROUTES.adminLootPools},
      {title: '地图管理', icon: Warehouse, href: ROUTES.adminVenues},
    ],
  },
  {
    label: '排位',
    entries: [
      {title: '榜单管理', icon: Medal, href: ROUTES.adminLeaderboards},
      {title: '赛季管理', icon: CalendarRange, href: ROUTES.adminSeasons},
      {title: '段位管理', icon: Mountain, href: ROUTES.adminTiers},
      {title: '对局管理', icon: Swords, href: ROUTES.adminMatches},
      {title: '可疑名单', icon: ShieldAlert, href: ROUTES.adminSuspects},
    ],
  },
  {
    label: '商城',
    entries: [
      {title: '商品管理', icon: Tags, href: ROUTES.adminShopItems},
      {title: '商城订单', icon: ShoppingBag, href: ROUTES.adminShopOrders},
    ],
  },
  {
    label: '运营',
    entries: [
      {title: '游戏邮件', icon: Mails, href: ROUTES.adminMails},
      {title: '任务管理', icon: ClipboardList, href: ROUTES.adminMissions},
    ],
  },
  {
    label: '系统',
    entries: [
      {title: '用户管理', icon: Users, href: ROUTES.adminUsers},
      {title: '流水查询', icon: ReceiptText, href: ROUTES.adminTransactions},
      {title: '操作日志', icon: ScrollText, href: ROUTES.adminAuditLogs},
      {title: '导入导出', icon: ArrowDownUp, href: ROUTES.adminPacks},
      {title: '系统设置', icon: Settings, href: ROUTES.adminSettings},
    ],
  },
];

/** 当前路径是否属于这个入口 */
export function isActive(entry: NavEntry, pathname: string) {
  if (entry.exact) return pathname === entry.href;
  return pathname === entry.href || pathname.startsWith(`${entry.href}/`);
}
