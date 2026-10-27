'use client';

import {cn} from 'cn';
import {useSite} from './SiteProvider';

/**
 * 备案号和联系邮箱, 都没有设置时不显示
 */
export function SiteFooter({className}: { className?: string }) {
  const {site} = useSite();
  if (!site.icp && !site.contact_email) return null;
  return (
    <footer className={cn('flex flex-wrap items-center justify-center gap-x-4 gap-y-1 text-xs text-muted-foreground', className)}>
      {site.icp && (
        <a href="https://beian.miit.gov.cn/" target="_blank" rel="noreferrer" className="hover:text-foreground">
          {site.icp}
        </a>
      )}
      {site.contact_email && (
        <a href={`mailto:${site.contact_email}`} className="hover:text-foreground">
          联系邮箱 {site.contact_email}
        </a>
      )}
    </footer>
  );
}
