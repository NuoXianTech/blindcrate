'use client';

import {Badge} from '@/components/ui/badge';
import {Table, TableBody, TableCell, TableHead, TableHeader, TableRow} from '@/components/ui/table';
import {TableEmpty} from '@/components/common/layout';
import type {PackEntryResult} from '@/lib/services/admin';
import {ACTION_LABELS, STATUS_LABELS, entryID} from './constants';
import {sectionLabel, usePackSections} from './use-pack-sections';

/**
 * 配置包中每一项的比较结果; 传入 renderAction 时最后一列为处理方式的选择, 否则显示已选的处理方式
 */
export function PackEntriesTable({entries, renderAction, empty = '没有需要处理的配置'}: {
  entries: PackEntryResult[];
  renderAction?: (e: PackEntryResult) => React.ReactNode;
  empty?: string;
}) {
  const {sections} = usePackSections();
  return (
    <div className="rounded-xl border bg-card">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>种类</TableHead>
            <TableHead>代码</TableHead>
            <TableHead>名称</TableHead>
            <TableHead>结果</TableHead>
            <TableHead>处理方式</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {entries.length ? (
            entries.map((e) => {
              const status = STATUS_LABELS[e.status];
              return (
                <TableRow key={entryID(e)}>
                  <TableCell className="text-muted-foreground">{sectionLabel(sections, e.section)}</TableCell>
                  <TableCell className="font-mono">{e.section === 'items' ? `#${e.key}` : e.key}</TableCell>
                  <TableCell>{e.name}</TableCell>
                  <TableCell>
                    <div className="flex flex-col items-start gap-0.5">
                      <Badge variant={status?.variant ?? 'outline'}>{status?.label ?? e.status}</Badge>
                      {e.reason && <span className="text-xs text-destructive">{e.reason}</span>}
                    </div>
                  </TableCell>
                  <TableCell>
                    {renderAction ? renderAction(e) : (
                      <span className={e.action === 'skip' ? 'text-muted-foreground' : undefined}>
                        {ACTION_LABELS[e.action] ?? e.action}
                        {e.code && <span className="font-mono"> → {e.code}</span>}
                      </span>
                    )}
                  </TableCell>
                </TableRow>
              );
            })
          ) : (
            <TableEmpty colSpan={5}>{empty}</TableEmpty>
          )}
        </TableBody>
      </Table>
    </div>
  );
}
