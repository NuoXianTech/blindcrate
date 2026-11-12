'use client';

import {Flame, Megaphone} from 'lucide-react';
import {cn} from 'cn';
import {Badge} from '@/components/ui/badge';
import {useQualities} from '@/components/common/site';
import {HOST_PERSONAS, hypeLevel} from '@/lib/auction';
import {HOST_HINT, type AuctionView, type HostHint, type LotView} from '@/lib/services/auction';
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
 * 主持人: 公开热度和当前拍品每轮的提示(可能说错), 拍场结束后公开性格和提示的真假
 * 没有主持人提示的地图不显示
 */
export function HostPanel({view, lot}: {
  view: AuctionView;
  /** 查看的拍品, 只列出针对它的提示 */
  lot: LotView;
}) {
  const {host} = view;
  if (!host) return null;

  const hints = host.hints.filter((h) => h.lot_no === lot.lot_no);
  const lotRound = (round: number) => round - lot.first_round + 1;
  // 性格拍场结束后才公开, 之前为空
  const persona = host.persona ? HOST_PERSONAS[host.persona] : undefined;
  const level = hypeLevel(host.hype);
  const lastHype = host.hypes.at(-1);

  return (
    <section className="flex flex-col gap-3 rounded-xl border bg-card p-4">
      <div className="flex items-center justify-between gap-2">
        <span className="flex items-center gap-1.5 text-sm font-medium">
          <Megaphone className="size-4" />
          主持人
          {persona && <Badge variant="outline" className="px-1.5" title={persona.description}>{persona.label}</Badge>}
        </span>
        <span className="flex items-center gap-1 text-xs" title="每轮按加价的人数和幅度变化">
          <Flame className={cn('size-3.5', level.className)} />
          <span className={level.className}>{level.label}</span>
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
          {lastHype.lot_no !== lot.lot_no && `拍品 ${lastHype.lot_no} `}
          第 {lastHype.round - (view.lots[lastHype.lot_no - 1]?.first_round ?? lot.first_round) + 1} 轮 {lastHype.raises > 0 ? `${lastHype.raises} 人加价` : '没人加价'}
        </span>
      )}
      {hints.length > 0 ? (
        <ul className="flex flex-col gap-1.5">
          {[...hints].reverse().map((h) => (
            <li key={h.round} className="flex items-start justify-between gap-2 rounded-lg border px-3 py-2 text-sm">
              <HintText hint={h} />
              <span className="flex shrink-0 items-center gap-1.5 text-xs text-muted-foreground">
                {h.truthful !== null && (
                  <span className={h.truthful ? 'text-quality-green' : 'text-quality-red'}>{h.truthful ? '属实' : '说错了'}</span>
                )}
                第 {lotRound(h.round)} 轮
              </span>
            </li>
          ))}
        </ul>
      ) : (
        <span className="text-xs text-muted-foreground">主持人还没有开口</span>
      )}
    </section>
  );
}
