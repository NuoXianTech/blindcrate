import {Pencil, Power, Trash2} from 'lucide-react';
import {Badge} from '@/components/ui/badge';
import {Button} from '@/components/ui/button';
import {TableCell, TableRow} from '@/components/ui/table';
import {bundleText} from '@/components/common/admin/editors';
import type {AdminItem, AdminVenue, Currency} from '@/lib/services/admin';
import {formatAmount, formatDateTime} from '@/lib/utils';
import {MODES} from './constants';

/**
 * 地图行的Props接口
 */
interface VenueRowProps {
  venue: AdminVenue;
  items: AdminItem[];
  currencies: Currency[];
  onEdit: () => void;
  onToggle: () => void;
  onDelete: () => void;
}

/**
 * 地图列表的一行: 名称、开放的模式、入场代价、模拟的价值中位数和操作按钮
 */
export function VenueRow({venue, items, currencies, onEdit, onToggle, onDelete}: VenueRowProps) {
  const bid = currencies.find((c) => c.code === venue.bid_currency);
  const cost = bundleText(venue.entry_cost, items, currencies);
  const modes = MODES.filter((m) => (m.value === 'pve' ? venue.pve_enabled : venue.ranked_enabled));

  return (
    <TableRow>
      <TableCell>
        <div className="flex min-w-0 flex-col">
          <span className="font-medium">{venue.name}</span>
          <span className="max-w-64 truncate text-xs text-muted-foreground" title={venue.description || undefined}>
            <span className="font-mono">{venue.code}</span>
            {venue.description && ` · ${venue.description}`}
          </span>
        </div>
      </TableCell>
      <TableCell>
        <div className="flex gap-1">
          {modes.length ? (
            modes.map((m) => (
              <Badge key={m.value} variant="secondary">
                {m.label}
              </Badge>
            ))
          ) : (
            <span className="text-muted-foreground">-</span>
          )}
        </div>
      </TableCell>
      <TableCell className="max-w-48 truncate" title={cost.join(', ') || undefined}>
        {cost.length ? cost.join(', ') : <span className="text-muted-foreground">免费</span>}
      </TableCell>
      <TableCell className="text-right font-mono tabular-nums" title="排位模式模拟的仓库价值中位数">
        {formatAmount(venue.simulation.ranked.p50, bid?.decimals ?? 0)}
      </TableCell>
      <TableCell>
        {venue.is_enabled ? <Badge variant="secondary">启用中</Badge> : <Badge variant="outline">已停用</Badge>}
      </TableCell>
      <TableCell className="text-muted-foreground">{formatDateTime(venue.updated_at)}</TableCell>
      <TableCell>
        <div className="flex justify-end gap-1">
          <Button variant="ghost" size="icon" title="编辑" aria-label="编辑" onClick={onEdit}>
            <Pencil />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            title={venue.is_enabled ? '停用' : '启用'}
            aria-label={venue.is_enabled ? '停用' : '启用'}
            className={venue.is_enabled ? undefined : 'text-destructive'}
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
