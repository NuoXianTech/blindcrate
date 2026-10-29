'use client';

import {useCallback, useEffect, useState} from 'react';
import {Pencil, Plus, Trash2} from 'lucide-react';
import {toast} from 'sonner';
import {Button} from '@/components/ui/button';
import {Table, TableBody, TableCell, TableHead, TableHeader, TableRow} from '@/components/ui/table';
import {useAuth} from '@/components/common/auth';
import {TableEmpty, TableSkeleton} from '@/components/common/layout';
import {TierName} from '@/components/common/ranked';
import services from '@/lib/services';
import {toErrorMessage} from '@/lib/services/core';
import type {Tier} from '@/lib/services/ranked';
import {DeleteTierDialog} from './DeleteTierDialog';
import {TierDialog} from './TierDialog';

/**
 * 段位管理: 列出全部段位, 新建、编辑、删除
 */
export function TiersMain() {
  const isAdmin = !!useAuth().user?.is_admin;
  const [tiers, setTiers] = useState<Tier[] | null>(null);

  // null: 关闭; "new": 新建; 段位: 编辑
  const [editing, setEditing] = useState<Tier | 'new' | null>(null);
  const [deleting, setDeleting] = useState<Tier | null>(null);

  const load = useCallback(async () => {
    // 确认是管理员后再拉取, 否则 AdminGuard 会跳走
    if (!isAdmin) return;
    try {
      setTiers((await services.admin.listTiers()).tiers);
    } catch (err) {
      toast.error(toErrorMessage(err));
    }
  }, [isAdmin]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 挂载和保存后重新拉取
    void load();
  }, [load]);

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between gap-2">
        <div className="flex flex-col gap-1">
          <h1 className="text-2xl font-bold">段位管理</h1>
          <p className="text-sm text-muted-foreground">
            段位分不低于分数线时属于这个段位, 低于最低的分数线时没有段位; 段位按段位分实时计算, 修改后所有人的段位随即变化
          </p>
        </div>
        <Button onClick={() => setEditing('new')}>
          <Plus />
          新建段位
        </Button>
      </div>

      <div className="rounded-xl border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>段位</TableHead>
              <TableHead>颜色</TableHead>
              <TableHead className="text-right">分数段</TableHead>
              <TableHead className="text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {!tiers ? (
              <TableSkeleton cols={4} />
            ) : tiers.length ? (
              tiers.map((tier, i) => (
                <TierRow
                  key={tier.id}
                  tier={tier}
                  next={tiers[i + 1] ?? null}
                  onEdit={() => setEditing(tier)}
                  onDelete={() => setDeleting(tier)}
                />
              ))
            ) : (
              <TableEmpty colSpan={4}>还没有段位, 所有人都显示为无段位</TableEmpty>
            )}
          </TableBody>
        </Table>
      </div>

      <TierDialog tier={editing} onClose={() => setEditing(null)} onSaved={load} />
      <DeleteTierDialog tier={deleting} onClose={() => setDeleting(null)} onSaved={load} />
    </div>
  );
}

/**
 * 段位列表的一行, 分数段到下一个段位的分数线为止
 */
function TierRow({tier, next, onEdit, onDelete}: {
  tier: Tier;
  next: Tier | null;
  onEdit: () => void;
  onDelete: () => void;
}) {
  return (
    <TableRow>
      <TableCell>
        <TierName tier={tier} />
      </TableCell>
      <TableCell className="font-mono text-xs">{tier.color}</TableCell>
      <TableCell className="text-right font-mono tabular-nums">
        {next ? `${tier.min_rating} - ${next.min_rating - 1}` : `${tier.min_rating} 及以上`}
      </TableCell>
      <TableCell>
        <div className="flex justify-end gap-1">
          <Button variant="ghost" size="icon" title="编辑" aria-label="编辑" onClick={onEdit}>
            <Pencil />
          </Button>
          <Button variant="ghost" size="icon" title="删除" aria-label="删除" className="hover:text-destructive" onClick={onDelete}>
            <Trash2 />
          </Button>
        </div>
      </TableCell>
    </TableRow>
  );
}
