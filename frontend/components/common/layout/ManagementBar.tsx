'use client';

import {useState} from 'react';
import {usePathname} from 'next/navigation';
import {useTheme} from 'next-themes';
import {LayoutDashboard, LogOut, Palette} from 'lucide-react';
import {toast} from 'sonner';
import {FloatingDock, type DockEntry} from '@/components/ui/floating-dock';
import {useAuth} from '@/components/common/auth';
import {ProfileDialog, UserAvatar} from '@/components/common/user';
import {useMailCounts, useMailCountsSync} from '@/hooks/use-mail-counts';
import {ROUTES} from '@/lib/access';
import {ADMIN_NAV, PLAYER_NAV, isActive, type NavBadge} from '@/lib/nav';
import {THEMES} from '@/lib/themes';

/** 底部悬浮导航: 桌面端居中, 移动端右下角 */
export function ManagementBar() {
  const pathname = usePathname();
  const {theme, setTheme} = useTheme();
  const {user, logout} = useAuth();
  const [profileOpen, setProfileOpen] = useState(false);
  useMailCountsSync(user?.id);
  const mailCounts = useMailCounts();
  const badges: Record<NavBadge, number> = {mail: mailCounts?.unread ?? 0};

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

  const items: DockEntry[] = PLAYER_NAV.map((entry) => ({
    title: entry.title,
    icon: <entry.icon />,
    href: entry.href,
    active: isActive(entry, pathname),
    badge: entry.badge && badges[entry.badge],
  }));

  // 管理区域只放一个入口, 各管理页面在管理区顶部切换
  if (user?.is_admin) {
    items.push(null, {
      title: '管理后台',
      icon: <LayoutDashboard />,
      href: ADMIN_NAV[0].entries[0].href,
      active: pathname === ROUTES.admin || pathname.startsWith(`${ROUTES.admin}/`),
    });
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
