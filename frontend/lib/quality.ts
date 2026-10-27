/**
 * 物品品质, 与后端 internal/apps/item/constants.go 的品质常量对应, 数值越大越稀有
 * 类名写全, Tailwind 才能扫描到; 炫彩是流动渐变, 见 app/globals.css 的 quality-rainbow
 */
export const QUALITIES = [
  {value: 1, label: '灰色', text: 'text-quality-gray', bar: 'bg-quality-gray'},
  {value: 2, label: '绿色', text: 'text-quality-green', bar: 'bg-quality-green'},
  {value: 3, label: '蓝色', text: 'text-quality-blue', bar: 'bg-quality-blue'},
  {value: 4, label: '紫色', text: 'text-quality-purple', bar: 'bg-quality-purple'},
  {value: 5, label: '金色', text: 'text-quality-gold', bar: 'bg-quality-gold'},
  {value: 6, label: '红色', text: 'text-quality-red', bar: 'bg-quality-red'},
  {value: 7, label: '炫彩', text: 'quality-rainbow bg-clip-text text-transparent', bar: 'quality-rainbow'},
] as const;

export type Quality = (typeof QUALITIES)[number];

/** 按数值取品质, 超出范围按灰色处理 */
export function qualityOf(value: number): Quality {
  return QUALITIES.find((q) => q.value === value) ?? QUALITIES[0];
}
