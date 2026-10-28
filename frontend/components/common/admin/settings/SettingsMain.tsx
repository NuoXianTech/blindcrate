'use client';

import {useEffect, useState} from 'react';
import {toast} from 'sonner';
import {Skeleton} from '@/components/ui/skeleton';
import {Tabs, TabsContent, TabsList, TabsTrigger} from '@/components/ui/tabs';
import {useAuth} from '@/components/common/auth';
import services from '@/lib/services';
import type {ListSettingsData, SettingGroup} from '@/lib/services/admin';
import {toErrorMessage} from '@/lib/services/core';
import {SettingGroupCard} from './SettingGroupCard';

/**
 * 系统设置: 按分类分页签, 每个设置组一张卡片, 表单由后端返回的字段描述生成
 */
export function SettingsMain() {
  const {user: me} = useAuth();
  const isAdmin = !!me?.is_admin;
  const [data, setData] = useState<ListSettingsData | null>(null);

  useEffect(() => {
    // 确认是管理员后再拉取, 否则 AdminGuard 会跳走
    if (!isAdmin) return;
    services.admin.listSettings()
        .then(setData)
        .catch((err) => toast.error(toErrorMessage(err)));
  }, [isAdmin]);

  const replaceGroup = (updated: SettingGroup) => {
    setData((prev) => prev && {...prev, groups: prev.groups.map((g) => (g.key === updated.key ? updated : g))});
  };

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-col gap-1">
        <h1 className="text-2xl font-bold">系统设置</h1>
        <p className="text-sm text-muted-foreground">保存后立即生效, 每次保存都会生成一个版本, 可以随时回滚</p>
      </div>
      {data ? (
        <Tabs defaultValue={data.categories[0]?.key}>
          <TabsList>
            {data.categories.map((c) => (
              <TabsTrigger key={c.key} value={c.key}>
                {c.label}
              </TabsTrigger>
            ))}
          </TabsList>
          {data.categories.map((c) => (
            <TabsContent key={c.key} value={c.key} className="flex flex-col gap-4">
              {data.groups
                  .filter((g) => g.category === c.key)
                  .map((g) => (
                    <SettingGroupCard key={g.key} group={g} onSaved={replaceGroup} />
                  ))}
            </TabsContent>
          ))}
        </Tabs>
      ) : (
        <>
          <Skeleton className="h-9 w-40 rounded-lg" />
          <Skeleton className="h-64 rounded-xl" />
        </>
      )}
    </div>
  );
}
