'use client';

import {useCallback, useEffect, useState} from 'react';
import Link from 'next/link';
import {usePathname, useRouter} from 'next/navigation';
import {Swords} from 'lucide-react';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import {Button} from '@/components/ui/button';
import {useAuth} from '@/components/common/auth';
import {useWsConnected, useWsEvent} from '@/hooks/use-ws';
import {ROUTES, matchHref} from '@/lib/access';
import {modeLabel} from '@/lib/auction';
import services from '@/lib/services';
import {MATCH_MESSAGES, MATCH_STATUS, type MatchView} from '@/lib/services/match';

/** 选择过暂不回去的对局 ID, 存在 sessionStorage, 同一个标签页内不再弹出对话框 */
const DISMISSED_KEY = 'match:rejoin-dismissed';

/** 这局是否已经选择过暂不回去 */
export function isDismissed(id: number) {
  try {
    return sessionStorage.getItem(DISMISSED_KEY) === String(id);
  } catch {
    return false;
  }
}

/**
 * 游戏内页面的对局提示: 有进行中且还没释放的对局时(如断线、关掉页面后回来)弹出对话框, 选择是否回到对局;
 * 选择暂不回去后改为顶部的提示条, 随时可以回去
 * 切换页面和重连后重新拉取, match:updated 推送到达时立即更新; 对局结束或释放后消失
 */
export function CurrentMatchBanner() {
  const {user} = useAuth();
  const pathname = usePathname();
  const router = useRouter();
  const connected = useWsConnected();
  const [match, setMatch] = useState<MatchView | null>(null);
  // 进入页面或重连时就已经在进行的对局, 只有它弹出对话框; 在大厅排到的新对局会直接进入, 不用问
  const [found, setFound] = useState<number | null>(null);
  // 选择过暂不回去的对局
  const [dismissed, setDismissed] = useState<number | null>(null);

  const load = useCallback(async () => {
    if (!user) {
      setMatch(null);
      return;
    }
    try {
      const current = await services.match.getCurrentMatch();
      if (current && isDismissed(current.id)) setDismissed(current.id);
      setMatch(current);
      setFound(current?.id ?? null);
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
  if (!active) return null;

  const dismiss = () => {
    setDismissed(match.id);
    try {
      sessionStorage.setItem(DISMISSED_KEY, String(match.id));
    } catch {
      // 不能写 sessionStorage 时下次进入再问一次
    }
  };

  const dialog = (
    <AlertDialog open={found === match.id && dismissed !== match.id} onOpenChange={(open) => !open && dismiss()}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>回到对局?</AlertDialogTitle>
          <AlertDialogDescription>
            你有一局{modeLabel(match.mode)}正在进行。对局不会等你, 不回去时每轮按不出价处理; 排位中整局都不出价会记为挂机。
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>暂不回去</AlertDialogCancel>
          <AlertDialogAction onClick={() => router.push(matchHref(match.id))}>回到对局</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );

  // 大厅的匹配状态栏已有回到对局的入口
  if (pathname === ROUTES.gameHome) return dialog;

  return (
    <div className="flex items-center justify-between gap-3 rounded-xl border border-primary/40 bg-primary/10 px-4 py-3">
      <span className="flex items-center gap-2 text-sm">
        <Swords className="size-4 text-primary" />
        你有一局{modeLabel(match.mode)}正在进行
      </span>
      <Button size="sm" asChild>
        <Link href={matchHref(match.id)}>回到对局</Link>
      </Button>
      {dialog}
    </div>
  );
}
