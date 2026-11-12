import {nanoid} from 'nanoid';
import {
  decimalsOf,
  fromBundleDrafts,
  fromConditionDrafts,
  PERCENT_DECIMALS,
  ratioInput,
  toBundleDrafts,
  toConditionDrafts,
  type BundleDraft,
  type ConditionDraft,
} from '@/components/common/admin/editors';
import {check} from '@/hooks/use-submit';
import type {BotPersona} from '@/lib/services/bot';
import type {ConditionType} from '@/lib/services/condition';
import type {BotWeight, VenueRequest, WarehouseRule} from '@/lib/services/venue';
import type {Currency} from '@/lib/services/wallet';
import {amountInput, parseAmount} from '@/lib/utils';
import {checkImageURL} from '@/lib/validation';
import {
  CODE_PATTERN,
  MAX_ALLOWED_ITEMS,
  MAX_BOT_PERSONAS,
  MAX_ENTRY_COST_LINES,
  MAX_GAP_SECONDS,
  MAX_LIE_RATE,
  MAX_LOTS,
  MAX_REQUIREMENTS,
  MAX_SLOTS,
  MAX_SWINGS,
  MAX_TIERS,
  MAX_USES,
} from './constants';

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

/** 编辑中的一档数量波动, 系数按百分比输入 */
export interface SwingDraft {
  key: string;
  factor: string;
  weight: string;
}

/** 编辑中的仓库生成规则, 每个档次一个 */
export interface WarehouseDraft {
  gridWidth: string;
  gridHeight: string;
  slots: SlotDraft[];
  swings: SwingDraft[];
  maxRerolls: string;
}

/** 编辑中的一种拍品档次 */
export interface TierDraft {
  key: string;
  code: string;
  name: string;
  weight: string;
  warehouse: WarehouseDraft;
}

/** 编辑中的一种出场的人机风格 */
export interface WeightDraft {
  key: string;
  persona: string;
  weight: string;
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
  lotMin: string;
  /** 为 1 时一局一个仓库, 带入资金按单个仓库的价值推导 */
  lotMax: string;
  /** 至少一种 */
  tiers: TierDraft[];
  gapSeconds: string;
  /** 百分比, 0 表示用玩法平衡设置的默认值; 多拍品时生效 */
  coverage: string;
  rounds: string;
  roundSeconds: string;
  revealSeconds: string;
  /** 长度保持为轮数减 1 */
  quickWin: string[];
  maxExtraRounds: string;
  /** 每个模式出场的人机风格和权重 */
  botsPve: WeightDraft[];
  botsRanked: WeightDraft[];
  practiceOnly: boolean;
  dailyProfitCap: string;
  valueFactor: string;
  /** 每局最多使用的道具数, 0 表示这个模式不能用道具 */
  usesPve: string;
  usesRanked: string;
  /** 允许使用的道具 ID, 为空表示都可以用 */
  allowedItems: number[];
  hostEnabled: boolean;
  /** 为空表示全部性格 */
  hostPersonas: string[];
  /** 百分比 */
  lieRate: string;
  /** 品质等级, 0 表示不区分 */
  highQuality: number;
}

/** 推荐的速胜倍率, 第 1 轮起, 更多轮次沿用最后一个; 在交接包的标准图上验算过的倍率 × 1.15 */
const RECOMMENDED_QUICK_WIN = [23000, 18975, 18630, 17825, 17480, 17250, 17020];

/** 轮数对应的推荐速胜倍率, 按倍数输入 */
export function recommendedQuickWin(rounds: number): string[] {
  return Array.from({length: Math.max(rounds - 1, 0)}, (_, i) =>
    ratioInput(RECOMMENDED_QUICK_WIN[Math.min(i, RECOMMENDED_QUICK_WIN.length - 1)], MULTIPLE),
  );
}

/** 新建地图时第一种档次的仓库 */
const DEFAULT_WAREHOUSE: WarehouseRule = {
  grid: {width: 6, height: 5},
  slots: [{pool: '', min: 4, max: 8, chance: 10000}],
  swings: [],
  max_rerolls: 10,
};

/** 仓库生成规则转为表单 */
function toWarehouseDraft(w: WarehouseRule): WarehouseDraft {
  return {
    gridWidth: String(w.grid.width),
    gridHeight: String(w.grid.height),
    slots: w.slots.map((s) => ({
      key: nanoid(),
      pool: s.pool,
      min: String(s.min),
      max: String(s.max),
      chance: ratioInput(s.chance, PERCENT_DECIMALS),
    })),
    swings: w.swings.map((s) => ({
      key: nanoid(),
      factor: ratioInput(s.factor, PERCENT_DECIMALS),
      weight: String(s.weight),
    })),
    maxRerolls: String(w.max_rerolls),
  };
}

/** 新建地图时的默认配置; 出价货币和人机风格见 toVenueForm */
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
  lots: {
    min: 1,
    max: 1,
    tiers: [{code: 'tier1', name: '档次 1', weight: 1, warehouse: DEFAULT_WAREHOUSE}],
    gap_seconds: 0,
    coverage: 0,
  },
  auction: {
    rounds: 5,
    round_seconds: 20,
    reveal_seconds: 10,
    quick_win: RECOMMENDED_QUICK_WIN.slice(0, 4),
    max_extra_rounds: 1,
  },
  // 种子数据中的三种风格
  bots: {
    pve: [{persona: 'novice', weight: 50}, {persona: 'steady', weight: 30}, {persona: 'aggressive', weight: 20}],
    ranked: [{persona: 'steady', weight: 50}, {persona: 'aggressive', weight: 30}, {persona: 'novice', weight: 20}],
  },
  pve: {practice_only: false, daily_profit_cap: 0, value_factor: 8000},
  consumables: {pve: {max_per_match: 0}, ranked: {max_per_match: 0}, allowed: []},
  host: {enabled: true, personas: [], lie_rate: 1800, high_quality: 0},
};

/** 旧地图没有主持人配置时的表单值: 不开启 */
const EMPTY_HOST = {enabled: false, personas: [], lie_rate: 1800, high_quality: 0};

/** 出场的人机风格转为表单 */
function toWeightDrafts(weights: BotWeight[]): WeightDraft[] {
  return weights.map((w) => ({key: nanoid(), persona: w.persona, weight: String(w.weight)}));
}

/** 表单换算用到的选项, 由列表页拉取一次 */
export interface VenueFormOptions {
  currencies: Currency[];
  /** 条件类型, 提供条件参数的表单描述 */
  types: ConditionType[];
  personas: BotPersona[];
  baseCurrency: string;
}

/**
 * 地图转为表单, venue 为空时新建
 * 新建时出价货币为基础货币, 基础货币不存在时换成第一个货币, 金额按它的小数位换算;
 * 人机风格只保留默认配置中仍然存在的风格
 */
export function toVenueForm(venue: VenueRequest | null, {currencies, types, personas, baseCurrency}: VenueFormOptions): VenueForm {
  const exists = (w: BotWeight) => personas.some((p) => p.code === w.persona);
  const v = venue ?? {
    ...DEFAULT_REQUEST,
    bid_currency: currencies.some((c) => c.code === baseCurrency) ? baseCurrency : currencies[0]?.code ?? '',
    bots: {pve: DEFAULT_REQUEST.bots.pve.filter(exists), ranked: DEFAULT_REQUEST.bots.ranked.filter(exists)},
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
    requirements: toConditionDrafts(v.requirements ?? [], types),
    lotMin: String(v.lots.min),
    lotMax: String(v.lots.max),
    tiers: v.lots.tiers.map((t) => ({
      key: nanoid(),
      code: t.code,
      name: t.name,
      weight: String(t.weight),
      warehouse: toWarehouseDraft(t.warehouse),
    })),
    gapSeconds: String(v.lots.gap_seconds),
    coverage: ratioInput(v.lots.coverage, PERCENT_DECIMALS),
    rounds: String(v.auction.rounds),
    roundSeconds: String(v.auction.round_seconds),
    revealSeconds: String(v.auction.reveal_seconds),
    quickWin: resizeQuickWin(v.auction.quick_win.map((x) => ratioInput(x, MULTIPLE)), v.auction.rounds),
    maxExtraRounds: String(v.auction.max_extra_rounds),
    botsPve: toWeightDrafts(v.bots.pve ?? []),
    botsRanked: toWeightDrafts(v.bots.ranked ?? []),
    practiceOnly: v.pve.practice_only,
    dailyProfitCap: amountInput(v.pve.daily_profit_cap, decimals),
    valueFactor: ratioInput(v.pve.value_factor, PERCENT_DECIMALS),
    usesPve: String(v.consumables?.pve.max_per_match ?? 0),
    usesRanked: String(v.consumables?.ranked.max_per_match ?? 0),
    allowedItems: v.consumables?.allowed ?? [],
    hostEnabled: (v.host ?? EMPTY_HOST).enabled,
    hostPersonas: v.host?.personas ?? [],
    lieRate: ratioInput(v.host?.lie_rate ?? EMPTY_HOST.lie_rate, PERCENT_DECIMALS),
    highQuality: v.host?.high_quality ?? 0,
  };
}

/** 速胜倍率的个数跟随轮数, 多出的去掉, 不够的沿用最后一个, 一个都没有时用推荐值 */
export function resizeQuickWin(quickWin: string[], rounds: number): string[] {
  const size = Math.max(rounds - 1, 0);
  const resized = quickWin.slice(0, size);
  while (resized.length < size) resized.push(resized.at(-1) ?? recommendedQuickWin(size)[resized.length]);
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
  readNumber(text, {label, min, max, decimals: PERCENT_DECIMALS, unit: '%'});

/** 仓库表单转为请求体, prefix 加在提示前面, 如档次名称 */
function fromWarehouseDraft(w: WarehouseDraft, prefix = ''): WarehouseRule {
  check(w.slots.length === 0 || w.slots.length > MAX_SLOTS ? `${prefix}档位需为 1-${MAX_SLOTS} 个` : null);
  check(w.swings.length > MAX_SWINGS ? `${prefix}数量波动最多 ${MAX_SWINGS} 档` : null);
  return {
    grid: {
      width: readNumber(w.gridWidth, {label: `${prefix}仓库宽度`, min: 1, max: 16}),
      height: readNumber(w.gridHeight, {label: `${prefix}仓库高度`, min: 1, max: 16}),
    },
    slots: w.slots.map((s, i) => {
      const row = `${prefix}第 ${i + 1} 档`;
      check(!s.pool ? `${row}: 请选择掉落池` : null);
      const min = readNumber(s.min, {label: `${row}: 最少次数`, min: 0, max: 100});
      const max = readNumber(s.max, {label: `${row}: 最多次数`, min: 0, max: 100});
      check(max < min ? `${row}: 最多次数不能少于最少次数` : null);
      return {pool: s.pool, min, max, chance: percent(s.chance, `${row}: 出现概率`, 1, 10000)};
    }),
    swings: w.swings.map((s, i) => {
      const row = `${prefix}第 ${i + 1} 档波动`;
      return {
        factor: percent(s.factor, `${row}: 数量系数`, 2000, 20000),
        weight: readNumber(s.weight, {label: `${row}: 权重`, min: 1, max: 10000}),
      };
    }),
    max_rerolls: readNumber(w.maxRerolls, {label: `${prefix}重抽次数上限`, min: 0, max: 50}),
  };
}

/** 拍场转为请求体 */
function fromLots(form: VenueForm): VenueRequest['lots'] {
  const tiers = form.tiers;
  check(tiers.length === 0 || tiers.length > MAX_TIERS ? `档次需为 1-${MAX_TIERS} 种` : null);
  const min = readNumber(form.lotMin, {label: '最少拍品数', min: 1, max: MAX_LOTS});
  const max = readNumber(form.lotMax, {label: '最多拍品数', min: 1, max: MAX_LOTS});
  check(max < min ? '最多拍品数不能少于最少拍品数' : null);
  return {
    min,
    max,
    gap_seconds: readNumber(form.gapSeconds, {label: '拍品间隔', min: 0, max: MAX_GAP_SECONDS, unit: ' 秒'}),
    coverage: percent(form.coverage, '带入覆盖率', 0, 20000),
    tiers: tiers.map((t, i) => {
      const row = `第 ${i + 1} 种档次`;
      const code = t.code.trim();
      check(code.length === 0 || [...code].length > 32 ? `${row}: 代码需为 1-32 个字符` : null);
      check(tiers.findIndex((x) => x.code.trim() === code) < i ? `${row}: 代码 ${code} 重复` : null);
      const name = t.name.trim();
      check(name.length === 0 || [...name].length > 32 ? `${row}: 名称需为 1-32 个字符` : null);
      return {
        code,
        name,
        weight: readNumber(t.weight, {label: `${row}: 权重`, min: 1, max: 10000}),
        warehouse: fromWarehouseDraft(t.warehouse, `档次 ${name}: `),
      };
    }),
  };
}

/** 一个模式出场的人机风格转为请求体, 开放的模式至少配置一种, 风格不能重复 */
function fromWeightDrafts(drafts: WeightDraft[], mode: string, enabled: boolean): BotWeight[] {
  check(enabled && drafts.length === 0 ? `${mode}模式至少配置一种人机风格` : null);
  check(drafts.length > MAX_BOT_PERSONAS ? `${mode}模式最多 ${MAX_BOT_PERSONAS} 种人机风格` : null);
  return drafts.map((d, i) => {
    const row = `${mode}模式第 ${i + 1} 种风格`;
    check(!d.persona ? `${row}: 请选择风格` : null);
    check(drafts.findIndex((x) => x.persona === d.persona) < i ? `${row}: 风格重复` : null);
    return {persona: d.persona, weight: readNumber(d.weight, {label: `${row}: 权重`, min: 1, max: 100})};
  });
}

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
  check(
    form.entryCost.some((d) => d.currency === form.bidCurrency) ?
      '入场代价中不能有出价货币, 出价货币的入场费由对局验算自动算出' :
      null,
  );
  check(form.requirements.length > MAX_REQUIREMENTS ? `入场条件最多 ${MAX_REQUIREMENTS} 个` : null);
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
    lots: fromLots(form),
    auction: {
      rounds,
      round_seconds: readNumber(form.roundSeconds, {label: '每轮时长', min: 5, max: 120, unit: ' 秒'}),
      reveal_seconds: readNumber(form.revealSeconds, {label: '开箱时长', min: 3, max: 60, unit: ' 秒'}),
      quick_win: form.quickWin.map((x, i) =>
        readNumber(x, {label: `第 ${i + 1} 轮速胜倍率`, min: 10000, max: 100000, decimals: MULTIPLE, unit: ' 倍'}),
      ),
      max_extra_rounds: readNumber(form.maxExtraRounds, {label: '加轮上限', min: 0, max: 5}),
    },
    bots: {
      pve: fromWeightDrafts(form.botsPve, '人机', form.pveEnabled),
      ranked: fromWeightDrafts(form.botsRanked, '排位', form.rankedEnabled),
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
    host: {
      enabled: form.hostEnabled,
      personas: form.hostPersonas,
      lie_rate: percent(form.lieRate, '撒谎率', 0, MAX_LIE_RATE),
      high_quality: form.highQuality,
    },
  };
}

/** 表单各页签共用的Props */
export interface FormTabProps {
  form: VenueForm;
  onChange: (patch: Partial<VenueForm>) => void;
  disabled: boolean;
}
