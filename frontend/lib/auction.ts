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

/** 主持人的性格, 与后端 internal/apps/venue/constants.go 的 Host 常量对应 */
export const HOST_PERSONAS: Record<string, { label: string; description: string }> = {
  steady: {label: '稳重', description: '很少说错, 说错时可能偏高也可能偏低'},
  showman: {label: '煽动', description: '说错时只会往高了吹, 场子越热越敢吹'},
  sly: {label: '狡黠', description: '说错时只会往低了压, 场子越冷越敢压'},
};

/** 热度的档位, 煽动的主持人在火热时、狡黠的在冷清时更敢说错 */
export function hypeLevel(hype: number) {
  if (hype >= 70) return {label: '火热', className: 'text-quality-red'};
  if (hype <= 30) return {label: '冷清', className: 'text-quality-blue'};
  return {label: '平稳', className: 'text-muted-foreground'};
}

/** 第 round 轮(共 rounds 轮)向主持人买到的情报, 与后端 auction.intelFor 对应 */
export function intelTier(round: number, rounds: number) {
  if (round * 3 <= rounds) return '一件还不知道品质的藏品的品质';
  if (round * 3 <= rounds * 2) return '一件藏品的名称和品质';
  return '仓库价值的大致区间';
}

/** 暗拍阶段的名称 */
export const PHASES: Record<string, string> = {
  bidding: '出价',
  reveal: '开箱',
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

/** 基点格式化为百分比, 如 7500 -> "75%" */
export function formatRate(bp: number) {
  return `${(bp / 100).toLocaleString('zh-CN', {maximumFractionDigits: 2})}%`;
}

/** 一件藏品的位置, 一局中不重复, 用来对应扫出的轮廓、情报和主持人的指认 */
export function cellKey(c: { x: number; y: number }) {
  return `${c.x},${c.y}`;
}

/** 开箱扫描的进度 */
export interface ScanProgress {
  /** 已扫描完的件数, 按扫描顺序前这么多件已公开 */
  done: number;
  /** 正在扫描的这件已扫过的比例, 0 到 1; 全部扫完时为 1 */
  fraction: number;
}

/**
 * 按开始扫描后过去的毫秒算出进度, durations 为按扫描顺序的每件扫描时长
 * 没有扫描时长的旧对局视为一开始就全部扫完
 */
export function scanProgress(durations: number[], elapsed: number): ScanProgress {
  let start = 0;
  for (let i = 0; i < durations.length; i++) {
    const end = start + durations[i];
    if (elapsed < end) return {done: i, fraction: Math.max(0, elapsed - start) / durations[i]};
    start = end;
  }
  return {done: durations.length, fraction: 1};
}
