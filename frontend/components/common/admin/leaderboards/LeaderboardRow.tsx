import {Pencil, Power, Trash2} from 'lucide-react';
import {Badge} from '@/components/ui/badge';
import {Button} from '@/components/ui/button';
import {TableCell, TableRow} from '@/components/ui/table';
import type {AdminLeaderboard} from '@/lib/services/admin';
import {formatDateTime} from '@/lib/utils';
import {PERIODS} from './constants';

/**
 * 榜单行的Props接口
 */
interface LeaderboardRowProps {
  board: AdminLeaderboard;
  /** 指标的展示名称, 未知指标时为指标本身 */
  metricLabel: string;
  /** 统计范围的展示名称, 未知范围时为范围本身 */
  scopeLabel: string;
  onEdit: () => void;
  onToggle: () => void;
  onDelete: () => void;
}

/**
 * 榜单列表的一行: 名称、指标、范围和周期、上榜规则和操作按钮
 */
export function LeaderboardRow({board, metricLabel, scopeLabel, onEdit, onToggle, onDelete}: LeaderboardRowProps) {
  const period = PERIODS.find((p) => p.value === board.period)?.label ?? board.period;
  return (
    <TableRow>
      <TableCell>
        <div className="flex min-w-0 flex-col">
          <span className="font-medium">{board.name}</span>
          <span className="max-w-64 truncate text-xs text-muted-foreground" title={board.description || undefined}>
            <span className="font-mono">{board.code}</span>
            {board.description && ` · ${board.description}`}
          </span>
        </div>
      </TableCell>
      <TableCell>
        <div className="flex flex-col">
          <span>{metricLabel}</span>
          <span className="text-xs text-muted-foreground">
            {scopeLabel} · {period}
          </span>
        </div>
      </TableCell>
      <TableCell className="text-muted-foreground">
        前 {board.size} 名{board.min_matches > 0 && `, 至少 ${board.min_matches} 局`}
      </TableCell>
      <TableCell className="text-right font-mono tabular-nums">{board.sort_order}</TableCell>
      <TableCell>
        {board.is_enabled ? <Badge variant="secondary">启用中</Badge> : <Badge variant="outline">已停用</Badge>}
      </TableCell>
      <TableCell className="text-muted-foreground">{formatDateTime(board.updated_at)}</TableCell>
      <TableCell>
        <div className="flex justify-end gap-1">
          <Button variant="ghost" size="icon" title="编辑" aria-label="编辑" onClick={onEdit}>
            <Pencil />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            title={board.is_enabled ? '停用' : '启用'}
            aria-label={board.is_enabled ? '停用' : '启用'}
            className={board.is_enabled ? undefined : 'text-destructive'}
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
