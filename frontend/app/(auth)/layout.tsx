import {SiteFooter} from '@/components/common/site';

/** 登录、注册等页面的布局: 内容居中, 不带导航, 底部备案号和联系邮箱 */
export default function AuthLayout({children}: LayoutProps<'/'>) {
  return (
    <div className="flex flex-1 flex-col items-center px-4 py-6">
      <div className="flex w-full flex-1 items-center justify-center py-6">{children}</div>
      <SiteFooter />
    </div>
  );
}
