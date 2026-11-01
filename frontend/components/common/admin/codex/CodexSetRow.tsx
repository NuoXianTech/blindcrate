import {Copy, Pencil, Power, Trash2} from 'lucide-react';
import {Badge} from '@/components/ui/badge';
import {Button} from '@/components/ui/button';
import {TableCell, TableRow} from '@/components/ui/table';
import {bundleText} from '@/components/common/admin/editors';
import type {AdminItem, Currency} from '@/lib/services/admin';
import type {AdminCodexSet} from '@/lib/services/codex';

/**
 * 收集组行的Props接口
 */
interface CodexSetRowProps {
  set: AdminCodexSet;
  items: AdminItem[];
  currencies: Currency[];
  /** 已下架、已删除或不是藏品的成员数, 物品目录加载完成前为 null */
  excluded: number | null;
  onEdit: () => void;
  onCopy: () => void;
  onToggle: () => void;
  onDelete: () => void;
}

/**
 * 收集组列表的一行: 名称、藏品数和不计入进度的数量、奖励、排序、启用状态和操作按钮
 */
export function CodexSetRow({set: s, items, currencies, excluded, onEdit, onCopy, onToggle, onDelete}: CodexSetRowProps) {
  const reward = bundleText(s.reward, items, currencies);
  return (
    <TableRow>
      <TableCell>
        <div className="flex min-w-0 flex-col">
          <span className="font-medium">{s.name}</span>
          <span className="max-w-64 truncate text-xs text-muted-foreground" title={s.description || undefined}>
            <span className="font-mono">{s.code}</span>
            {s.description && ` · ${s.description}`}
          </span>
        </div>
      </TableCell>
      <TableCell>
        <div className="flex flex-col">
          <span className="tabular-nums">{s.item_ids.length} 种</span>
          {!!excluded && <span className="text-xs text-muted-foreground tabular-nums">{excluded} 种已下架或已删除, 不计入</span>}
        </div>
      </TableCell>
      <TableCell className="max-w-48 truncate" title={reward.join(', ')}>
        {reward.join(', ')}
      </TableCell>
      <TableCell className="text-right font-mono tabular-nums">{s.sort_order}</TableCell>
      <TableCell>
        <Badge variant={s.is_enabled ? 'secondary' : 'outline'}>{s.is_enabled ? '已启用' : '已停用'}</Badge>
      </TableCell>
      <TableCell>
        <div className="flex justify-end gap-1">
          <Button variant="ghost" size="icon" title="编辑" aria-label="编辑" onClick={onEdit}>
            <Pencil />
          </Button>
          <Button variant="ghost" size="icon" title="复制为新收集组" aria-label="复制为新收集组" onClick={onCopy}>
            <Copy />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            title={s.is_enabled ? '停用' : '启用'}
            aria-label={s.is_enabled ? '停用' : '启用'}
            className={s.is_enabled ? undefined : 'text-destructive'}
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
