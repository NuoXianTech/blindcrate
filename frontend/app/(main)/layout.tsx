import { ManagementBar } from "@/components/common/layout/ManagementBar";

/** 游戏内页面布局(游戏首页等): 居中内容 + 底部悬浮 dock 导航, 底部留白避免 dock 遮挡内容 */
export default function MainLayout({ children }: LayoutProps<"/">) {
  return (
    <div className="flex flex-1 flex-col px-4 pt-12 pb-28 sm:px-6 md:px-8 lg:px-12">
      <div className="mx-auto flex w-full max-w-7xl flex-1 flex-col gap-4 md:gap-6">
        {children}
      </div>
      <ManagementBar />
    </div>
  );
}
