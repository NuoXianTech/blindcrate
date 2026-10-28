'use client';

import {useState} from 'react';
import {usePathname} from 'next/navigation';
import {useTheme} from 'next-themes';
import {
  Backpack,
  Dices,
  Gamepad2,
  LogOut,
  Package,
  Palette,
  ReceiptText,
  ScrollText,
  Settings,
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
    {title: '背包', icon: <Backpack />, href: ROUTES.inventory, active: pathname.startsWith(ROUTES.inventory)},
    {title: '钱包', icon: <Wallet />, href: ROUTES.wallet, active: pathname.startsWith(ROUTES.wallet)},
  ];

  // 管理区域, 单独一组
  if (user?.is_admin) {
    items.push(
        null,
        {title: '用户管理', icon: <Users />, href: ROUTES.adminUsers, active: pathname.startsWith(ROUTES.adminUsers)},
        {title: '物品管理', icon: <Package />, href: ROUTES.adminItems, active: pathname.startsWith(ROUTES.adminItems)},
        {title: '掉落池', icon: <Dices />, href: ROUTES.adminLootPools, active: pathname.startsWith(ROUTES.adminLootPools)},
        {title: '地图管理', icon: <Warehouse />, href: ROUTES.adminVenues, active: pathname.startsWith(ROUTES.adminVenues)},
        {title: '流水查询', icon: <ReceiptText />, href: ROUTES.adminTransactions, active: pathname.startsWith(ROUTES.adminTransactions)},
        {title: '操作日志', icon: <ScrollText />, href: ROUTES.adminAuditLogs, active: pathname.startsWith(ROUTES.adminAuditLogs)},
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
