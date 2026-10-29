import {cn} from 'cn';

/**
 * 段位名称: 段位颜色的圆点加名称, 没有段位时显示 "无段位"
 */
export function TierName({tier, className}: {
  tier: { name: string; color: string } | null;
  className?: string;
}) {
  return (
    <span className={cn('flex min-w-0 items-center gap-1.5', className)}>
      <span
        className={cn('size-2.5 shrink-0 rounded-full', !tier && 'bg-muted-foreground/40')}
        style={tier ? {backgroundColor: tier.color} : undefined}
      />
      <span className="truncate font-medium" style={tier ? {color: tier.color} : undefined}>
        {tier?.name ?? '无段位'}
      </span>
    </span>
  );
}
