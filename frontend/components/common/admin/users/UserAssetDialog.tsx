import {Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle} from '@/components/ui/dialog';
import type {AdminUser} from '@/lib/services/admin';

/**
 * 钱包、背包调整对话框的外壳
 */
export function UserAssetDialog({
  user,
  title,
  onClose,
  children,
}: {
  user: AdminUser | null;
  title: string;
  onClose: () => void;
  children: React.ReactNode;
}) {
  return (
    <Dialog open={!!user} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>
            {user?.nickname} 的{title}
          </DialogTitle>
          <DialogDescription>
            @{user?.username} #{user?.id} · 调整会写入流水, 用户可以看到备注
          </DialogDescription>
        </DialogHeader>
        {children}
      </DialogContent>
    </Dialog>
  );
}
