import {CircleStop, Pencil, Trash2} from 'lucide-react';
import {Badge} from '@/components/ui/badge';
import {Button} from '@/components/ui/button';
import {TableCell, TableRow} from '@/components/ui/table';
import {SEASON_STATUS, type Season} from '@/lib/services/season';
import {formatDateTime} from '@/lib/utils';
import {STATUS_LABELS} from './constants';

/**
 * 赛季行的Props接口
 */
interface SeasonRowProps {
  season: Season;
  onEdit: () => void;
  onEnd: () => void;
  onDelete: () => void;
}

/**
 * 赛季列表的一行: 名称、时间、状态和操作按钮; 只有进行中的赛季能提前结束, 只有未开始的赛季能删除
 */
export function SeasonRow({season, onEdit, onEnd, onDelete}: SeasonRowProps) {
  return (
    <TableRow>
      <TableCell>
        <div className="flex flex-col">
          <span className="font-medium">{season.name}</span>
          <span className="font-mono text-xs text-muted-foreground">{season.code}</span>
        </div>
      </TableCell>
      <TableCell className="text-muted-foreground tabular-nums">{formatDateTime(season.starts_at)}</TableCell>
      <TableCell className="text-muted-foreground tabular-nums">{formatDateTime(season.ends_at)}</TableCell>
      <TableCell>
        <Badge variant={season.status === SEASON_STATUS.active ? 'default' : season.status === SEASON_STATUS.upcoming ? 'secondary' : 'outline'}>
          {STATUS_LABELS[season.status]}
        </Badge>
      </TableCell>
      <TableCell className="text-muted-foreground">{formatDateTime(season.updated_at)}</TableCell>
      <TableCell>
        <div className="flex justify-end gap-1">
          <Button variant="ghost" size="icon" title="编辑" aria-label="编辑" onClick={onEdit}>
            <Pencil />
          </Button>
          {season.status === SEASON_STATUS.active && (
            <Button variant="ghost" size="icon" title="提前结束" aria-label="提前结束" className="hover:text-destructive" onClick={onEnd}>
              <CircleStop />
            </Button>
          )}
          {season.status === SEASON_STATUS.upcoming && (
            <Button variant="ghost" size="icon" title="删除" aria-label="删除" className="hover:text-destructive" onClick={onDelete}>
              <Trash2 />
            </Button>
          )}
        </div>
      </TableCell>
    </TableRow>
  );
}
