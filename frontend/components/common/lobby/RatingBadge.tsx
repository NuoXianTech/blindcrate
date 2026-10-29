'use client';

import {useEffect, useState} from 'react';
import {Trophy} from 'lucide-react';
import {useAuth} from '@/components/common/auth';
import services from '@/lib/services';
import type {RatingView} from '@/lib/services/ranked';

/**
 * 大厅标题旁自己在当前赛季的段位分, 悬停显示最高分和本赛季的排位局数
 */
export function RatingBadge() {
  const {user} = useAuth();
  const [data, setData] = useState<RatingView | null>(null);

  useEffect(() => {
    if (!user) return;
    services.ranked.getRating()
        .then(setData)
        .catch(() => {
          // 只是提示信息, 失败时不显示
        });
  }, [user]);

  if (!data) return null;

  const title = data.matches ?
    `本赛季最高 ${data.peak} 分, 已打 ${data.matches} 局排位` :
    '本赛季还没打过排位, 当前为初始分';
  return (
    <span
      className="flex items-center gap-1.5 rounded-full border bg-card px-3 py-1 text-xs text-muted-foreground"
      title={title}
    >
      <Trophy className="size-3.5" />
      段位分
      <span className="font-mono font-medium text-foreground tabular-nums">{data.rating}</span>
    </span>
  );
}
