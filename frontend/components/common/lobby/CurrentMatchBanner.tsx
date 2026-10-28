'use client';

import {useCallback, useEffect, useState} from 'react';
import Link from 'next/link';
import {usePathname} from 'next/navigation';
import {Swords} from 'lucide-react';
import {Button} from '@/components/ui/button';
import {useAuth} from '@/components/common/auth';
import {useWsConnected, useWsEvent} from '@/hooks/use-ws';
import {ROUTES, matchHref} from '@/lib/access';
import {modeLabel} from '@/lib/auction';
import services from '@/lib/services';
import {MATCH_MESSAGES, MATCH_STATUS, type MatchView} from '@/lib/services/match';

/**
 * 游戏内页面顶部的对局提示: 有进行中且还没释放的对局时提示回到对局
 * 切换页面和重连后重新拉取, match:updated 推送到达时立即更新; 对局结束或释放后消失
 */
export function CurrentMatchBanner() {
  const {user} = useAuth();
  const pathname = usePathname();
  const connected = useWsConnected();
  const [match, setMatch] = useState<MatchView | null>(null);

  const load = useCallback(async () => {
    if (!user) {
      setMatch(null);
      return;
    }
    try {
      setMatch(await services.match.getCurrentMatch());
    } catch {
      // 只是提示, 失败时保留上次的结果
    }
  }, [user]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 切换页面和重连后刷新
    void load();
  }, [load, pathname, connected]);

  useWsEvent<MatchView>(MATCH_MESSAGES.updated, (view) => {
    // 推送可能乱序, 同一局只采用 version 更大的
    setMatch((prev) => (prev?.id === view.id && prev.version > view.version ? prev : view));
  });

  const active = !!match && match.status === MATCH_STATUS.playing && !match.released_at;
  // 大厅的匹配状态栏已有回到对局的入口
  if (!active || pathname === ROUTES.gameHome) return null;

  return (
    <div className="flex items-center justify-between gap-3 rounded-xl border border-primary/40 bg-primary/10 px-4 py-3">
      <span className="flex items-center gap-2 text-sm">
        <Swords className="size-4 text-primary" />
        你有一局{modeLabel(match.mode)}正在进行
      </span>
      <Button size="sm" asChild>
        <Link href={matchHref(match.id)}>回到对局</Link>
      </Button>
    </div>
  );
}
