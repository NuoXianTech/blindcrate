import {CircleQuestionMark, Package} from 'lucide-react';
import {cn} from 'cn';

/** 物品图片, 没有图片时显示通用图标, 占位物品(已被删除)显示问号 */
export function ItemImage({
  item,
  className,
}: {
  item: { name: string; image_url: string; is_placeholder?: boolean };
  className?: string;
}) {
  if (item.image_url) {
    // 图片地址由管理员填写, 域名不固定, 不走 next/image
    // eslint-disable-next-line @next/next/no-img-element
    return <img src={item.image_url} alt={item.name} className={cn('size-full object-contain', className)} />;
  }
  const Icon = item.is_placeholder ? CircleQuestionMark : Package;
  return <Icon className={cn('size-1/2 text-muted-foreground', className)} aria-hidden />;
}
