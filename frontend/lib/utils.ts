const dateTimeFormat = new Intl.DateTimeFormat('zh-CN', {
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit',
  hour12: false,
});

/** 格式化后端返回的时间, 空值显示 "-" */
export function formatDateTime(value: string | null | undefined): string {
  return value ? dateTimeFormat.format(new Date(value)) : '-';
}

/**
 * 格式化金额, 后端按最小单位的整数返回, decimals 为货币的展示小数位
 * signed: 正数带 + 号, 用于流水; compact: 大数缩写为 万/亿, 用于空间有限的地方
 */
export function formatAmount(amount: number, decimals = 0, {signed = false, compact = false} = {}): string {
  return new Intl.NumberFormat('zh-CN', {
    minimumFractionDigits: compact ? 0 : decimals,
    maximumFractionDigits: compact ? 1 : decimals,
    notation: compact ? 'compact' : 'standard',
    signDisplay: signed ? 'exceptZero' : 'auto',
  }).format(amount / 10 ** decimals);
}

/** 把输入的金额转成最小单位的整数, 格式不对或小数位超出时返回 null; 用于表单 */
export function parseAmount(input: string, decimals = 0): number | null {
  const value = input.trim();
  if (!new RegExp(decimals > 0 ? `^-?\d+(\.\d{1,${decimals}})?$` : '^-?\d+$').test(value)) return null;
  return Math.round(Number(value) * 10 ** decimals);
}

/** 最小单位的整数转成表单里的金额, 不带千分位 */
export function amountInput(amount: number, decimals = 0): string {
  return (amount / 10 ** decimals).toFixed(decimals);
}
