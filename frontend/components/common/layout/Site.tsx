"use client";

import { createContext, useContext, useState, useSyncExternalStore } from "react";
import { Megaphone, Package, X } from "lucide-react";
import { cn } from "cn";
import type { PublicSettings } from "@/lib/site";

const SiteContext = createContext<PublicSettings | null>(null);

/** 把服务端读取的公开设置交给客户端组件, 见 app/layout.tsx */
export function SiteProvider({ settings, children }: { settings: PublicSettings; children: React.ReactNode }) {
  return <SiteContext.Provider value={settings}>{children}</SiteContext.Provider>;
}

export function useSite(): PublicSettings {
  const settings = useContext(SiteContext);
  if (!settings) throw new Error("useSite must be used within SiteProvider");
  return settings;
}

/** 站点 Logo, 没有设置时显示默认图标 */
export function SiteLogo({ className }: { className?: string }) {
  const { site } = useSite();
  if (site.logo_url) {
    // 图片地址由管理员填写, 域名不固定, 不走 next/image
    // eslint-disable-next-line @next/next/no-img-element
    return <img src={site.logo_url} alt={site.name} className={cn("size-full object-contain", className)} />;
  }
  return <Package className={cn("size-1/2", className)} aria-hidden />;
}

/** 关闭时记下公告内容, 公告改了会重新显示 */
const DISMISSED_KEY = "blindcrate:announcement-dismissed";

function subscribeStorage(callback: () => void) {
  window.addEventListener("storage", callback);
  return () => window.removeEventListener("storage", callback);
}

/** 站点公告, 可关闭 */
export function Announcement({ className }: { className?: string }) {
  const text = useSite().site.announcement.trim();
  // 服务端不知道是否关闭过, 按已关闭处理, 避免关掉的公告闪一下
  const dismissed = useSyncExternalStore(
    subscribeStorage,
    () => localStorage.getItem(DISMISSED_KEY),
    () => text,
  );
  const [closed, setClosed] = useState(false);

  if (!text || closed || dismissed === text) return null;

  const close = () => {
    localStorage.setItem(DISMISSED_KEY, text);
    setClosed(true);
  };

  return (
    <div
      role="status"
      className={cn(
        "flex animate-in items-start gap-3 rounded-xl border border-primary/30 bg-primary/10 px-4 py-3 text-sm fade-in",
        className,
      )}
    >
      <Megaphone className="mt-0.5 size-4 shrink-0 text-primary" aria-hidden />
      <p className="min-w-0 flex-1 text-left break-words whitespace-pre-wrap">{text}</p>
      <button
        type="button"
        aria-label="关闭公告"
        className="-m-1 rounded-md p-1 text-muted-foreground hover:bg-background/50 hover:text-foreground"
        onClick={close}
      >
        <X className="size-4" />
      </button>
    </div>
  );
}

/** 备案号和联系邮箱, 都没有设置时不显示 */
export function SiteFooter({ className }: { className?: string }) {
  const { site } = useSite();
  if (!site.icp && !site.contact_email) return null;
  return (
    <footer className={cn("flex flex-wrap items-center justify-center gap-x-4 gap-y-1 text-xs text-muted-foreground", className)}>
      {site.icp && (
        <a href="https://beian.miit.gov.cn/" target="_blank" rel="noreferrer" className="hover:text-foreground">
          {site.icp}
        </a>
      )}
      {site.contact_email && (
        <a href={`mailto:${site.contact_email}`} className="hover:text-foreground">
          联系邮箱 {site.contact_email}
        </a>
      )}
    </footer>
  );
}
