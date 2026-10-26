/** 登录、注册等页面的布局: 内容居中, 不带导航 */
export default function AuthLayout({ children }: LayoutProps<"/">) {
  return (
    <div className="flex flex-1 items-center justify-center px-4 py-12">
      {children}
    </div>
  );
}
