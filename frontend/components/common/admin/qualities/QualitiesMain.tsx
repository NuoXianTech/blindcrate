'use client';

import {useCallback, useEffect, useState} from 'react';
import {Pencil, Plus, Trash2} from 'lucide-react';
import {toast} from 'sonner';
import {Badge} from '@/components/ui/badge';
import {Button} from '@/components/ui/button';
import {Table, TableBody, TableCell, TableHead, TableHeader, TableRow} from '@/components/ui/table';
import {useAuth} from '@/components/common/auth';
import {TableEmpty, TableSkeleton} from '@/components/common/layout';
import services from '@/lib/services';
import type {AdminQuality} from '@/lib/services/admin';
import {toErrorMessage} from '@/lib/services/core';
import {EFFECTS} from './constants';
import {DeleteQualityDialog} from './DeleteQualityDialog';
import {QualityDialog} from './QualityDialog';
import {QualityPreview} from './QualityPreview';

/**
 * 品质管理: 列出全部品质, 新建、编辑、删除
 */
export function QualitiesMain() {
  const isAdmin = !!useAuth().user?.is_admin;
  const [qualities, setQualities] = useState<AdminQuality[] | null>(null);

  // null: 关闭; "new": 新建; 品质: 编辑
  const [editing, setEditing] = useState<AdminQuality | 'new' | null>(null);
  const [deleting, setDeleting] = useState<AdminQuality | null>(null);

  const load = useCallback(async () => {
    // 确认是管理员后再拉取, 否则 AdminGuard 会跳走
    if (!isAdmin) return;
    try {
      setQualities((await services.admin.listQualities()).qualities);
    } catch (err) {
      toast.error(toErrorMessage(err));
    }
  }, [isAdmin]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 挂载和保存后重新拉取
    void load();
  }, [load]);

  const nextLevel = Math.max(0, ...(qualities ?? []).map((q) => q.level)) + 1;

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between gap-2">
        <div className="flex flex-col gap-1">
          <h1 className="text-2xl font-bold">品质管理</h1>
          <p className="text-sm text-muted-foreground">等级越大越稀有; 品质的名称和颜色全站通用, 扫描时长决定开箱时每件藏品扫多久</p>
        </div>
        <Button onClick={() => setEditing('new')}>
          <Plus />
          新建品质
        </Button>
      </div>

      <div className="rounded-xl border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>等级</TableHead>
              <TableHead>品质</TableHead>
              <TableHead>颜色</TableHead>
              <TableHead className="text-right">扫描时长</TableHead>
              <TableHead className="text-right">排序</TableHead>
              <TableHead className="text-right">物品数</TableHead>
              <TableHead className="text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {!qualities ? (
              <TableSkeleton cols={7} />
            ) : qualities.length ? (
              qualities.map((q) => (
                <QualityRow key={q.level} quality={q} onEdit={() => setEditing(q)} onDelete={() => setDeleting(q)} />
              ))
            ) : (
              <TableEmpty colSpan={7}>还没有品质</TableEmpty>
            )}
          </TableBody>
        </Table>
      </div>

      <QualityDialog quality={editing} nextLevel={nextLevel} onClose={() => setEditing(null)} onSaved={load} />
      <DeleteQualityDialog quality={deleting} onClose={() => setDeleting(null)} onSaved={load} />
    </div>
  );
}

/**
 * 品质列表的一行
 */
function QualityRow({quality: q, onEdit, onDelete}: {
  quality: AdminQuality;
  onEdit: () => void;
  onDelete: () => void;
}) {
  const effect = q.effect ? EFFECTS.find((e) => e.value === q.effect)?.label ?? q.effect : null;
  return (
    <TableRow>
      <TableCell className="font-mono tabular-nums">{q.level}</TableCell>
      <TableCell>
        <div className="flex min-w-0 flex-col">
          <QualityPreview name={q.name} color={q.color} effect={q.effect} />
          <span className="pl-5 font-mono text-xs text-muted-foreground">{q.code}</span>
        </div>
      </TableCell>
      <TableCell>
        <span className="flex items-center gap-2">
          <span className="font-mono text-xs">{q.color}</span>
          {effect && <Badge variant="outline">{effect}</Badge>}
        </span>
      </TableCell>
      <TableCell className="text-right font-mono tabular-nums">{q.scan_ms} 毫秒</TableCell>
      <TableCell className="text-right font-mono tabular-nums">{q.sort_order}</TableCell>
      <TableCell className="text-right font-mono tabular-nums">{q.item_count}</TableCell>
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
