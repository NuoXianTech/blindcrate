import {nanoid} from 'nanoid';
import {
  decimalsOf,
  fromBundleDrafts,
  fromConditionDrafts,
  toBundleDrafts,
  toConditionDrafts,
  type BundleDraft,
  type ConditionDraft,
} from '@/components/common/admin/editors';
import {check} from '@/hooks/use-submit';
import type {ConditionType, Currency, VenueRequest} from '@/lib/services/admin';
import {amountInput, parseAmount} from '@/lib/utils';
import {checkImageURL} from '@/lib/validation';
import {
  BOT_LEVELS,
  CODE_PATTERN,
  MAX_ALLOWED_ITEMS,
  MAX_ENTRY_COST_LINES,
  MAX_REQUIREMENTS,
  MAX_ROUND_FACTORS,
  MAX_SLOTS,
  MAX_USES,
} from './constants';

/** 基点按百分比输入, 保留 2 位小数 */
const PERCENT = 2;
/** 基点按倍数输入, 保留 4 位小数 */
const MULTIPLE = 4;
/** 金额上限, 最小单位 */
const MAX_AMOUNT = 1_000_000_000_000;

/** 编辑中的一个档位, 出现概率按百分比输入 */
export interface SlotDraft {
  key: string;
  pool: string;
  min: string;
  max: string;
  chance: string;
}

/**
 * 地图表单, 数值保留输入框里的原文, 提交时由 toVenueRequest 统一换算和校验
 * 金额按出价货币的小数位输入, 基点按百分比输入, 速胜倍率按倍数输入
 */
export interface VenueForm {
  code: string;
  name: string;
  description: string;
  imageURL: string;
  sortOrder: string;
  isEnabled: boolean;
  pveEnabled: boolean;
  rankedEnabled: boolean;
  bidCurrency: string;
  entryCost: BundleDraft[];
  requirements: ConditionDraft[];
  gridWidth: string;
  gridHeight: string;
  slots: SlotDraft[];
  maxRerolls: string;
  startPrice: string;
  maxCarry: string;
  rounds: string;
  roundSeconds: string;
  revealSeconds: string;
  /** 长度保持为轮数减 1 */
  quickWin: string[];
  maxExtraRounds: string;
  levelWeights: string[];
  estimateErrors: string[];
  roundFactors: string[];
  jitter: string;
  maxFactor: string;
  bluffChance: string;
  bluffRounds: string;
  bluffFactor: string;
  practiceOnly: boolean;
  dailyProfitCap: string;
  valueFactor: string;
  /** 每局最多使用的道具数, 0 表示这个模式不能用道具 */
  usesPve: string;
  usesRanked: string;
  /** 允许使用的道具 ID, 为空表示都可以用 */
  allowedItems: number[];
}

/** 新建地图时的默认配置, 与老城区仓库的种子数据相近; 出价货币见 toVenueForm */
const DEFAULT_REQUEST: VenueRequest = {
  code: '',
  name: '',
  description: '',
  image_url: '',
  sort_order: 0,
  is_enabled: false,
  pve_enabled: true,
  ranked_enabled: true,
  bid_currency: '',
  entry_cost: {},
  requirements: [],
  warehouse: {grid: {width: 6, height: 5}, slots: [{pool: '', min: 4, max: 8, chance: 10000}], max_rerolls: 10},
  auction: {
    start_price: 100,
    max_carry: 5000,
    rounds: 3,
    round_seconds: 20,
    reveal_seconds: 10,
    quick_win: [20000, 15000],
    max_extra_rounds: 2,
  },
  bots: {
    level_weights: [40, 40, 20],
    estimate_errors: [4000, 2000, 800],
    round_factors: [4000, 6000, 8500],
    jitter: 1000,
    max_factor: 9500,
    bluff_chance: 500,
    bluff_rounds: 1,
    bluff_factor: 9000,
  },
  pve: {practice_only: false, daily_profit_cap: 0, value_factor: 8000},
  consumables: {pve: {max_per_match: 0}, ranked: {max_per_match: 0}, allowed: []},
};

/** 基点或倍数转为输入框的原文, 去掉多余的 0 */
function ratioInput(value: number, decimals: number): string {
  return String(value / 10 ** decimals);
}

/**
 * 地图转为表单, venue 为空时新建
 * 新建时出价货币为基础货币, 基础货币不存在时换成第一个货币, 金额按它的小数位换算
 */
export function toVenueForm(venue: VenueRequest | null, currencies: Currency[], baseCurrency: string): VenueForm {
  const v = venue ?? {
    ...DEFAULT_REQUEST,
    bid_currency: currencies.some((c) => c.code === baseCurrency) ? baseCurrency : currencies[0]?.code ?? '',
  };
  const decimals = decimalsOf(currencies, v.bid_currency);
  return {
    code: v.code,
    name: v.name,
    description: v.description,
    imageURL: v.image_url,
    sortOrder: String(v.sort_order),
    isEnabled: v.is_enabled,
    pveEnabled: v.pve_enabled,
    rankedEnabled: v.ranked_enabled,
    bidCurrency: v.bid_currency,
    entryCost: toBundleDrafts(v.entry_cost, currencies),
    requirements: toConditionDrafts(v.requirements ?? []),
    gridWidth: String(v.warehouse.grid.width),
    gridHeight: String(v.warehouse.grid.height),
    slots: v.warehouse.slots.map((s) => ({
      key: nanoid(),
      pool: s.pool,
      min: String(s.min),
      max: String(s.max),
      chance: ratioInput(s.chance, PERCENT),
    })),
    maxRerolls: String(v.warehouse.max_rerolls),
    startPrice: amountInput(v.auction.start_price, decimals),
    maxCarry: amountInput(v.auction.max_carry, decimals),
    rounds: String(v.auction.rounds),
    roundSeconds: String(v.auction.round_seconds),
    revealSeconds: String(v.auction.reveal_seconds),
    quickWin: resizeQuickWin(v.auction.quick_win.map((x) => ratioInput(x, MULTIPLE)), v.auction.rounds),
    maxExtraRounds: String(v.auction.max_extra_rounds),
    levelWeights: v.bots.level_weights.map(String),
    estimateErrors: v.bots.estimate_errors.map((x) => ratioInput(x, PERCENT)),
    roundFactors: v.bots.round_factors.map((x) => ratioInput(x, PERCENT)),
    jitter: ratioInput(v.bots.jitter, PERCENT),
    maxFactor: ratioInput(v.bots.max_factor, PERCENT),
    bluffChance: ratioInput(v.bots.bluff_chance, PERCENT),
    bluffRounds: String(v.bots.bluff_rounds),
    bluffFactor: ratioInput(v.bots.bluff_factor, PERCENT),
    practiceOnly: v.pve.practice_only,
    dailyProfitCap: amountInput(v.pve.daily_profit_cap, decimals),
    valueFactor: ratioInput(v.pve.value_factor, PERCENT),
    usesPve: String(v.consumables?.pve.max_per_match ?? 0),
    usesRanked: String(v.consumables?.ranked.max_per_match ?? 0),
    allowedItems: v.consumables?.allowed ?? [],
  };
}

/** 速胜倍率的个数跟随轮数, 多出的去掉, 不够的沿用最后一个 */
export function resizeQuickWin(quickWin: string[], rounds: number): string[] {
  const size = Math.max(rounds - 1, 0);
  const resized = quickWin.slice(0, size);
  while (resized.length < size) resized.push(resized.at(-1) ?? '1.5');
  return resized;
}

/** 数值输入的校验参数, min 和 max 为换算后的整数 */
interface NumberRule {
  label: string;
  min: number;
  max: number;
  /** 输入的小数位, 如百分比为 2 */
  decimals?: number;
  /** 提示中范围后面的单位 */
  unit?: string;
}

/** 输入框的原文换算为整数, 格式不对或超出范围时中断提交 */
function readNumber(text: string, {label, min, max, decimals = 0, unit = ''}: NumberRule): number {
  const value = parseAmount(text, decimals);
  const range = `${ratioInput(min, decimals)}-${ratioInput(max, decimals)}${unit}`;
  check(value === null || value < min || value > max ? `${label}需为 ${range}` : null);
  return value as number;
}

const percent = (text: string, label: string, min: number, max: number) =>
  readNumber(text, {label, min, max, decimals: PERCENT, unit: '%'});

/**
 * 表单转为新建、修改或模拟接口的请求体, 校验不通过时中断提交
 * 只做格式和范围的校验, 速胜倍率、摆放失败率等和其他配置相关的规则由后端校验
 */
export function toVenueRequest(form: VenueForm, currencies: Currency[], types: ConditionType[]): VenueRequest {
  const code = form.code.trim();
  check(CODE_PATTERN.test(code) ? null : '代码需为 1-24 个小写字母、数字或下划线');
  const name = form.name.trim();
  check(name.length === 0 || [...name].length > 64 ? '名称需为 1-64 个字符' : null);
  const description = form.description.trim();
  check([...description].length > 255 ? '描述最多 255 个字符' : null);
  check(checkImageURL(form.imageURL.trim()));
  check(!form.bidCurrency ? '请选择出价货币' : null);
  const decimals = decimalsOf(currencies, form.bidCurrency);
  const amount = (text: string, label: string, min: number) => readNumber(text, {label, min, max: MAX_AMOUNT, decimals});

  check(form.entryCost.length > MAX_ENTRY_COST_LINES ? `入场代价最多 ${MAX_ENTRY_COST_LINES} 项` : null);
  check(form.requirements.length > MAX_REQUIREMENTS ? `入场条件最多 ${MAX_REQUIREMENTS} 个` : null);
  check(form.slots.length === 0 || form.slots.length > MAX_SLOTS ? `档位需为 1-${MAX_SLOTS} 个` : null);
  check(form.roundFactors.length === 0 || form.roundFactors.length > MAX_ROUND_FACTORS ?
    `每轮出价系数需为 1-${MAX_ROUND_FACTORS} 个` :
    null);

  check(form.allowedItems.length > MAX_ALLOWED_ITEMS ? `允许的道具最多 ${MAX_ALLOWED_ITEMS} 个` : null);

  const rounds = readNumber(form.rounds, {label: '轮数', min: 1, max: 10});
  return {
    code,
    name,
    description,
    image_url: form.imageURL.trim(),
    sort_order: readNumber(form.sortOrder, {label: '排序', min: -1_000_000, max: 1_000_000}),
    is_enabled: form.isEnabled,
    pve_enabled: form.pveEnabled,
    ranked_enabled: form.rankedEnabled,
    bid_currency: form.bidCurrency,
    entry_cost: fromBundleDrafts(form.entryCost, currencies, '入场代价'),
    requirements: fromConditionDrafts(form.requirements, types, '入场条件'),
    warehouse: {
      grid: {
        width: readNumber(form.gridWidth, {label: '仓库宽度', min: 1, max: 16}),
        height: readNumber(form.gridHeight, {label: '仓库高度', min: 1, max: 16}),
      },
      slots: form.slots.map((s, i) => {
        const row = `第 ${i + 1} 档`;
        check(!s.pool ? `${row}: 请选择掉落池` : null);
        const min = readNumber(s.min, {label: `${row}: 最少次数`, min: 0, max: 100});
        const max = readNumber(s.max, {label: `${row}: 最多次数`, min: 0, max: 100});
        check(max < min ? `${row}: 最多次数不能少于最少次数` : null);
        return {pool: s.pool, min, max, chance: percent(s.chance, `${row}: 出现概率`, 1, 10000)};
      }),
      max_rerolls: readNumber(form.maxRerolls, {label: '重抽次数上限', min: 0, max: 50}),
    },
    auction: {
      start_price: amount(form.startPrice, '起拍价', 1),
      max_carry: amount(form.maxCarry, '带入资金上限', 1),
      rounds,
      round_seconds: readNumber(form.roundSeconds, {label: '每轮时长', min: 5, max: 120, unit: ' 秒'}),
      reveal_seconds: readNumber(form.revealSeconds, {label: '开箱时长', min: 3, max: 60, unit: ' 秒'}),
      quick_win: form.quickWin.map((x, i) =>
        readNumber(x, {label: `第 ${i + 1} 轮速胜倍率`, min: 10000, max: 100000, decimals: MULTIPLE, unit: ' 倍'}),
      ),
      max_extra_rounds: readNumber(form.maxExtraRounds, {label: '加轮上限', min: 0, max: 5}),
    },
    bots: {
      level_weights: form.levelWeights.map((x, i) => readNumber(x, {label: `${BOT_LEVELS[i]}级人机的权重`, min: 0, max: 100})),
      estimate_errors: form.estimateErrors.map((x, i) => percent(x, `${BOT_LEVELS[i]}级人机的估值误差`, 0, 9000)),
      round_factors: form.roundFactors.map((x, i) => percent(x, `第 ${i + 1} 轮出价系数`, 100, 10000)),
      jitter: percent(form.jitter, '出价浮动', 0, 5000),
      max_factor: percent(form.maxFactor, '出价上限系数', 100, 10000),
      bluff_chance: percent(form.bluffChance, '诈唬概率', 0, 10000),
      bluff_rounds: readNumber(form.bluffRounds, {label: '诈唬轮数', min: 0, max: 10}),
      bluff_factor: percent(form.bluffFactor, '诈唬系数', 100, 10000),
    },
    pve: {
      practice_only: form.practiceOnly,
      daily_profit_cap: amount(form.dailyProfitCap, '每日收益上限', 0),
      value_factor: percent(form.valueFactor, '仓库价值系数', 1000, 10000),
    },
    consumables: {
      pve: {max_per_match: readNumber(form.usesPve, {label: '人机每局道具数', min: 0, max: MAX_USES})},
      ranked: {max_per_match: readNumber(form.usesRanked, {label: '排位每局道具数', min: 0, max: MAX_USES})},
      allowed: form.allowedItems,
    },
  };
}

/** 表单各页签共用的Props */
export interface FormTabProps {
  form: VenueForm;
  onChange: (patch: Partial<VenueForm>) => void;
  disabled: boolean;
}
