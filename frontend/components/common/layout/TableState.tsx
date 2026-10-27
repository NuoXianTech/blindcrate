import {Skeleton} from '@/components/ui/skeleton';
import {TableCell, TableRow} from '@/components/ui/table';

/**
 * 表格首次加载时的骨架行
 */
export function TableSkeleton({cols, rows = 5}: { cols: number; rows?: number }) {
  return (
    <>
      {Array.from({length: rows}, (_, i) => (
        <TableRow key={i} className="hover:bg-transparent">
          {Array.from({length: cols}, (_, j) => (
            <TableCell key={j} className="h-12">
              <Skeleton className="h-4 w-full max-w-32" />
            </TableCell>
          ))}
        </TableRow>
      ))}
    </>
  );
}

/**
 * 表格没有数据时的提示行
 */
export function TableEmpty({colSpan, children}: { colSpan: number; children: React.ReactNode }) {
  return (
    <TableRow className="hover:bg-transparent">
      <TableCell colSpan={colSpan} className="h-24 text-center text-muted-foreground">
        {children}
      </TableCell>
    </TableRow>
  );
}
