import type {Metadata} from 'next';
import {Package} from 'lucide-react';

export const metadata: Metadata = {title: '游戏大厅'};

/** 游戏大厅, 玩法开发中 */
export default function GameHomePage() {
  return (
    <div className="flex flex-1 flex-col items-center justify-center gap-3 px-4 pb-24 text-center">
      <div className="flex size-12 items-center justify-center rounded-xl border bg-card text-primary">
        <Package className="size-6" />
      </div>
      <h1 className="text-xl font-bold">开箱玩法正在开发中</h1>
      <p className="max-w-sm text-sm text-muted-foreground">
        开箱、保险柜和竞拍会陆续在这里开放。现在可以点下方的头像完善个人资料、绑定邮箱。
      </p>
    </div>
  );
}
