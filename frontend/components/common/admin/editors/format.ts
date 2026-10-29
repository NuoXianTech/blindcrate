import {MODES, type ModeKind} from '@/lib/auction';
import type {AdminItem, Currency} from '@/lib/services/admin';
import type {Bundle} from '@/lib/services/auction';
import {formatAmount} from '@/lib/utils';
import type {ScopeOption} from './ConditionEditor';

/** 一组货币和物品的简短文字, 每行一项, 如 "100 筹码"、"旧怀表 ×2"; 找不到的货币显示代码, 找不到的物品显示 ID */
export function bundleText(bundle: Bundle, items: AdminItem[], currencies: Currency[]): string[] {
  return [
    ...(bundle.currencies ?? []).map((l) => {
      const currency = currencies.find((c) => c.code === l.currency);
      return `${formatAmount(l.amount, currency?.decimals ?? 0)} ${currency?.name ?? l.currency}`;
    }),
    ...(bundle.items ?? []).map((l) => `${items.find((i) => i.id === l.item_id)?.name ?? `#${l.item_id}`} ×${l.quantity}`),
  ];
}

/** 条件编辑器中统计范围的候选项: 全部对局、每种模式、每张地图和地图的每种模式 */
export function scopeOptions(venues: { code: string; name: string }[]): ScopeOption[] {
  const modes = (Object.keys(MODES) as ModeKind[]).map((kind) => ({value: kind, label: MODES[kind].label}));
  return [
    {value: 'all', label: '全部对局'},
    ...modes.map((m) => ({value: `mode:${m.value}`, label: `全部${m.label}对局`})),
    ...venues.flatMap((v) => [
      {value: `venue:${v.code}`, label: v.name},
      ...modes.map((m) => ({value: `venue:${v.code}:${m.value}`, label: `${v.name} ${m.label}`})),
    ]),
  ];
}
