'use client';

import {useEffect, useState} from 'react';
import {toast} from 'sonner';
import {Skeleton} from '@/components/ui/skeleton';
import {useAuth} from '@/components/common/auth';
import services from '@/lib/services';
import type {SettingGroup} from '@/lib/services/admin';
import {toErrorMessage} from '@/lib/services/core';
import {SettingGroupCard} from './SettingGroupCard';

/**
 * 系统设置: 每个设置组一张卡片, 表单由后端返回的字段描述生成
 */
export function SettingsMain() {
  const {user: me} = useAuth();
  const isAdmin = !!me?.is_admin;
  const [groups, setGroups] = useState<SettingGroup[] | null>(null);

  useEffect(() => {
    // 确认是管理员后再拉取, 否则 AdminGuard 会跳走
    if (!isAdmin) return;
    services.admin.listSettings()
        .then((data) => setGroups(data.groups))
        .catch((err) => toast.error(toErrorMessage(err)));
  }, [isAdmin]);

  const replaceGroup = (updated: SettingGroup) => {
    setGroups((prev) => prev?.map((g) => (g.key === updated.key ? updated : g)) ?? null);
  };

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-col gap-1">
        <h1 className="text-2xl font-bold">系统设置</h1>
        <p className="text-sm text-muted-foreground">保存后立即生效</p>
      </div>
      {groups ? (
        groups.map((g) => <SettingGroupCard key={g.key} group={g} onSaved={replaceGroup} />)
      ) : (
        <>
          <Skeleton className="h-64 rounded-xl" />
          <Skeleton className="h-64 rounded-xl" />
        </>
      )}
    </div>
  );
}
