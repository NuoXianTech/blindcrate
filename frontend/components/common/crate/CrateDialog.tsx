'use client';

import {useEffect, useMemo, useRef, useState} from 'react';
import {FastForward, PackageOpen} from 'lucide-react';
import {animate, motion, useMotionValue, useReducedMotion} from 'motion/react';
import {nanoid} from 'nanoid';
import {toast} from 'sonner';
import {cn} from 'cn';
import {Button} from '@/components/ui/button';
import {Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle} from '@/components/ui/dialog';
import {Skeleton} from '@/components/ui/skeleton';
import {SubmitButton} from '@/components/common/form';
import {ItemImage} from '@/components/common/item';
import {useQualities} from '@/components/common/site';
import {CurrencyIcon} from '@/components/common/wallet';
import services from '@/lib/services';
import {toErrorMessage} from '@/lib/services/core';
import type {CrateOpenResult, CrateView} from '@/lib/services/crate';
import type {RewardLine} from '@/lib/services/reward';
import {CrateReel} from './CrateReel';
import {
  amountText,
  buildReel,
  formatProbability,
  idleReel,
  randomJitter,
  reelOffset,
  SPIN_EASE,
  SPIN_SECONDS,
  START_INDEX,
  WIN_INDEX,
} from './reel';

/**
 * 开箱对话框的Props接口
 */
interface CrateDialogProps {
  /** 要开启的箱子, null 时关闭 */
  crate: { item_id: number; name: string } | null;
  onClose: () => void;
  /** 每开启一次后调用, 用于刷新背包 */
  onOpened: () => void;
}

/**
 * 开箱对话框: 公示箱子中每一项的概率, 开启后播放滚动动画揭晓抽中的一项
 */
export function CrateDialog({crate, onClose, onOpened}: CrateDialogProps) {
  // 从点击开启到揭晓之前不能关闭, 否则已经发放的结果没有展示
  const [locked, setLocked] = useState(false);

  return (
    <Dialog open={!!crate} onOpenChange={(open) => !open && !locked && onClose()}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-2xl" showCloseButton={!locked}>
        {crate && (
          <>
            <DialogHeader>
              <DialogTitle>开启 {crate.name}</DialogTitle>
              <DialogDescription>每次消耗 1 个, 按下方公示的概率开出其中一项, 直接放进背包或钱包</DialogDescription>
            </DialogHeader>
            {/* 每次打开都重新挂载, 重新读取箱子的内容 */}
            <CrateStage key={crate.item_id} itemID={crate.item_id} onClose={onClose} onOpened={onOpened} onLockedChange={setLocked} />
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}

/** 一次开启: 滚动条上的每一格和开启的结果 */
interface Spin {
  tiles: RewardLine[];
  result: CrateOpenResult;
}

/** 正在播放的滚动 */
interface Playback {
  stop(): void;
}

/**
 * 开箱的主体: 滚动条、结果、箱内物品和按钮
 */
function CrateStage({itemID, onClose, onOpened, onLockedChange}: {
  itemID: number;
  onClose: () => void;
  onOpened: () => void;
  onLockedChange: (locked: boolean) => void;
}) {
  const reduced = useReducedMotion();
  const [view, setView] = useState<CrateView | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [quantity, setQuantity] = useState(0);
  // 幂等键在开启成功后才换新的, 请求失败后重试不会多开一个
  const [idempotencyKey, setIdempotencyKey] = useState(() => nanoid());
  const [spin, setSpin] = useState<Spin | null>(null);
  const [pending, setPending] = useState(false);
  const [spinning, setSpinning] = useState(false);

  const x = useMotionValue(reelOffset(START_INDEX));
  // 正在播放的滚动和它停下的位置, 跳过时直接停到这个位置
  const playback = useRef<Playback | null>(null);
  const restAt = useRef(reelOffset(START_INDEX));

  useEffect(() => {
    let alive = true;
    services.crate.getCrate(itemID)
        .then((data) => {
          if (!alive) return;
          setView(data);
          setQuantity(data.quantity);
        })
        .catch((err) => alive && setError(toErrorMessage(err)));
    return () => {
      alive = false;
    };
  }, [itemID]);

  // 关闭时停掉还在播放的滚动
  useEffect(() => () => playback.current?.stop(), []);

  const idleTiles = useMemo(() => (view ? idleReel(view.prizes) : []), [view]);

  const settle = () => {
    setSpinning(false);
    onLockedChange(false);
  };

  // 连续点击时可能发出两次请求, 幂等键相同, 只会开启一次
  const handleOpen = async () => {
    if (!view || pending) return;
    setPending(true);
    onLockedChange(true);
    let result: CrateOpenResult;
    try {
      result = await services.crate.open(view.item_id, idempotencyKey);
    } catch (err) {
      toast.error(toErrorMessage(err));
      setPending(false);
      onLockedChange(false);
      return;
    }
    // 结果已经结算, 下面只是揭晓
    setPending(false);
    setIdempotencyKey(nanoid());
    setQuantity(result.quantity);
    setSpin({tiles: buildReel(view.prizes, result.prize), result});
    onOpened();

    const to = reelOffset(WIN_INDEX, randomJitter());
    restAt.current = to;
    if (reduced) {
      // 减少动态效果时不滚动, 直接显示结果
      x.set(to);
      settle();
      return;
    }
    x.set(reelOffset(START_INDEX));
    setSpinning(true);
    playback.current = animate(x, to, {duration: SPIN_SECONDS, ease: SPIN_EASE, onComplete: settle});
  };

  const skip = () => {
    playback.current?.stop();
    x.set(restAt.current);
    settle();
  };

  if (error) {
    return (
      <>
        <p className="rounded-lg border bg-card px-4 py-6 text-center text-sm text-muted-foreground">{error}</p>
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            关闭
          </Button>
        </DialogFooter>
      </>
    );
  }
  if (!view) {
    return (
      <>
        <Skeleton className="h-36 rounded-xl" />
        <Skeleton className="h-14 rounded-lg" />
        <Skeleton className="h-32 rounded-lg" />
      </>
    );
  }

  const revealed = spin && !spinning ? spin.result : null;
  return (
    <>
      <CrateReel tiles={spin?.tiles ?? idleTiles} x={x} wonIndex={revealed ? WIN_INDEX : null} />

      <div role="status" aria-live="polite" className="flex min-h-14 items-center justify-center rounded-lg border bg-card px-4 py-2">
        {revealed ? (
          <RevealedPrize key={revealed.id} prize={revealed.prize} animated={!reduced} />
        ) : (
          <span className="text-sm text-muted-foreground">
            {spinning || pending ? '正在开启…' : quantity > 0 ? '点击开启, 揭晓这一次开出的内容' : '背包中没有这个箱子了'}
          </span>
        )}
      </div>

      <PrizeList view={view} />

      <DialogFooter className="sm:items-center">
        <span className="text-sm text-muted-foreground sm:mr-auto">
          剩余 <span className="font-mono tabular-nums text-foreground">×{quantity}</span>
        </span>
        {spinning ? (
          <Button variant="outline" onClick={skip}>
            <FastForward />
            跳过
          </Button>
        ) : (
          <>
            <Button variant="outline" onClick={onClose} disabled={pending}>
              {spin ? '收下' : '取消'}
            </Button>
            <SubmitButton type="button" pending={pending} disabled={quantity < 1} onClick={handleOpen}>
              {!pending && <PackageOpen />}
              {spin ? '再开一个' : '开启'}
            </SubmitButton>
          </>
        )}
      </DialogFooter>
    </>
  );
}

/**
 * 揭晓的结果: 抽中的货币或物品和数量
 */
function RevealedPrize({prize, animated}: { prize: RewardLine; animated: boolean }) {
  const q = useQualities().qualityOf(prize.quality);
  return (
    <motion.div
      className="flex min-w-0 items-center gap-3"
      initial={animated ? {opacity: 0, y: 8} : false}
      animate={{opacity: 1, y: 0}}
      transition={{duration: 0.3}}
    >
      <span className="shrink-0 text-sm text-muted-foreground">获得</span>
      <span className="flex size-9 shrink-0 items-center justify-center overflow-hidden rounded-md border bg-background p-0.5">
        {prize.currency ? <CurrencyIcon currency={prize.currency} className="size-5" /> : <ItemImage item={prize} />}
      </span>
      <span className={cn('truncate text-lg font-semibold', prize.item_id && q.text)} title={prize.name}>
        {prize.name}
      </span>
      <span className="shrink-0 font-mono tabular-nums">{amountText(prize)}</span>
    </motion.div>
  );
}

/**
 * 箱内物品: 每一项的数量和概率
 */
function PrizeList({view}: { view: CrateView }) {
  const {qualityOf} = useQualities();
  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-baseline justify-between gap-2">
        <span className="text-sm font-medium">箱内物品</span>
        <span className="text-xs text-muted-foreground">开启一次得到其中一项</span>
      </div>
      <ul className="grid max-h-44 grid-cols-1 gap-1.5 overflow-y-auto sm:grid-cols-2">
        {view.prizes.map((prize, i) => (
          <li key={i} className="flex items-center gap-2 rounded-md border bg-card px-2 py-1.5 text-sm">
            <span className="flex size-7 shrink-0 items-center justify-center overflow-hidden rounded-md border bg-background p-0.5">
              {prize.currency ? <CurrencyIcon currency={prize.currency} className="size-4" /> : <ItemImage item={prize} />}
            </span>
            <span className={cn('min-w-0 flex-1 truncate', prize.item_id && qualityOf(prize.quality).text)} title={prize.name}>
              {prize.name}
            </span>
            <span className="shrink-0 font-mono text-xs tabular-nums text-muted-foreground">{amountText(prize, prize.max_amount)}</span>
            <span className="w-14 shrink-0 text-right font-mono text-xs tabular-nums">{formatProbability(prize.probability)}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}
