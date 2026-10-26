import { Button } from "@/components/ui/button";

export default function HomePage() {
  return (
    <div className="flex flex-1 flex-col items-center justify-center gap-6 text-center">
      <h1 className="text-4xl font-bold tracking-tight">
        Blind<span className="text-primary">Crate</span>
      </h1>
      <p className="text-muted-foreground">盲盒开箱 · 保险柜竞拍对战平台</p>
      <Button className="glow">开始游戏</Button>
    </div>
  );
}
