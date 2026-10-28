import type {Metadata} from 'next';
import {Suspense} from 'react';
import {AuctionMain} from '@/components/common/auction';

export const metadata: Metadata = {title: '对局'};

/** 对局页: 全屏的拍卖桌 */
export default function MatchPage({params}: PageProps<'/match/[id]'>) {
  // 对局 id 要到请求时才知道, 需要包在 Suspense 里
  return (
    <Suspense>
      <MatchTable params={params} />
    </Suspense>
  );
}

async function MatchTable({params}: Pick<PageProps<'/match/[id]'>, 'params'>) {
  const {id} = await params;
  return <AuctionMain matchId={Number(id)} />;
}
