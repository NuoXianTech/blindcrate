'use client';

import {useEffect, useMemo, useState} from 'react';
import {Megaphone, Search, Send} from 'lucide-react';
import {toast} from 'sonner';
import {Tabs, TabsContent, TabsList, TabsTrigger} from '@/components/ui/tabs';
import {scopeOptions} from '@/components/common/admin/editors';
import {useAuth} from '@/components/common/auth';
import {useAdminCatalog} from '@/hooks/use-admin-catalog';
import services from '@/lib/services';
import type {AdminVenue, ConditionType} from '@/lib/services/admin';
import {toErrorMessage, type CodeLabel} from '@/lib/services/core';
import type {BroadcastOptions} from './BroadcastDialog';
import {BroadcastsPanel} from './BroadcastsPanel';
import {MailQueryPanel} from './MailQueryPanel';
import {SendPanel} from './SendPanel';
import {VENUE_LIMIT} from './constants';

/**
 * 游戏邮件: 发给指定用户、全服邮件和邮件查询三个页签
 */
export function MailsMain() {
  const {user: me} = useAuth();
  const isAdmin = !!me?.is_admin;
  const catalog = useAdminCatalog();
  const [types, setTypes] = useState<ConditionType[] | null>(null);
  const [venues, setVenues] = useState<AdminVenue[] | null>(null);
  const [sources, setSources] = useState<CodeLabel[]>([]);

  // 条件类型、地图和邮件来源挂载时拉取一次
  useEffect(() => {
    if (!isAdmin) return;
    Promise.all([
      services.admin.listConditionTypes(),
      services.admin.listVenues({current: 1, size: VENUE_LIMIT}),
      services.mailboxAdmin.listSources(),
    ])
        .then(([typeData, venueData, sourceData]) => {
          setTypes(typeData.types);
          setVenues(venueData.venues);
          setSources(sourceData.sources);
        })
        .catch((err) => toast.error(toErrorMessage(err)));
  }, [isAdmin]);

  const options = useMemo<BroadcastOptions>(() => ({
    items: catalog.items,
    currencies: catalog.currencies,
    types: types ?? [],
    scopes: scopeOptions(venues ?? []),
    loaded: catalog.loaded && types !== null && venues !== null,
  }), [catalog, types, venues]);

  return (
    <div className="flex flex-col gap-4">
      <h1 className="text-2xl font-bold">游戏邮件</h1>
      <Tabs defaultValue="send">
        <TabsList>
          <TabsTrigger value="send">
            <Send />
            发送
          </TabsTrigger>
          <TabsTrigger value="broadcasts">
            <Megaphone />
            全服邮件
          </TabsTrigger>
          <TabsTrigger value="query">
            <Search />
            邮件查询
          </TabsTrigger>
        </TabsList>
        <TabsContent value="send" className="mt-2">
          <SendPanel items={catalog.items} currencies={catalog.currencies} loaded={catalog.loaded} />
        </TabsContent>
        {/* 列表确认是管理员后再拉取, 否则 AdminGuard 会跳走 */}
        <TabsContent value="broadcasts" className="mt-2">
          {isAdmin && <BroadcastsPanel options={options} />}
        </TabsContent>
        <TabsContent value="query" className="mt-2">
          {isAdmin && <MailQueryPanel items={catalog.items} currencies={catalog.currencies} sources={sources} />}
        </TabsContent>
      </Tabs>
    </div>
  );
}
