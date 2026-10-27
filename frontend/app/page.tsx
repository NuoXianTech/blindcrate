import {Announcement, SiteFooter} from '@/components/common/site';
import {CrateHero, StartGameButton} from '@/components/common/home';
import {getPublicSettings} from '@/lib/site';

/** 首页: 独立于 (main), 不显示 dock 导航; 名称、简介和公告来自系统设置 */
export default async function HomePage() {
  const {site} = await getPublicSettings();
  return (
    <div className="flex flex-1 flex-col items-center gap-6 px-4 py-6">
      <Announcement className="w-full max-w-2xl" />
      <div className="flex flex-1 flex-col items-center justify-center gap-8 text-center">
        <CrateHero />
        <div className="flex flex-col items-center gap-3">
          <h1 className="text-4xl font-bold tracking-tight">{site.name}</h1>
          <p className="max-w-lg text-balance whitespace-pre-wrap text-muted-foreground">
            {site.description || '开盲盒收集物品, 把战利品带上保险柜竞拍, 和其他玩家一决高下。'}
          </p>
        </div>
        <StartGameButton />
      </div>
      <SiteFooter />
    </div>
  );
}
