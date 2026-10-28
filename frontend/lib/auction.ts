/** 玩法代码为 <模式>:<地图代码>, 模式与后端 internal/apps/venue/constants.go 对应 */
export const MODES = {
  pve: {label: '人机', description: '1 名玩家对 3 个人机, 立即开局'},
  ranked: {label: '排位', description: '至少 2 名玩家, 等待 15 秒后由人机补齐'},
} as const;

export type ModeKind = keyof typeof MODES;

/** 拼出玩法代码 */
export function modeCode(kind: ModeKind, venueCode: string) {
  return `${kind}:${venueCode}`;
}

/** 拆开玩法代码, 不是暗拍的玩法返回 null */
export function parseMode(code: string): { kind: ModeKind; venue: string } | null {
  const [kind, venue] = code.split(':', 2);
  if (!venue || !(kind in MODES)) return null;
  return {kind: kind as ModeKind, venue};
}

/** 玩法的展示名称, 如 "人机" */
export function modeLabel(code: string) {
  const mode = parseMode(code);
  return mode ? MODES[mode.kind].label : code;
}

/** 每轮反馈的提示, 与后端 internal/apps/auction/constants.go 的 Hint 常量对应 */
export const HINTS: Record<string, { label: string; className: string }> = {
  no_bid: {label: '还没出价', className: 'text-muted-foreground'},
  far_ahead: {label: '遥遥领先', className: 'text-quality-green'},
  ahead: {label: '小幅领先', className: 'text-quality-green'},
  tied: {label: '并列最高', className: 'text-quality-gold'},
  close: {label: '稍稍落后', className: 'text-quality-gold'},
  far_behind: {label: '落后较多', className: 'text-quality-red'},
};

/** 结算结果的名称 */
export const OUTCOMES: Record<string, string> = {
  sold: '成交',
  unsold: '流拍',
  cancelled: '已取消',
};

/** 基点, 倍率和比例都按它存储 */
export const BASIS_POINTS = 10000;

/** 基点格式化为倍数, 如 16000 -> "1.6" */
export function formatRatio(bp: number) {
  return (bp / BASIS_POINTS).toFixed(1);
}
