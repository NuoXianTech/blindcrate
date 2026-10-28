import {cn} from 'cn';

/** 图表中的一天 */
export interface DailyPoint {
  /** 日期, 如 2026-10-28 */
  day: string;
  value: number;
}

/** 日期的简写, 今天显示为 "今天" */
function dayLabel(day: string, today: string) {
  return day === today ? '今天' : day.slice(5);
}

/**
 * 最近几天的柱状图: 每天一根柱子, 有负值时以 0 为基线上下分开; 悬停显示当天的数值, 另附给读屏用的表格
 * diverging 为 true 时正数绿色、负数红色, 用于盈亏; 否则用主题色
 */
export function DailyChart({title, points, format, diverging}: {
  title: string;
  /** 从早到晚, 最后一天为今天 */
  points: DailyPoint[];
  format: (value: number) => string;
  diverging?: boolean;
}) {
  const today = points.at(-1)?.day ?? '';
  const maxPos = Math.max(0, ...points.map((p) => p.value));
  const maxNeg = Math.max(0, ...points.map((p) => -p.value));
  const range = maxPos + maxNeg || 1;
  // 基线距底部的比例, 全部非负时在最底部
  const baseline = (maxNeg / range) * 100;

  return (
    <figure className="flex flex-col gap-3 rounded-xl border bg-card p-4">
      <figcaption className="text-sm font-medium">{title}</figcaption>
      <div className="relative flex h-40 items-stretch gap-2" aria-hidden>
        <div className="absolute inset-x-0 border-t border-border" style={{bottom: `${baseline}%`}} />
        {points.map((p) => {
          const height = (Math.abs(p.value) / range) * 100;
          const negative = p.value < 0;
          return (
            <div key={p.day} className="group relative flex-1">
              {p.value !== 0 && (
                <div
                  className={cn(
                      'absolute inset-x-0 mx-auto max-w-10 transition-opacity group-hover:opacity-80',
                      negative ? 'rounded-b' : 'rounded-t',
                      !diverging ? 'bg-primary' : negative ? 'bg-quality-red' : 'bg-quality-green',
                  )}
                  style={negative ?
                    {top: `${100 - baseline}%`, height: `${height}%`} :
                    {bottom: `${baseline}%`, height: `${height}%`}}
                />
              )}
              <div className="pointer-events-none absolute inset-x-0 top-0 z-10 mx-auto w-max rounded-md border bg-popover px-2 py-1 text-xs whitespace-nowrap opacity-0 shadow-md transition-opacity group-hover:opacity-100">
                <span className="text-muted-foreground">{p.day}</span>
                <span className="ml-2 font-mono font-medium tabular-nums">{format(p.value)}</span>
              </div>
            </div>
          );
        })}
      </div>
      <div className="flex gap-2 text-center text-xs text-muted-foreground" aria-hidden>
        {points.map((p) => (
          <span key={p.day} className={cn('flex-1', p.day === today && 'font-medium text-foreground')}>
            {dayLabel(p.day, today)}
          </span>
        ))}
      </div>
      <table className="sr-only">
        <caption>{title}</caption>
        <tbody>
          {points.map((p) => (
            <tr key={p.day}>
              <th scope="row">{p.day}</th>
              <td>{format(p.value)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </figure>
  );
}
