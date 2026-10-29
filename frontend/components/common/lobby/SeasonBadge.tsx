'use client';

import {useEffect, useState} from 'react';
import {CalendarRange} from 'lucide-react';
import {useAuth} from '@/components/common/auth';
import services from '@/lib/services';
import type {CurrentSeasonData} from '@/lib/services/season';

const HOUR = 60 * 60 * 1000;
const DAY = 24 * HOUR;

const startFormat = new Intl.DateTimeFormat('zh-CN', {month: 'long', day: 'numeric', hour: '2-digit', minute: '2-digit', hour12: false});

/** 赛季剩余时间, 如 "剩余 12 天" */
function remaining(endsAt: string) {
  const ms = new Date(endsAt).getTime() - Date.now();
  if (ms >= DAY) return `剩余 ${Math.floor(ms / DAY)} 天`;
  if (ms >= HOUR) return `剩余 ${Math.floor(ms / HOUR)} 小时`;
  return '即将结束';
}

/**
 * 大厅标题旁的赛季信息: 进行中的赛季和剩余时间, 没有时显示下一个赛季的开始时间, 都没有时不显示
 */
export function SeasonBadge() {
  const {user} = useAuth();
  const [data, setData] = useState<CurrentSeasonData | null>(null);

  useEffect(() => {
    if (!user) return;
    services.season.getCurrent()
        .then(setData)
        .catch(() => {
          // 只是提示信息, 失败时不显示
        });
  }, [user]);

  const text = data?.current ?
    `${data.current.name} · ${remaining(data.current.ends_at)}` :
    data?.next ? `${data.next.name} · ${startFormat.format(new Date(data.next.starts_at))} 开始` : null;
  if (!text) return null;

  return (
    <span className="flex items-center gap-1.5 rounded-full border bg-card px-3 py-1 text-xs text-muted-foreground">
      <CalendarRange className="size-3.5" />
      {text}
    </span>
  );
}
