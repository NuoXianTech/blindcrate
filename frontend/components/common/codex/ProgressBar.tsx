import {cn} from 'cn';

/**
 * 收集进度条, 集齐后变为绿色
 */
export function ProgressBar({value, max, label, className}: {
  value: number;
  max: number;
  /** 读屏时的名称, 如 "图鉴进度" */
  label: string;
  className?: string;
}) {
  const ratio = max > 0 ? Math.min(1, value / max) : 0;
  return (
    <div
      className={cn('h-2 overflow-hidden rounded-full bg-muted', className)}
      role="progressbar"
      aria-label={label}
      aria-valuemin={0}
      aria-valuemax={max}
      aria-valuenow={value}
    >
      <div
        className={cn('h-full rounded-full transition-[width]', ratio >= 1 ? 'bg-quality-green' : 'bg-primary')}
        style={{width: `${ratio * 100}%`}}
      />
    </div>
  );
}
