'use client';

import {useState, useSyncExternalStore} from 'react';
import {Megaphone, X} from 'lucide-react';
import {cn} from 'cn';
import {useSite} from './SiteProvider';

/** 关闭时记下公告内容, 公告改了会重新显示 */
const DISMISSED_KEY = 'blindcrate:announcement-dismissed';

/**
 * 订阅其他标签页对 localStorage 的修改
 */
function subscribeStorage(callback: () => void) {
  window.addEventListener('storage', callback);
  return () => window.removeEventListener('storage', callback);
}

/**
 * 站点公告, 可关闭
 */
export function Announcement({className}: { className?: string }) {
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
          'flex animate-in items-start gap-3 rounded-xl border border-primary/30 bg-primary/10 px-4 py-3 text-sm fade-in',
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
