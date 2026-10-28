import type {Metadata} from 'next';
import {LobbyMain} from '@/components/common/lobby';

export const metadata: Metadata = {title: '游戏大厅'};

export default function GameHomePage() {
  return <LobbyMain />;
}
