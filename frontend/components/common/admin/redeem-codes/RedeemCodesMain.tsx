'use client';

import {useEffect, useMemo, useState} from 'react';
import {ScrollText, Ticket} from 'lucide-react';
import {toast} from 'sonner';
import {Tabs, TabsContent, TabsList, TabsTrigger} from '@/components/ui/tabs';
import {useAuth} from '@/components/common/auth';
import {useAdminCatalog} from '@/hooks/use-admin-catalog';
import services from '@/lib/services';
import type {ConditionType} from '@/lib/services/condition';
import {toErrorMessage} from '@/lib/services/core';
import type {BatchOptions} from './BatchDialog';
import {BatchesPanel} from './BatchesPanel';
import {RecordsPanel} from './RecordsPanel';

/**
 * 兑换码: 兑换码批次和兑换记录两个页签
 */
export function RedeemCodesMain() {
  const {user: me} = useAuth();
  const isAdmin = !!me?.is_admin;
  const catalog = useAdminCatalog();
  const [types, setTypes] = useState<ConditionType[] | null>(null);

  // 条件类型只在编辑使用条件时用到, 挂载时拉取一次
  useEffect(() => {
    if (!isAdmin) return;
    services.conditionAdmin.listConditionTypes()
        .then((typeData) => setTypes(typeData.types))
        .catch((err) => toast.error(toErrorMessage(err)));
  }, [isAdmin]);

  const options = useMemo<BatchOptions>(() => ({
    items: catalog.items,
    currencies: catalog.currencies,
    types: types ?? [],
    loaded: catalog.loaded && types !== null,
  }), [catalog, types]);

  return (
    <div className="flex flex-col gap-4">
      <h1 className="text-2xl font-bold">兑换码</h1>
      <Tabs defaultValue="batches">
        <TabsList>
          <TabsTrigger value="batches">
            <Ticket />
            兑换码批次
          </TabsTrigger>
          <TabsTrigger value="records">
            <ScrollText />
            兑换记录
          </TabsTrigger>
        </TabsList>
        {/* 列表确认是管理员后再拉取, 否则 AdminGuard 会跳走 */}
        <TabsContent value="batches" className="mt-2">
          {isAdmin && <BatchesPanel options={options} />}
        </TabsContent>
        <TabsContent value="records" className="mt-2">
          {isAdmin && <RecordsPanel items={catalog.items} currencies={catalog.currencies} />}
        </TabsContent>
      </Tabs>
    </div>
  );
}
