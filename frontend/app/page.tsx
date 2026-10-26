import { StartGameButton } from "@/components/common/home/StartGameButton";

/** 首页: 独立于 (main), 不显示 dock 导航 */
export default function HomePage() {
  return (
    <div className="flex flex-1 flex-col items-center justify-center gap-6 px-4 text-center">
      <h1 className="text-4xl font-bold tracking-tight">
        Blind<span className="text-primary">Crate</span>
      </h1>
      <p className="text-muted-foreground">盲盒开箱 · 保险柜竞拍对战平台</p>
      <StartGameButton />
    </div>
  );
}
