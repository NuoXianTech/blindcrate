import {cn} from 'cn';
import {HINTS} from '@/lib/auction';
import type {Feedback} from '@/lib/services/auction';

/**
 * 每轮结束后自己的排名和提示, 不透露其他人的出价, 最新的一轮在最上面
 */
export function FeedbackList({feedback}: { feedback: Feedback[] }) {
  if (!feedback.length) {
    return <p className="text-sm text-muted-foreground">每轮结束后, 这里会显示你的出价排名和与最高价的差距</p>;
  }
  return (
    <ul className="flex flex-col gap-1.5 text-sm">
      {[...feedback].reverse().map((f) => {
        const hint = HINTS[f.hint] ?? {label: f.hint, className: ''};
        return (
          <li key={f.round} className="flex items-center justify-between gap-3 rounded-lg border bg-card px-3 py-2">
            <span className="text-muted-foreground">第 {f.round} 轮</span>
            <span className="flex items-center gap-3">
              {f.rank > 0 && <span className="font-mono tabular-nums">第 {f.rank} 名</span>}
              <span className={cn('font-medium', hint.className)}>{hint.label}</span>
            </span>
          </li>
        );
      })}
    </ul>
  );
}
