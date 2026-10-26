"use client";

import { usePathname } from "next/navigation";
import { useTheme } from "next-themes";
import { House, Palette } from "lucide-react";
import { FloatingDock, type DockEntry } from "@/components/ui/floating-dock";
import { THEMES } from "@/lib/themes";

/** 底部悬浮导航: 桌面端居中, 移动端右下角 */
export function ManagementBar() {
  const pathname = usePathname();
  const { theme, setTheme } = useTheme();

  const cycleTheme = () => {
    const idx = THEMES.findIndex((t) => t.value === theme);
    setTheme(THEMES[(idx + 1) % THEMES.length].value);
  };

  const items: DockEntry[] = [
    { title: "首页", icon: <House />, href: "/", active: pathname === "/" },
    null,
    { title: "切换主题", icon: <Palette />, onClick: cycleTheme },
  ];

  return (
    <div className="fixed right-4 bottom-4 z-50 md:right-auto md:left-1/2 md:-translate-x-1/2">
      <FloatingDock items={items} />
    </div>
  );
}
