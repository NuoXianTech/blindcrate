/** 主站布局: 顶部悬浮 dock 导航 + 居中内容, dock 后续实现 */
export default function MainLayout({ children }: LayoutProps<"/">) {
  return (
    <div className="flex flex-1 flex-col px-4 py-12 sm:px-6 md:px-8 lg:px-12">
      <div className="mx-auto flex w-full max-w-7xl flex-1 flex-col gap-4 md:gap-6">
        {children}
      </div>
    </div>
  );
}
