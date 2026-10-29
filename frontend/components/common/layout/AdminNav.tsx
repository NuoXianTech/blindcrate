'use client';

import Link from 'next/link';
import {usePathname} from 'next/navigation';
import {cn} from 'cn';
import {ADMIN_NAV, isActive} from '@/lib/nav';

/** 管理区顶部的页面切换, 按分组排列, 窄屏时横向滚动 */
export function AdminNav() {
  const pathname = usePathname();
  return (
    <nav aria-label="管理页面" className="-mx-1 overflow-x-auto px-1">
      <div className="flex w-max gap-4 rounded-xl border bg-card p-2 lg:w-auto lg:flex-wrap">
        {ADMIN_NAV.map((group) => (
          <div key={group.label} className="flex items-center gap-1">
            <span className="px-1.5 text-xs text-muted-foreground">{group.label}</span>
            {group.entries.map((entry) => {
              const active = isActive(entry, pathname);
              return (
                <Link
                  key={entry.href}
                  href={entry.href}
                  aria-current={active ? 'page' : undefined}
                  className={cn(
                      'inline-flex h-8 items-center gap-1.5 rounded-md px-2.5 text-sm whitespace-nowrap transition-colors',
                    active ? 'bg-primary text-primary-foreground' : 'hover:bg-accent hover:text-accent-foreground',
                  )}
                >
                  <entry.icon className="size-4" />
                  {entry.title}
                </Link>
              );
            })}
          </div>
        ))}
      </div>
    </nav>
  );
}
