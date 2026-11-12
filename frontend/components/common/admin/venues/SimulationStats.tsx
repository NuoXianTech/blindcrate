import {CircleCheck, CircleX} from 'lucide-react';
import {cn} from 'cn';
import type {Balance, BalanceCheck, CheckCode, Simulation, Squad, SquadStats, ValueStats} from '@/lib/services/venue';
import type {Currency} from '@/lib/services/wallet';
import {formatAmount, formatDateTime} from '@/lib/utils';

/** 摆放失败率的上限, 基点, 与后端 venue.maxFailureRate 一致 */
const MAX_FAILURE_RATE = 100;

/** 阵容的名称, 与后端 venue 的 squadNames 一致 */
const SQUAD_NAMES: Record<Squad, string> = {
  humans: '4 个真人',
  ranked: '排位补人机',
  pve: '人机模式',
};

/** 闸门的名称, 与后端 venue 的 checkNames 一致 */
const CHECK_NAMES: Record<CheckCode, string> = {
  rounds: '轮数',
  grid_width: '仓库宽度',
  grid_height: '仓库高度',
  top_share: '最贵一件的占比',
  price_value: '成交价与价值之比',
  bid_cv: '出价离散度',
  speed_win: '速胜率',
  appraise_bias: '人机估值偏差',
  seat_ev: '真人盈亏',
  win_rate: '真人胜率',
};

/** 基点转为百分比文本 */
const percent = (bp: number) => `${(bp / 100).toFixed(1)}%`;

/**
 * 仓库模拟结果: 排位和人机两种模式的价值分布, 以及对局验算
 */
export function SimulationStats({simulation, currency, stale, lots, tierNames}: {
  simulation: Simulation;
  /** 一局多个拍品 */
  lots: boolean;
  /** 档次代码对应的名称, 找不到时显示代码 */
  tierNames: Record<string, string>;
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
    {label: '离散度', value: (s) => ((s.cv ?? 0) / 10000).toFixed(2)},
  ];

  // 只有一种档次时和上表相同, 不再单独列出
  const tiers = simulation.tiers.length > 1 ? simulation.tiers : [];

  return (
    <div className={cn('flex flex-col gap-3', stale && 'opacity-60')}>
      <div className="flex flex-col gap-2 rounded-lg border p-3 text-sm">
        <div className="flex items-center justify-between gap-2 text-xs text-muted-foreground">
          <span>
            模拟 {simulation.runs} 个仓库, 价值为{currency?.name ?? '出价货币'}的回收价之和
          </span>
          <span>{stale ? '配置已修改, 需要重新模拟' : formatDateTime(simulation.simulated_at)}</span>
        </div>
        {tiers.length > 0 && <span className="text-xs text-muted-foreground">下表为单个拍品按权重混合各档次的分布</span>}
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
      {tiers.length > 0 && (
        <div className="flex flex-col gap-2 rounded-lg border p-3 text-sm">
          <span className="font-medium">各档次(排位)</span>
          <div className="overflow-x-auto">
            <div
              className="grid gap-x-6 gap-y-1"
              style={{gridTemplateColumns: `minmax(7rem, 1fr) repeat(${tiers.length}, auto)`}}
            >
              <span />
              {tiers.map((t) => (
                <span key={t.code} className="text-right text-xs text-muted-foreground">
                  {tierNames[t.code] ?? t.code}
                </span>
              ))}
              {rows.map((row) => (
                <div key={row.label} className="contents">
                  <span className="text-muted-foreground">{row.label}</span>
                  {tiers.map((t) => (
                    <span key={t.code} className="text-right font-mono tabular-nums">
                      {row.value(t.ranked)}
                    </span>
                  ))}
                </div>
              ))}
              <span className="text-muted-foreground">人机平均价值</span>
              {tiers.map((t) => (
                <span key={t.code} className="text-right font-mono tabular-nums">
                  {amount(t.pve.mean)}
                </span>
              ))}
            </div>
          </div>
        </div>
      )}
      {simulation.balance ? (
        <BalanceStats balance={simulation.balance} amount={amount} lots={lots} />
      ) : (
        <p className="rounded-lg border p-3 text-sm text-muted-foreground">旧数据没有对局验算, 重新模拟或保存后生成</p>
      )}
    </div>
  );
}

/**
 * 对局验算: 各阵容的结果和闸门
 */
function BalanceStats({balance, amount, lots}: {
  balance: Balance;
  amount: (value: number) => string;
  /** 一局多个拍品; 胜率闸门看拍场第 1 名的比例 */
  lots: boolean;
}) {
  const rows: { label: string; value: (s: SquadStats) => React.ReactNode }[] = [
    ...(lots ? [{label: '平均拍品数', value: (s: SquadStats) => ((s.lots ?? 0) / 100).toFixed(2)}] : []),
    {label: '成交价 ÷ 价值', value: (s) => percent(s.price_value)},
    {label: '速胜率', value: (s) => percent(s.speed_win)},
    {label: '出价离散度', value: (s) => (s.bid_cv / 10000).toFixed(3)},
    {label: '最贵一件占比', value: (s) => percent(s.top_share)},
    {label: '人机估值 ÷ 价值', value: (s) => (s.humans < 4 ? percent(s.appraise_bias) : '-')},
    {label: '平均结束轮次', value: (s) => (s.avg_rounds / 100).toFixed(2)},
    {label: lots ? '真人每拍品成交率' : '真人成交率', value: (s) => percent(s.win_rate)},
    ...(lots ?
      [
        {label: '拍场第 1 名率', value: (s: SquadStats) => percent(s.rank_win ?? 0)},
        {label: '资金耗尽率', value: (s: SquadStats) => percent(s.budget_out ?? 0)},
        {label: '规划预算多赚', value: (s: SquadStats) => (s.plan_gain ? amount(s.plan_gain) : '-')},
      ] :
      []),
    {label: '末轮前出价占比', value: (s) => (s.early_share ? percent(s.early_share) : '-')},
    {label: '真人每局盈亏', value: (s) => amount(s.seat_ev)},
    {label: '成交时盈亏', value: (s) => amount(s.winner_ev)},
    {label: '热度 火热 / 冷清', value: (s) => `${percent(s.hype_hot ?? 0)} / ${percent(s.hype_cold ?? 0)}`},
  ];
  const failed = balance.checks.filter((c) => !c.passed);

  return (
    <div className="flex flex-col gap-2 rounded-lg border p-3 text-sm">
      <div className="flex items-center justify-between gap-2">
        <span className="font-medium">对局验算</span>
        <span className={cn('flex items-center gap-1 text-xs', balance.passed ? 'text-quality-green' : 'text-destructive')}>
          {balance.passed ? <CircleCheck className="size-3.5" /> : <CircleX className="size-3.5" />}
          {balance.passed ? '全部通过' : `${failed.length} 项未通过, 不能启用`}
        </span>
      </div>
      <p className="text-xs text-muted-foreground">
        每种阵容模拟 {balance.runs} 局; 真人按模拟的估值误差出价, 人机按地图配置的风格出价; 成交价 ÷ 价值的目标为{' '}
        {percent(balance.target)}(扣掉出售手续费), 藏品按扣掉手续费后的回收价计算盈亏
      </p>
      <div className="overflow-x-auto">
        <div
          className="grid gap-x-6 gap-y-1"
          style={{gridTemplateColumns: `minmax(7rem, 1fr) repeat(${balance.squads.length}, auto)`}}
        >
          <span />
          {balance.squads.map((s) => (
            <span key={s.squad} className="text-right text-xs text-muted-foreground">
              {SQUAD_NAMES[s.squad] ?? s.squad}
            </span>
          ))}
          {rows.map((row) => (
            <div key={row.label} className="contents">
              <span className="text-muted-foreground">{row.label}</span>
              {balance.squads.map((s) => (
                <span key={s.squad} className="text-right font-mono tabular-nums">
                  {row.value(s)}
                </span>
              ))}
            </div>
          ))}
        </div>
      </div>
      <div className="flex flex-col gap-1 border-t pt-2">
        <span className="text-xs text-muted-foreground">
          闸门: 结构类只看 4 个真人, 有人机的阵容看人机估值和真人盈亏{lots && '; 多拍品时真人胜率看拍场第 1 名率'}
        </span>
        {balance.checks.map((c) => (
          <CheckRow key={`${c.code}-${c.squad ?? ''}`} check={c} amount={amount} />
        ))}
      </div>
    </div>
  );
}

/**
 * 一项闸门: 名称、数值和安全区间
 */
function CheckRow({check, amount}: { check: BalanceCheck; amount: (value: number) => string }) {
  const format = (value: number) => {
    switch (check.unit) {
      case 'bp':
        return percent(value);
      case 'amount':
        return amount(value);
      default:
        return String(value);
    }
  };
  let range = '';
  if (check.min !== null && check.max !== null) range = `${format(check.min)} ~ ${format(check.max)}`;
  else if (check.min !== null) range = `≥ ${format(check.min)}`;
  else if (check.max !== null) range = `≤ ${format(check.max)}`;

  return (
    <div className={cn('flex items-center justify-between gap-3', !check.passed && 'text-destructive')}>
      <span className="flex min-w-0 items-center gap-1.5">
        {check.passed ? (
          <CircleCheck className="size-3.5 shrink-0 text-quality-green" />
        ) : (
          <CircleX className="size-3.5 shrink-0" />
        )}
        <span className="truncate">
          {CHECK_NAMES[check.code] ?? check.code}
          {check.squad && <span className="text-muted-foreground"> · {SQUAD_NAMES[check.squad] ?? check.squad}</span>}
        </span>
      </span>
      <span className="shrink-0 font-mono text-xs tabular-nums">
        {format(check.value)}
        <span className="text-muted-foreground"> / {range}</span>
      </span>
    </div>
  );
}
