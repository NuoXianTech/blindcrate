'use client';

import {useCallback, useEffect, useState} from 'react';
import {LoaderCircle, Package} from 'lucide-react';
import {toast} from 'sonner';
import {cn} from 'cn';
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
import {ItemImage} from '@/components/common/item';
import {useQualities} from '@/components/common/site';
import {useSubmit} from '@/hooks/use-submit';
import services from '@/lib/services';
import {AUCTION_PHASE, type AuctionView, type UsableItem} from '@/lib/services/auction';
import {toErrorMessage} from '@/lib/services/core';

/**
 * 道具栏: 背包中这张地图允许的道具, 点击确认后使用, 每次消耗 1 个
 * 只在出价阶段可用; 练习局、次数用完、加轮中不能出价时置灰
 */
export function ItemBar({view, onUse}: {
  view: AuctionView;
  /** 使用成功后返回的最新视角 */
  onUse: (view: AuctionView) => void;
}) {
  const {qualityOf} = useQualities();
  const {pending, submit} = useSubmit();
  const [items, setItems] = useState<UsableItem[] | null>(null);
  const [confirming, setConfirming] = useState<UsableItem | null>(null);
  const {self} = view;
  const matchId = view.match_id;

  const load = useCallback(async () => {
    try {
      setItems((await services.auction.listUsableItems(matchId)).items);
    } catch (err) {
      toast.error(toErrorMessage(err));
    }
  }, [matchId]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 进入对局时拉取背包中的道具
    void load();
  }, [load]);

  const contender = view.contenders.length === 0 || view.contenders.includes(self.seat);
  let blocked = '';
  if (self.is_practice) blocked = '练习局不能使用道具';
  else if (view.phase !== AUCTION_PHASE.bidding) blocked = '只能在出价阶段使用道具';
  else if (!contender) blocked = '加轮中只有并列最高的玩家可以使用道具';
  else if (self.uses_left === 0) blocked = '本局的道具次数已用完';
  const usedThisRound = new Set(self.uses.filter((u) => u.round === view.round).map((u) => u.item_id));

  const handleConfirm = submit(async () => {
    if (!confirming) return;
    onUse(await services.auction.use({match_id: matchId, round: view.round, item_id: confirming.item_id}));
    toast.success(`已使用 ${confirming.name}`);
    setConfirming(null);
    await load();
  });

  return (
    <section className="flex flex-col gap-2 rounded-xl border bg-card p-4">
      <div className="flex items-center justify-between text-sm">
        <span className="font-medium">道具</span>
        <span className="text-muted-foreground">
          本局还能用 <span className="font-mono tabular-nums text-foreground">{self.uses_left}</span> 个
        </span>
      </div>

      {items === null ? (
        <p className="text-xs text-muted-foreground">正在读取背包</p>
      ) : items.length === 0 ? (
        <p className="flex items-center gap-1.5 text-xs text-muted-foreground">
          <Package className="size-3.5" />
          背包里没有这张地图能用的道具, 可以在商城购买
        </p>
      ) : (
        <ul className="flex flex-col gap-1.5">
          {items.map((it) => {
            const q = qualityOf(it.quality);
            const used = usedThisRound.has(it.item_id);
            const reason = blocked || (used ? '本轮已经用过这个道具' : '');
            return (
              <li key={it.item_id}>
                <button
                  type="button"
                  className={cn(
                      'flex w-full items-center gap-2.5 rounded-lg border p-2 text-left transition-colors',
                    reason ? 'cursor-not-allowed opacity-50' : 'hover:bg-accent',
                  )}
                  title={reason || it.description || undefined}
                  disabled={!!reason || pending}
                  onClick={() => setConfirming(it)}
                >
                  <span className="relative flex size-9 shrink-0 items-center justify-center overflow-hidden rounded-md border bg-background">
                    <ItemImage item={it} />
                    <span className={cn('absolute inset-x-0 top-0 h-0.5', q.bar)} />
                  </span>
                  <span className="flex min-w-0 flex-1 flex-col">
                    <span className={cn('truncate text-sm font-medium', q.text)}>{it.name}</span>
                    <span className="truncate text-xs text-muted-foreground">{used ? '本轮已用' : it.description || '使用后获得情报'}</span>
                  </span>
                  <span className="font-mono text-sm tabular-nums text-muted-foreground">×{it.quantity}</span>
                </button>
              </li>
            );
          })}
        </ul>
      )}
      {blocked && items && items.length > 0 && <span className="text-xs text-muted-foreground">{blocked}</span>}

      <AlertDialog open={!!confirming} onOpenChange={(open) => !open && !pending && setConfirming(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>使用 {confirming?.name}</AlertDialogTitle>
            <AlertDialogDescription className="flex flex-col gap-1">
              {confirming?.description && <span>{confirming.description}</span>}
              <span>消耗 1 个, 得到的情报只有你看得到; 使用后本局还能用 {Math.max(self.uses_left - 1, 0)} 个</span>
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={pending}>取消</AlertDialogCancel>
            <AlertDialogAction disabled={pending} onClick={handleConfirm}>
              {pending && <LoaderCircle className="animate-spin" />}
              使用
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </section>
  );
}
