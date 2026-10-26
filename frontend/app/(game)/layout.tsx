/** 游戏页面布局: 全屏沉浸, 不带顶部导航 */
export default function GameLayout({ children }: LayoutProps<"/">) {
  return <div className="flex h-dvh flex-col overflow-hidden">{children}</div>;
}
