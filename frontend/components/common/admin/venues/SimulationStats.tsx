import {cn} from 'cn';
import type {Currency, Simulation, ValueStats} from '@/lib/services/admin';
import {formatAmount, formatDateTime} from '@/lib/utils';

/** 摆放失败率的上限, 基点, 与后端 venue.maxFailureRate 一致 */
const MAX_FAILURE_RATE = 100;

/**
 * 仓库模拟结果: 排位和人机两种模式的价值分布
 */
export function SimulationStats({simulation, currency, stale}: {
  simulation: Simulation;
  /** 出价货币, 价值按它的小数位展示 */
  currency?: Currency;
  /** 表单改动后结果已过期 */
  stale: boolean;
}) {
  const decimals = currency?.decimals ?? 0;
  const amount = (value: number) => formatAmount(value, decimals);
  const rows: { label: string; value: (s: ValueStats) => React.ReactNode }[] = [
    {
      label: '摆放失败率',
      value: (s) => (
        <span className={cn(s.failure_rate > MAX_FAILURE_RATE && 'text-destructive')}>{(s.failure_rate / 100).toFixed(2)}%</span>
      ),
    },
    {label: '平均件数', value: (s) => s.mean_pieces},
    {label: 'P10', value: (s) => amount(s.p10)},
    {label: 'P50', value: (s) => amount(s.p50)},
    {label: 'P90', value: (s) => amount(s.p90)},
    {label: '平均价值', value: (s) => amount(s.mean)},
    {label: '最高价值', value: (s) => amount(s.max)},
  ];

  return (
    <div className={cn('flex flex-col gap-2 rounded-lg border p-3 text-sm', stale && 'opacity-60')}>
      <div className="flex items-center justify-between gap-2 text-xs text-muted-foreground">
        <span>
          模拟 {simulation.runs} 个仓库, 价值为{currency?.name ?? '出价货币'}的回收价之和
        </span>
        <span>{stale ? '配置已修改, 需要重新模拟' : formatDateTime(simulation.simulated_at)}</span>
      </div>
      <div className="grid grid-cols-[1fr_auto_auto] gap-x-6 gap-y-1">
        <span />
        <span className="text-right text-xs text-muted-foreground">排位</span>
        <span className="text-right text-xs text-muted-foreground">人机</span>
        {rows.map((row) => (
          <div key={row.label} className="contents">
            <span className="text-muted-foreground">{row.label}</span>
            <span className="text-right font-mono tabular-nums">{row.value(simulation.ranked)}</span>
            <span className="text-right font-mono tabular-nums">{row.value(simulation.pve)}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
