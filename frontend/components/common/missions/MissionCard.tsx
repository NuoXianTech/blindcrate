import {Check, CircleCheck, LoaderCircle, X} from 'lucide-react';
import {cn} from 'cn';
import {Button} from '@/components/ui/button';
import {PriceText} from '@/components/common/reward';
import {CurrencyIcon} from '@/components/common/wallet';
import type {Mission} from '@/lib/services/mission';
import {progressRatio, valueText} from './constants';

/**
 * 任务卡片的Props接口
 */
interface MissionCardProps {
  mission: Mission;
  /** 正在领取这个任务 */
  claiming: boolean;
  /** 有任务正在领取时其他按钮也不能点 */
  disabled: boolean;
  onClaim: () => void;
}

/** 领取按钮上的文字 */
function buttonText(mission: Mission) {
  if (mission.claimed) return '已领取';
  if (mission.claimable) return '领取';
  if (mission.progress >= mission.target) return '未满足条件';
  return '未完成';
}

/**
 * 一个任务: 名称、描述、进度条、任务条件、奖励和领取按钮
 */
export function MissionCard({mission: m, claiming, disabled, onClaim}: MissionCardProps) {
  const ratio = progressRatio(m);
  return (
    <div className={cn('flex flex-col gap-3 rounded-xl border bg-card p-4 sm:flex-row sm:items-center', m.claimed && 'opacity-60')}>
      <div className="flex min-w-0 flex-1 flex-col gap-2">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="flex min-w-0 items-center gap-1.5 font-bold">
            {m.claimed && <CircleCheck className="size-4 shrink-0 text-quality-green" />}
            <span className="truncate" title={m.name}>
              {m.name}
            </span>
          </h2>
          {m.description && <p className="text-sm text-muted-foreground">{m.description}</p>}
        </div>

        <div className="flex items-center gap-3">
          <div
            className="h-2 flex-1 overflow-hidden rounded-full bg-muted"
            role="progressbar"
            aria-label={`${m.name}进度`}
            aria-valuemin={0}
            aria-valuemax={m.target}
            aria-valuenow={Math.min(Math.max(m.progress, 0), m.target)}
          >
            <div
              className={cn('h-full rounded-full transition-[width]', ratio >= 1 ? 'bg-quality-green' : 'bg-primary')}
              style={{width: `${ratio * 100}%`}}
            />
          </div>
          <span className="flex shrink-0 items-center gap-1 font-mono text-sm tabular-nums">
            {m.currency && <CurrencyIcon currency={m.currency} className="size-4" />}
            {valueText(m, m.progress)}/{valueText(m, m.target)}
          </span>
        </div>

        {m.requirements.length > 0 && (
          <ul className="flex flex-col gap-1 text-sm">
            {m.requirements.map((r, i) => (
              <li key={i} className={cn('flex items-center gap-1.5', r.met ? 'text-muted-foreground' : 'text-destructive')}>
                {r.met ? <Check className="size-4 shrink-0" /> : <X className="size-4 shrink-0" />}
                {r.text}
              </li>
            ))}
          </ul>
        )}
      </div>

      <div className="flex items-center justify-between gap-3 border-t pt-3 sm:w-56 sm:flex-col sm:items-end sm:border-t-0 sm:border-l sm:pt-0 sm:pl-4">
        <PriceText lines={m.reward} className="text-sm sm:justify-end" />
        <Button size="sm" className="shrink-0" disabled={!m.claimable || disabled} onClick={onClaim}>
          {claiming && <LoaderCircle className="animate-spin" />}
          {buttonText(m)}
        </Button>
      </div>
    </div>
  );
}
