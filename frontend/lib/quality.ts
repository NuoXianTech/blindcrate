/**
 * 物品品质, 对应后端 GET /api/v1/qualities, 由管理员在品质管理中修改; 等级越大越稀有
 */
export interface Quality {
  level: number;
  code: string;
  name: string;
  /** #rrggbb */
  color: string;
  /** 特效, 空表示纯色, 见 QUALITY_EFFECT */
  effect: string;
  /** 开箱时扫描这一品质藏品的时长, 毫秒 */
  scan_ms: number;
  sort_order: number;
}

/** 品质特效, 与后端 internal/apps/item/constants.go 一致 */
export const QUALITY_EFFECT = {
  /** 炫彩: 文字和色条都用流动渐变, 见 app/globals.css 的 quality-rainbow */
  rainbow: 'rainbow',
} as const;

/** 页面上展示的品质: 名称和文字、色条的类名 */
export interface QualityStyle {
  level: number;
  name: string;
  text: string;
  bar: string;
}

/** 等级不存在(品质已被删除)时的展示 */
export const UNKNOWN_QUALITY: QualityStyle = {
  level: 0,
  name: '未知',
  text: 'text-muted-foreground',
  bar: 'bg-muted-foreground',
};

const COLOR_PATTERN = /^#[0-9a-f]{6}$/i;

/** 品质的展示样式; 纯色品质的类名由 qualityStyleSheet 生成 */
export function qualityStyle(q: Quality): QualityStyle {
  if (q.effect === QUALITY_EFFECT.rainbow) {
    return {level: q.level, name: q.name, text: 'quality-rainbow bg-clip-text text-transparent', bar: 'quality-rainbow'};
  }
  return {level: q.level, name: q.name, text: `quality-text-${q.level}`, bar: `quality-bar-${q.level}`};
}

/** 纯色品质的样式表, 颜色来自接口, 不是 #rrggbb 的跳过 */
export function qualityStyleSheet(qualities: Quality[]): string {
  return qualities
      .filter((q) => Number.isInteger(q.level) && COLOR_PATTERN.test(q.color))
      .map((q) => `.quality-text-${q.level}{color:${q.color}}.quality-bar-${q.level}{background-color:${q.color}}`)
      .join('');
}
