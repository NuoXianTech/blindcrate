'use client';

import {Drama, IdCard} from 'lucide-react';
import {Tabs, TabsContent, TabsList, TabsTrigger} from '@/components/ui/tabs';
import {useAuth} from '@/components/common/auth';
import {NamesPanel} from './NamesPanel';
import {PersonasPanel} from './PersonasPanel';

/**
 * 人机管理: 风格和昵称两个页签; 地图在人机页签中按模式选择出场的风格和权重
 */
export function BotsMain() {
  const {user: me} = useAuth();
  const isAdmin = !!me?.is_admin;

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-col gap-0.5">
        <h1 className="text-2xl font-bold">人机管理</h1>
        <p className="text-sm text-muted-foreground">
          人机只按公开信息出价, 风格决定估值方式和出价节奏; 开局时为每个人机选一种风格和一个昵称
        </p>
      </div>
      <Tabs defaultValue="personas">
        <TabsList>
          <TabsTrigger value="personas">
            <Drama />
            风格
          </TabsTrigger>
          <TabsTrigger value="names">
            <IdCard />
            昵称
          </TabsTrigger>
        </TabsList>
        {/* 列表确认是管理员后再拉取, 否则 AdminGuard 会跳走 */}
        <TabsContent value="personas" className="mt-2">
          {isAdmin && <PersonasPanel />}
        </TabsContent>
        <TabsContent value="names" className="mt-2">
          {isAdmin && <NamesPanel />}
        </TabsContent>
      </Tabs>
    </div>
  );
}
