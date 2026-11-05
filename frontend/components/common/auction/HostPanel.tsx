'use client';

import {useState} from 'react';
import {Eye, Flame, LoaderCircle, Megaphone} from 'lucide-react';
import {toast} from 'sonner';
import {cn} from 'cn';
import {Badge} from '@/components/ui/badge';
import {Button} from '@/components/ui/button';
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
import {useQualities} from '@/components/common/site';
import {CurrencyIcon} from '@/components/common/wallet';
import {useSubmit} from '@/hooks/use-submit';
import {HOST_PERSONAS, hypeLevel, intelTier} from '@/lib/auction';
import services from '@/lib/services';
import {AUCTION_PHASE, HOST_HINT, type AuctionView, type HostHint} from '@/lib/services/auction';
import {formatAmount} from '@/lib/utils';

/** 提示中的品质, 带品质颜色 */
function QualityName({quality}: { quality: number }) {
  const q = useQualities().qualityOf(quality);
  return <span className={cn('font-medium', q.text)}>{q.name}</span>;
}

/** 一条提示的内容 */
function HintText({hint}: { hint: HostHint }) {
  switch (hint.kind) {
    case HOST_HINT.count:
      return (
        <span>
          <QualityName quality={hint.quality} /> 及以上的藏品共有 <span className="font-mono tabular-nums">{hint.count}</span> 件
        </span>
      );
    case HOST_HINT.best:
      return <span>仓库里最好的藏品是 <QualityName quality={hint.quality} /></span>;
    case HOST_HINT.piece:
      return (
        <span>
          <Megaphone className="inline size-3.5 align-[-2px]" /> 标出的这件是 <QualityName quality={hint.quality} />
        </span>
      );
    case HOST_HINT.value:
      return <span>这一仓{hint.above ? '比平均更值钱' : '不如平均值钱'}</span>;
  }
  return null;
}

/**
 * 主持人: 公开热度和每轮的提示(可能说错), 开箱后公开性格和提示的真假; 出价阶段可以向他买只有自己看得到的情报
 * 没有主持人提示的地图只显示情报商店; 两者都没有时不显示
 */
export function HostPanel({view, decimals, onBuy}: {
  view: AuctionView;
  decimals: number;
  /** 买到情报后返回的最新视角 */
  onBuy: (view: AuctionView) => void;
}) {
  const {pending, submit} = useSubmit();
  const [confirming, setConfirming] = useState(false);
  const {host, venue, self} = view;
  const selling = venue.intel_price > 0 && venue.intel_limit > 0;
  if (!host && !selling) return null;

  const persona = host ? HOST_PERSONAS[host.persona] : undefined;
  const level = host ? hypeLevel(host.hype) : undefined;
  const lastHype = host?.hypes.at(-1);
  const others = view.intel_seats.filter((s) => s.seat !== self.seat && s.round === view.round);

  const contender = view.contenders.length === 0 || view.contenders.includes(self.seat);
  let blocked = '';
  if (self.is_practice) blocked = '练习局不能买情报';
  else if (view.phase !== AUCTION_PHASE.bidding) blocked = '只能在出价阶段买情报';
  else if (!contender) blocked = '加轮中只有并列最高的玩家可以买情报';
  else if (self.intel_bought) blocked = '本轮已经买过情报';
  else if (self.intel_left === 0) blocked = '本局的情报次数已用完';

  const handleConfirm = submit(async () => {
    onBuy(await services.auction.buyIntel({match_id: view.match_id, round: view.round}));
    toast.success('已买到情报');
    setConfirming(false);
  });

  return (
    <section className="flex flex-col gap-3 rounded-xl border bg-card p-4">
      {host && (
        <>
          <div className="flex items-center justify-between gap-2">
            <span className="flex items-center gap-1.5 text-sm font-medium">
              <Megaphone className="size-4" />
              主持人
              {persona && <Badge variant="outline" className="px-1.5" title={persona.description}>{persona.label}</Badge>}
            </span>
            <span className="flex items-center gap-1 text-xs" title="每轮按加价的人数和幅度变化">
              <Flame className={cn('size-3.5', level?.className)} />
              <span className={level?.className}>{level?.label}</span>
              <span className="font-mono tabular-nums">{host.hype}</span>
              {lastHype && lastHype.round > 0 && (
                <span className="font-mono text-muted-foreground tabular-nums">
                  ({formatAmount(lastHype.delta, 0, {signed: true})})
                </span>
              )}
            </span>
          </div>
          <div className="h-1.5 overflow-hidden rounded-full bg-muted">
            <div className="h-full rounded-full bg-gradient-to-r from-quality-blue via-quality-gold to-quality-red" style={{width: `${host.hype}%`}} />
          </div>
          {persona && <span className="text-xs text-muted-foreground">这局的主持人{persona.label}: {persona.description}</span>}
          {lastHype && lastHype.round > 0 && (
            <span className="text-xs text-muted-foreground">
              第 {lastHype.round} 轮 {lastHype.raises > 0 ? `${lastHype.raises} 人加价` : '没人加价'}
            </span>
          )}
          {host.hints.length > 0 ? (
            <ul className="flex flex-col gap-1.5">
              {[...host.hints].reverse().map((h) => (
                <li key={h.round} className="flex items-start justify-between gap-2 rounded-lg border px-3 py-2 text-sm">
                  <HintText hint={h} />
                  <span className="flex shrink-0 items-center gap-1.5 text-xs text-muted-foreground">
                    {h.truthful !== null && (
                      <span className={h.truthful ? 'text-quality-green' : 'text-quality-red'}>{h.truthful ? '属实' : '说错了'}</span>
                    )}
                    第 {h.round} 轮
                  </span>
                </li>
              ))}
            </ul>
          ) : (
            <span className="text-xs text-muted-foreground">主持人还没有开口</span>
          )}
        </>
      )}

      {selling && !view.result && (
        <div className={cn('flex flex-col gap-2', host && 'border-t pt-3')}>
          <div className="flex items-center justify-between gap-2 text-sm">
            <span className="font-medium">情报</span>
            <span className="text-xs text-muted-foreground">
              本局还能买 <span className="font-mono tabular-nums text-foreground">{self.intel_left}</span> 次
            </span>
          </div>
          <Button variant="outline" disabled={!!blocked || pending} title={blocked || undefined} onClick={() => setConfirming(true)}>
            <Eye />
            向主持人买情报
            <span className="flex items-center gap-1 font-mono tabular-nums">
              <CurrencyIcon currency={venue.bid_currency} className="size-3.5" />
              {formatAmount(venue.intel_price, decimals)}
            </span>
          </Button>
          <span className="text-xs text-muted-foreground">
            {blocked || `本轮能买到${intelTier(view.round, venue.rounds)}, 只有你看得到`}
          </span>
          {others.length > 0 && (
            <span className="text-xs text-quality-gold">
              {others.map((s) => `${s.seat} 号位`).join('、')} 本轮买了情报
            </span>
          )}
        </div>
      )}

      <AlertDialog open={confirming} onOpenChange={(open) => !open && !pending && setConfirming(false)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>向主持人买情报</AlertDialogTitle>
            <AlertDialogDescription className="flex flex-col gap-1">
              <span>本轮能买到{intelTier(view.round, venue.rounds)}, 只有你看得到; 其他人会看到你买了情报</span>
              <span>
                从可用余额扣除 {formatAmount(venue.intel_price, decimals)}, 带入资金不能用来买情报; 流拍不退, 对局取消时退还
              </span>
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={pending}>取消</AlertDialogCancel>
            <AlertDialogAction disabled={pending} onClick={handleConfirm}>
              {pending && <LoaderCircle className="animate-spin" />}
              购买
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </section>
  );
}
