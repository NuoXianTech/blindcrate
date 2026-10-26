import { CrateHero } from "@/components/common/home/CrateHero";
import { StartGameButton } from "@/components/common/home/StartGameButton";

/** 首页: 独立于 (main), 不显示 dock 导航 */
export default function HomePage() {
  return (
    <div className="flex flex-1 flex-col items-center justify-center gap-8 px-4 text-center">
      <CrateHero />
      <div className="flex flex-col items-center gap-3">
        <h1 className="text-4xl font-bold tracking-tight">BlindCrate</h1>
        <p className="max-w-lg text-balance text-muted-foreground">开盲盒收集物品, 把战利品带上保险柜竞拍, 和其他玩家一决高下。</p>
      </div>
      <StartGameButton />
    </div>
  );
}
