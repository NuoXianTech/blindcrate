'use client';

import {useEffect, useState} from 'react';
import Link from 'next/link';
import {ClipboardList} from 'lucide-react';
import {cn} from 'cn';
import {useAuth} from '@/components/common/auth';
import {ROUTES} from '@/lib/access';
import services from '@/lib/services';

/**
 * 大厅标题旁的任务入口: 有奖励待领取时显示数量并高亮, 点击进入任务页
 * version 变化时重新拉取, 如签到后可能完成了签到任务
 */
export function MissionsBadge({version = 0}: { version?: number }) {
  const {user} = useAuth();
  const [claimable, setClaimable] = useState(0);

  useEffect(() => {
    if (!user) return;
    services.mission.list()
        .then((data) => setClaimable(data.claimable))
        .catch(() => {
          // 只是提示信息, 失败时只显示入口
        });
  }, [user, version]);

  const tone = claimable ? 'border-primary text-primary' : 'text-muted-foreground';
  return (
    <Link
      href={ROUTES.missions}
      className={cn('flex items-center gap-1.5 rounded-full border bg-card px-3 py-1 text-xs transition-colors hover:text-foreground', tone)}
    >
      <ClipboardList className="size-3.5" />
      {claimable ? `${claimable} 个任务奖励待领取` : '任务'}
    </Link>
  );
}
