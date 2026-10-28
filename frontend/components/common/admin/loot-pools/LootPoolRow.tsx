import {ListTree, Pencil, Power, Trash2} from 'lucide-react';
import {Badge} from '@/components/ui/badge';
import {Button} from '@/components/ui/button';
import {TableCell, TableRow} from '@/components/ui/table';
import type {LootPoolListItem} from '@/lib/services/admin';
import {formatDateTime} from '@/lib/utils';

/**
 * 掉落池行的Props接口
 */
interface LootPoolRowProps {
  pool: LootPoolListItem;
  onEntries: () => void;
  onEdit: () => void;
  onToggle: () => void;
  onDelete: () => void;
}

/**
 * 掉落池列表的一行: 代码、名称、条目数和操作按钮
 */
export function LootPoolRow({pool, onEntries, onEdit, onToggle, onDelete}: LootPoolRowProps) {
  return (
    <TableRow>
      <TableCell>
        <div className="flex min-w-0 flex-col">
          <span className="font-medium">{pool.name}</span>
          <span className="max-w-64 truncate text-xs text-muted-foreground" title={pool.description || undefined}>
            <span className="font-mono">{pool.code}</span>
            {pool.description && ` · ${pool.description}`}
          </span>
        </div>
      </TableCell>
      <TableCell className="text-right font-mono tabular-nums">{pool.entry_count}</TableCell>
      <TableCell>
        {pool.is_enabled ? <Badge variant="secondary">启用中</Badge> : <Badge variant="outline">已停用</Badge>}
      </TableCell>
      <TableCell className="text-muted-foreground">{formatDateTime(pool.updated_at)}</TableCell>
      <TableCell>
        <div className="flex justify-end gap-1">
          <Button variant="ghost" size="icon" title="编辑条目" aria-label="编辑条目" onClick={onEntries}>
            <ListTree />
          </Button>
          <Button variant="ghost" size="icon" title="编辑" aria-label="编辑" onClick={onEdit}>
            <Pencil />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            title={pool.is_enabled ? '停用' : '启用'}
            aria-label={pool.is_enabled ? '停用' : '启用'}
            className={pool.is_enabled ? undefined : 'text-destructive'}
            onClick={onToggle}
          >
            <Power />
          </Button>
          <Button variant="ghost" size="icon" title="删除" aria-label="删除" className="hover:text-destructive" onClick={onDelete}>
            <Trash2 />
          </Button>
        </div>
      </TableCell>
    </TableRow>
  );
}
