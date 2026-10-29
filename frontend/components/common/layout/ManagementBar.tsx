'use client';

import {useState} from 'react';
import {usePathname} from 'next/navigation';
import {useTheme} from 'next-themes';
import {
  ArrowDownUp,
  Backpack,
  CalendarRange,
  ChartColumn,
  Coins,
  Dices,
  Gem,
  Gamepad2,
  History,
  LogOut,
  Medal,
  Mountain,
  Package,
  Palette,
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
} from 'lucide-react';
import {toast} from 'sonner';
import {FloatingDock, type DockEntry} from '@/components/ui/floating-dock';
import {useAuth} from '@/components/common/auth';
import {ProfileDialog, UserAvatar} from '@/components/common/user';
import {ROUTES} from '@/lib/access';
import {THEMES} from '@/lib/themes';

/** 底部悬浮导航: 桌面端居中, 移动端右下角 */
export function ManagementBar() {
  const pathname = usePathname();
  const {theme, setTheme} = useTheme();
  const {user, logout} = useAuth();
  const [profileOpen, setProfileOpen] = useState(false);

  const cycleTheme = () => {
    const idx = THEMES.findIndex((t) => t.value === theme);
    setTheme(THEMES[(idx + 1) % THEMES.length].value);
  };

  // 退出成功后由 AuthProvider 跳回首页
  const handleLogout = async () => {
    try {
      await logout();
    } catch {
      toast.error('退出登录失败, 请稍后再试');
    }
  };

  const items: DockEntry[] = [
    {title: '大厅', icon: <Gamepad2 />, href: ROUTES.gameHome, active: pathname === ROUTES.gameHome},
    {title: '战绩', icon: <History />, href: ROUTES.matches, active: pathname.startsWith(ROUTES.matches)},
    {title: '背包', icon: <Backpack />, href: ROUTES.inventory, active: pathname.startsWith(ROUTES.inventory)},
    {title: '商城', icon: <Store />, href: ROUTES.shop, active: pathname.startsWith(ROUTES.shop)},
    {title: '钱包', icon: <Wallet />, href: ROUTES.wallet, active: pathname.startsWith(ROUTES.wallet)},
    {title: '排行榜', icon: <Trophy />, href: ROUTES.leaderboard, active: pathname.startsWith(ROUTES.leaderboard)},
    {title: '数据', icon: <ChartColumn />, href: ROUTES.stats, active: pathname.startsWith(ROUTES.stats)},
  ];

  // 管理区域, 单独一组
  if (user?.is_admin) {
    items.push(
        null,
        {title: '用户管理', icon: <Users />, href: ROUTES.adminUsers, active: pathname.startsWith(ROUTES.adminUsers)},
        {title: '物品管理', icon: <Package />, href: ROUTES.adminItems, active: pathname.startsWith(ROUTES.adminItems)},
        {title: '品质管理', icon: <Gem />, href: ROUTES.adminQualities, active: pathname.startsWith(ROUTES.adminQualities)},
        {title: '货币管理', icon: <Coins />, href: ROUTES.adminCurrencies, active: pathname.startsWith(ROUTES.adminCurrencies)},
        {title: '掉落池', icon: <Dices />, href: ROUTES.adminLootPools, active: pathname.startsWith(ROUTES.adminLootPools)},
        {title: '地图管理', icon: <Warehouse />, href: ROUTES.adminVenues, active: pathname.startsWith(ROUTES.adminVenues)},
        {title: '榜单管理', icon: <Medal />, href: ROUTES.adminLeaderboards, active: pathname.startsWith(ROUTES.adminLeaderboards)},
        {title: '赛季管理', icon: <CalendarRange />, href: ROUTES.adminSeasons, active: pathname.startsWith(ROUTES.adminSeasons)},
        {title: '段位管理', icon: <Mountain />, href: ROUTES.adminTiers, active: pathname.startsWith(ROUTES.adminTiers)},
        {title: '对局管理', icon: <Swords />, href: ROUTES.adminMatches, active: pathname.startsWith(ROUTES.adminMatches)},
        {title: '可疑名单', icon: <ShieldAlert />, href: ROUTES.adminSuspects, active: pathname.startsWith(ROUTES.adminSuspects)},
        {title: '商品管理', icon: <Tags />, href: ROUTES.adminShopItems, active: pathname.startsWith(ROUTES.adminShopItems)},
        {title: '商城订单', icon: <ShoppingBag />, href: ROUTES.adminShopOrders, active: pathname.startsWith(ROUTES.adminShopOrders)},
        {title: '流水查询', icon: <ReceiptText />, href: ROUTES.adminTransactions, active: pathname.startsWith(ROUTES.adminTransactions)},
        {title: '操作日志', icon: <ScrollText />, href: ROUTES.adminAuditLogs, active: pathname.startsWith(ROUTES.adminAuditLogs)},
        {title: '导入导出', icon: <ArrowDownUp />, href: ROUTES.adminPacks, active: pathname.startsWith(ROUTES.adminPacks)},
        {title: '系统设置', icon: <Settings />, href: ROUTES.adminSettings, active: pathname.startsWith(ROUTES.adminSettings)},
    );
  }

  // 个人区域: 只有一个主题时不显示切换
  const personal: DockEntry[] = [];
  if (THEMES.length > 1) personal.push({title: '切换主题', icon: <Palette />, onClick: cycleTheme});
  if (user) {
    personal.push(
        {
          title: user.nickname,
          icon: <UserAvatar user={user} className="text-sm" />,
          onClick: () => setProfileOpen(true),
        },
        {title: '退出登录', icon: <LogOut />, onClick: handleLogout},
    );
  }
  if (personal.length) items.push(null, ...personal);

  return (
    <div className="fixed right-4 bottom-4 z-50 md:right-auto md:left-1/2 md:-translate-x-1/2">
      <FloatingDock items={items} />
      <ProfileDialog open={profileOpen} onOpenChange={setProfileOpen} />
    </div>
  );
}
