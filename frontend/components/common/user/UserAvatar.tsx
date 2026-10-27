import {cn} from 'cn';
import type {UserInfo} from '@/lib/services/auth';

/** 用户头像, 未设置头像时显示昵称首字 */
export function UserAvatar({
  user,
  className,
}: {
  user: Pick<UserInfo, 'nickname' | 'avatar_url'>;
  className?: string;
}) {
  return (
    <span
      className={cn(
          'flex size-full shrink-0 items-center justify-center overflow-hidden rounded-full bg-primary/20 font-bold text-primary uppercase',
          className,
      )}
    >
      {user.avatar_url ? (
        // 头像地址由用户填写, 域名不固定, 不走 next/image
        // eslint-disable-next-line @next/next/no-img-element
        <img src={user.avatar_url} alt={user.nickname} className="size-full object-cover" />
      ) : (
        user.nickname.slice(0, 1)
      )}
    </span>
  );
}
