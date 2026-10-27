import {ChevronLeft, ChevronRight} from 'lucide-react';
import {Button} from '@/components/ui/button';

/** 简单分页: 共几条 + 上一页/下一页 */
export function Pagination({
  current,
  size,
  total,
  onChange,
}: {
  current: number;
  size: number;
  total: number;
  onChange: (page: number) => void;
}) {
  const pages = Math.max(1, Math.ceil(total / size));

  return (
    <div className="flex items-center justify-between gap-4 text-sm text-muted-foreground">
      <span>共 {total} 条</span>
      <div className="flex items-center gap-2">
        <Button variant="outline" size="icon" aria-label="上一页" disabled={current <= 1} onClick={() => onChange(current - 1)}>
          <ChevronLeft />
        </Button>
        <span className="min-w-16 text-center">
          {current} / {pages}
        </span>
        <Button variant="outline" size="icon" aria-label="下一页" disabled={current >= pages} onClick={() => onChange(current + 1)}>
          <ChevronRight />
        </Button>
      </div>
    </div>
  );
}
