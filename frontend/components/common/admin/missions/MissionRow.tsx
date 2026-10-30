import {Copy, Pencil, Power, Trash2} from 'lucide-react';
import {Badge} from '@/components/ui/badge';
import {Button} from '@/components/ui/button';
import {TableCell, TableRow} from '@/components/ui/table';
import {bundleText} from '@/components/common/admin/editors';
import type {AdminItem, Currency} from '@/lib/services/admin';
import type {AdminMission} from '@/lib/services/mission';
import {formatAmount} from '@/lib/utils';
import {metricCurrency} from './constants';

/**
 * 任务行的Props接口
 */
interface MissionRowProps {
  mission: AdminMission;
  /** 周期的展示名称, 未知周期时为周期本身 */
  cycleLabel: string;
  /** 指标的展示名称, 未知指标时为指标本身 */
  metricLabel: string;
  /** 统计范围的展示名称, 未知范围时为范围本身 */
  scopeLabel: string;
  items: AdminItem[];
  currencies: Currency[];
  onEdit: () => void;
  onCopy: () => void;
  onToggle: () => void;
  onDelete: () => void;
}

/**
 * 任务列表的一行: 名称、周期、指标和目标、奖励、条件数、启用状态和操作按钮
 */
export function MissionRow({mission: m, cycleLabel, metricLabel, scopeLabel, items, currencies, onEdit, onCopy, onToggle, onDelete}: MissionRowProps) {
  const reward = bundleText(m.reward, items, currencies);
  const currency = metricCurrency(m.metric, currencies);
  return (
    <TableRow>
      <TableCell>
        <div className="flex min-w-0 flex-col">
          <span className="font-medium">{m.name}</span>
          <span className="max-w-64 truncate text-xs text-muted-foreground" title={m.description || undefined}>
            <span className="font-mono">{m.code}</span>
            {m.description && ` · ${m.description}`}
          </span>
        </div>
      </TableCell>
      <TableCell>{cycleLabel}</TableCell>
      <TableCell>
        <div className="flex flex-col">
          <span>
            {metricLabel} ≥ <span className="font-mono tabular-nums">{formatAmount(m.target, currency?.decimals ?? 0)}</span>
          </span>
          <span className="text-xs text-muted-foreground">{scopeLabel}</span>
        </div>
      </TableCell>
      <TableCell className="max-w-48 truncate" title={reward.join(', ')}>
        {reward.join(', ')}
      </TableCell>
      <TableCell className="text-right tabular-nums">
        {m.requirements.length || <span className="text-muted-foreground">-</span>}
      </TableCell>
      <TableCell className="text-right font-mono tabular-nums">{m.sort_order}</TableCell>
      <TableCell>
        <Badge variant={m.is_enabled ? 'secondary' : 'outline'}>{m.is_enabled ? '已启用' : '已停用'}</Badge>
      </TableCell>
      <TableCell>
        <div className="flex justify-end gap-1">
          <Button variant="ghost" size="icon" title="编辑" aria-label="编辑" onClick={onEdit}>
            <Pencil />
          </Button>
          <Button variant="ghost" size="icon" title="复制为新任务" aria-label="复制为新任务" onClick={onCopy}>
            <Copy />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            title={m.is_enabled ? '停用' : '启用'}
            aria-label={m.is_enabled ? '停用' : '启用'}
            className={m.is_enabled ? undefined : 'text-destructive'}
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
