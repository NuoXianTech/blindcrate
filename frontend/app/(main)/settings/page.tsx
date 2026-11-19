import type {Metadata} from 'next';
import {GameSettingsMain} from '@/components/common/player-settings/GameSettingsMain';

export const metadata: Metadata = {title: '游戏设置'};

export default function GameSettingsPage() {
  return <GameSettingsMain />;
}
