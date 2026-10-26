"use client";

import { useState } from "react";
import { usePathname } from "next/navigation";
import { useTheme } from "next-themes";
import { Gamepad2, LogOut, Palette } from "lucide-react";
import { toast } from "sonner";
import { FloatingDock, type DockEntry } from "@/components/ui/floating-dock";
import { useAuth } from "@/components/common/auth/AuthProvider";
import { ProfileDialog } from "@/components/common/user/ProfileDialog";
import { UserAvatar } from "@/components/common/user/UserAvatar";
import { THEMES } from "@/lib/themes";

/** 底部悬浮导航: 桌面端居中, 移动端右下角 */
export function ManagementBar() {
  const pathname = usePathname();
  const { theme, setTheme } = useTheme();
  const { user, logout } = useAuth();
  const [profileOpen, setProfileOpen] = useState(false);

  const cycleTheme = () => {
    const idx = THEMES.findIndex((t) => t.value === theme);
    setTheme(THEMES[(idx + 1) % THEMES.length].value);
  };

  // 退出成功后由登录守卫跳回首页
  const handleLogout = async () => {
    try {
      await logout();
    } catch {
      toast.error("退出登录失败, 请稍后再试");
    }
  };

  const items: DockEntry[] = [
    { title: "大厅", icon: <Gamepad2 />, href: "/home", active: pathname === "/home" },
    null,
    { title: "切换主题", icon: <Palette />, onClick: cycleTheme },
  ];

  if (user) {
    items.push(
      {
        title: user.nickname,
        icon: <UserAvatar user={user} className="text-sm" />,
        onClick: () => setProfileOpen(true),
      },
      { title: "退出登录", icon: <LogOut />, onClick: handleLogout },
    );
  }

  return (
    <div className="fixed right-4 bottom-4 z-50 md:right-auto md:left-1/2 md:-translate-x-1/2">
      <FloatingDock items={items} />
      <ProfileDialog open={profileOpen} onOpenChange={setProfileOpen} />
    </div>
  );
}
