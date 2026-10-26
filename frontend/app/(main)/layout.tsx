import { AuthGuard } from "@/components/common/auth/AuthGuard";
import { ManagementBar } from "@/components/common/layout/ManagementBar";

/** 游戏内页面布局(游戏首页等), 需要登录(其下每个页面都要声明 instant = false, 见 home/page.tsx): 居中内容 + 底部悬浮 dock 导航, 底部留白避免 dock 遮挡内容 */
export default function MainLayout({ children }: LayoutProps<"/">) {
  return (
    <div className="flex flex-1 flex-col px-4 pt-12 pb-28 sm:px-6 md:px-8 lg:px-12">
      <div className="mx-auto flex w-full max-w-7xl flex-1 flex-col gap-4 md:gap-6">
        <AuthGuard>{children}</AuthGuard>
      </div>
      <ManagementBar />
    </div>
  );
}
