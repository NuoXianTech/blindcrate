'use client';

import {useCallback, useEffect, useState} from 'react';
import {LoaderCircle, Undo2} from 'lucide-react';
import {toast} from 'sonner';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import {Badge} from '@/components/ui/badge';
import {Button} from '@/components/ui/button';
import {Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle} from '@/components/ui/dialog';
import {Skeleton} from '@/components/ui/skeleton';
import {Pagination} from '@/components/common/layout';
import {useSubmit} from '@/hooks/use-submit';
import services from '@/lib/services';
import type {ListSettingRevisionsData, SettingGroup, SettingRevision} from '@/lib/services/admin';
import {toErrorMessage} from '@/lib/services/core';
import {formatDateTime} from '@/lib/utils';
import {formatChange, REVISION_ACTIONS, REVISION_PAGE_SIZE} from './constants';

/**
 * 历史版本的Props接口
 */
interface SettingHistoryProps {
  group: SettingGroup;
  open: boolean;
  onClose: () => void;
  /** 回滚成功后返回保存后的设置组 */
  onRolledBack: (group: SettingGroup) => void;
  /** 已被别人修改时重新加载设置组 */
  onConflict: (err: unknown) => Promise<boolean>;
}

/**
 * 设置组的历史版本: 每个版本改动的字段, 可以回滚到任意一个旧版本
 */
export function SettingHistory({group, open, onClose, onRolledBack, onConflict}: SettingHistoryProps) {
  const [current, setCurrent] = useState(1);
  const [data, setData] = useState<ListSettingRevisionsData | null>(null);
  const [target, setTarget] = useState<SettingRevision | null>(null);
  const {pending, submit} = useSubmit();

  const load = useCallback(async () => {
    try {
      setData(await services.admin.listSettingRevisions(group.key, {current, size: REVISION_PAGE_SIZE}));
    } catch (err) {
      toast.error(toErrorMessage(err));
    }
  }, [group.key, current]);

  // 打开时和版本变化后(如回滚后)重新拉取
  useEffect(() => {
    if (!open) return;
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 打开或翻页时拉取
    void load();
  }, [open, load, group.version]);

  const handleRollback = submit(async () => {
    if (!target) return;
    try {
      const updated = await services.admin.rollbackSetting(group.key, {version: group.version, target: target.version});
      toast.success(updated.version === group.version ? '内容与当前版本相同, 没有变化' : `已回滚到版本 ${target.version}`);
      onRolledBack(updated);
      setCurrent(1);
      setTarget(null);
    } catch (err) {
      if (await onConflict(err)) {
        setTarget(null);
        return;
      }
      throw err;
    }
  });

  return (
    <>
      <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
        <DialogContent className="sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle>{group.label} · 历史版本</DialogTitle>
            <DialogDescription>每次保存、回滚和恢复默认值都会生成一个版本, 回滚也会生成新版本</DialogDescription>
          </DialogHeader>
          <div className="flex max-h-[60vh] flex-col gap-2 overflow-y-auto">
            {!data ? (
              <>
                <Skeleton className="h-20 rounded-lg" />
                <Skeleton className="h-20 rounded-lg" />
              </>
            ) : data.revisions.length ? (
              data.revisions.map((r) => (
                <RevisionRow
                  key={r.version}
                  group={group}
                  revision={r}
                  onRollback={() => setTarget(r)}
                />
              ))
            ) : (
              <p className="py-6 text-center text-sm text-muted-foreground">还没有保存过, 当前使用默认值</p>
            )}
          </div>
          {!!data && data.total > REVISION_PAGE_SIZE && (
            <Pagination current={current} size={REVISION_PAGE_SIZE} total={data.total} onChange={setCurrent} />
          )}
        </DialogContent>
      </Dialog>

      <AlertDialog open={!!target} onOpenChange={(o) => !o && !pending && setTarget(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>回滚到版本 {target?.version}</AlertDialogTitle>
            <AlertDialogDescription>
              {group.label}的全部字段恢复为版本 {target?.version} 的内容并立即生效, 包括密钥字段。回滚会生成一个新版本, 之后还可以再回滚回来。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={pending}>取消</AlertDialogCancel>
            <AlertDialogAction disabled={pending} onClick={handleRollback}>
              {pending && <LoaderCircle className="animate-spin" />}
              回滚
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}

/**
 * 一个版本: 版本号、来源、操作人、时间和改动的字段
 */
function RevisionRow({group, revision: r, onRollback}: {
  group: SettingGroup;
  revision: SettingRevision;
  onRollback: () => void;
}) {
  const isCurrent = r.version === group.version;
  const action = r.action === 'rollback' ? `回滚到版本 ${r.from_version}` : REVISION_ACTIONS[r.action] ?? r.action;
  const changes = Object.entries(r.changes ?? {});

  return (
    <div className="flex flex-col gap-2 rounded-lg border p-3">
      <div className="flex flex-wrap items-center gap-2 text-sm">
        <Badge variant={isCurrent ? 'default' : 'secondary'} className="font-mono">
          v{r.version}
        </Badge>
        <span className="font-medium">{action}</span>
        {isCurrent && <span className="text-xs text-muted-foreground">当前版本</span>}
        <span className="ml-auto text-xs text-muted-foreground">
          {r.updated_by_username || (r.updated_by ? `#${r.updated_by}` : '系统')} · {formatDateTime(r.created_at)}
        </span>
        {!isCurrent && (
          <Button type="button" variant="outline" size="sm" onClick={onRollback}>
            <Undo2 />
            回滚到此版本
          </Button>
        )}
      </div>
      {r.action === 'baseline' ? (
        <p className="text-xs text-muted-foreground">启用版本记录前保存的内容</p>
      ) : changes.length ? (
        <ul className="flex flex-col gap-1 text-xs">
          {changes.map(([key, change]) => {
            const field = group.fields.find((f) => f.key === key);
            return (
              <li key={key} className="flex gap-2">
                <span className="shrink-0 text-muted-foreground">{field?.label ?? key}</span>
                <span className="min-w-0 break-all">{formatChange(field, change)}</span>
              </li>
            );
          })}
        </ul>
      ) : (
        <p className="text-xs text-muted-foreground">没有改动的字段</p>
      )}
    </div>
  );
}
