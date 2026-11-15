'use client';

import {useCallback, useEffect, useState} from 'react';
import {Search} from 'lucide-react';
import {toast} from 'sonner';
import {Button} from '@/components/ui/button';
import {Input} from '@/components/ui/input';
import {Table, TableBody, TableCell, TableHead, TableHeader, TableRow} from '@/components/ui/table';
import {bundleText} from '@/components/common/admin/editors';
import {Pagination, TableEmpty, TableSkeleton} from '@/components/common/layout';
import services from '@/lib/services';
import {toErrorMessage} from '@/lib/services/core';
import type {AdminItem} from '@/lib/services/item';
import type {ListRedeemRecordsData} from '@/lib/services/redeem';
import type {Currency} from '@/lib/services/wallet';
import {formatDateTime, parseID} from '@/lib/utils';
import {MAX_CODE_INPUT, PAGE_SIZE} from './constants';

/** 输入框中的 ID: 留空为 undefined, 不是正整数时为 null */
function inputID(text: string): number | undefined | null {
  if (!text) return undefined;
  const id = parseID(text);
  return id && Number.isSafeInteger(id) ? id : null;
}

/**
 * 兑换记录: 按批次 ID、用户 ID 和兑换码筛选, 包括已删除的批次的记录
 */
export function RecordsPanel({items, currencies}: {
  items: AdminItem[];
  currencies: Currency[];
}) {
  const [batchInput, setBatchInput] = useState('');
  const [userInput, setUserInput] = useState('');
  const [codeInput, setCodeInput] = useState('');
  const [filters, setFilters] = useState<{ batchID?: number; userID?: number; code: string }>({code: ''});
  const [current, setCurrent] = useState(1);
  const [data, setData] = useState<ListRedeemRecordsData | null>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      setData(
          await services.redeemAdmin.listRecords({
            current,
            size: PAGE_SIZE,
            batch_id: filters.batchID,
            user_id: filters.userID,
            code: filters.code || undefined,
          }),
      );
    } catch (err) {
      toast.error(toErrorMessage(err));
    } finally {
      setLoading(false);
    }
  }, [current, filters]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 筛选条件变化时重新拉取
    void load();
  }, [load]);

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    const batchID = inputID(batchInput.trim());
    const userID = inputID(userInput.trim());
    if (batchID === null || userID === null) {
      toast.error('批次 ID 和用户 ID 需为正整数');
      return;
    }
    setCurrent(1);
    setFilters({batchID, userID, code: codeInput.trim()});
  };

  return (
    <div className="flex flex-col gap-4">
      <form onSubmit={handleSearch} className="flex flex-wrap gap-2">
        <Input className="w-32 font-mono" inputMode="numeric" placeholder="批次 ID" value={batchInput} onChange={(e) => setBatchInput(e.target.value)} />
        <Input className="w-32 font-mono" inputMode="numeric" placeholder="用户 ID" value={userInput} onChange={(e) => setUserInput(e.target.value)} />
        <Input
          className="w-56 font-mono"
          placeholder="完整的兑换码"
          maxLength={MAX_CODE_INPUT}
          value={codeInput}
          onChange={(e) => setCodeInput(e.target.value)}
        />
        <Button type="submit" variant="outline" size="icon" aria-label="搜索">
          <Search />
        </Button>
      </form>

      <div className="rounded-xl border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>用户</TableHead>
              <TableHead>批次</TableHead>
              <TableHead>兑换码</TableHead>
              <TableHead>得到的奖励</TableHead>
              <TableHead>兑换时间</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading && !data ? (
              <TableSkeleton cols={5} />
            ) : data?.records.length ? (
              data.records.map((r) => {
                const reward = bundleText(r.reward, items, currencies);
                return (
                  <TableRow key={r.id}>
                    <TableCell>
                      <div className="flex flex-col">
                        <span>{r.username || '-'}</span>
                        <span className="font-mono text-xs text-muted-foreground">#{r.user_id}</span>
                      </div>
                    </TableCell>
                    <TableCell className="max-w-48">
                      <div className="flex min-w-0 flex-col">
                        <span className="truncate font-medium" title={r.batch_name || undefined}>
                          {r.batch_name || '-'}
                        </span>
                        <span className="font-mono text-xs text-muted-foreground">#{r.batch_id}</span>
                      </div>
                    </TableCell>
                    <TableCell className="max-w-48 truncate font-mono" title={r.code || undefined}>
                      {r.code || '-'}
                    </TableCell>
                    <TableCell className="max-w-56 truncate" title={reward.join(', ') || undefined}>
                      {reward.join(', ')}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground tabular-nums">{formatDateTime(r.created_at)}</TableCell>
                  </TableRow>
                );
              })
            ) : (
              <TableEmpty colSpan={5}>没有找到兑换记录</TableEmpty>
            )}
          </TableBody>
        </Table>
      </div>

      <Pagination current={current} size={PAGE_SIZE} total={data?.total ?? 0} onChange={setCurrent} />
    </div>
  );
}
