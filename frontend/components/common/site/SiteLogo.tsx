'use client';

import {Package} from 'lucide-react';
import {cn} from 'cn';
import {useSite} from './SiteProvider';

/**
 * 站点 Logo, 没有设置时显示默认图标
 */
export function SiteLogo({className}: { className?: string }) {
  const {site} = useSite();
  if (site.logo_url) {
    // 图片地址由管理员填写, 域名不固定, 不走 next/image
    // eslint-disable-next-line @next/next/no-img-element
    return <img src={site.logo_url} alt={site.name} className={cn('size-full object-contain', className)} />;
  }
  return <Package className={cn('size-1/2', className)} aria-hidden />;
}
