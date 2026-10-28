import {cn} from 'cn';
import {QUALITY_EFFECT} from '@/lib/quality';

/**
 * 按填写的颜色和特效预览品质名称; 不读站点的品质样式, 保存前也能看到效果
 */
export function QualityPreview({name, color, effect, className}: {
  name: string;
  color: string;
  effect: string;
  className?: string;
}) {
  const rainbow = effect === QUALITY_EFFECT.rainbow;
  return (
    <span className={cn('flex min-w-0 items-center gap-2', className)}>
      <span
        className={cn('size-3 shrink-0 rounded-full', rainbow && 'quality-rainbow')}
        style={rainbow ? undefined : {backgroundColor: color}}
      />
      <span
        className={cn('truncate font-medium', rainbow && 'quality-rainbow bg-clip-text text-transparent')}
        style={rainbow ? undefined : {color}}
      >
        {name}
      </span>
    </span>
  );
}
