import type {Metadata} from 'next';
// eslint-disable-next-line camelcase
import {Inter, JetBrains_Mono, Noto_Sans_SC} from 'next/font/google';
import {AuthProvider} from '@/components/common/auth';
import {SiteProvider} from '@/components/common/site';
import {ThemeProvider} from '@/components/common/layout';
import {Toaster} from '@/components/ui/sonner';
import {DEFAULT_DESCRIPTION, getPublicSettings} from '@/lib/site';
import './globals.css';

// eslint-disable-next-line new-cap
const inter = Inter({
  variable: '--font-inter',
  subsets: ['latin'],
  display: 'swap',
});

// eslint-disable-next-line new-cap
const notoSansSC = Noto_Sans_SC({
  variable: '--font-noto-sans-sc',
  subsets: ['latin'],
  display: 'swap',
  weight: ['400', '500', '600', '700'],
});

// 数字(价格、倒计时)用等宽字体, 跳动时不会左右晃
// eslint-disable-next-line new-cap
const jetbrainsMono = JetBrains_Mono({
  variable: '--font-jetbrains-mono',
  subsets: ['latin'],
  display: 'swap',
});

/** 站点名称和简介来自系统设置 */
export async function generateMetadata(): Promise<Metadata> {
  const {site} = await getPublicSettings();
  return {
    title: {
      template: `%s - ${site.name}`,
      default: site.name,
    },
    description: site.description || DEFAULT_DESCRIPTION,
  };
}

export default async function RootLayout({children}: LayoutProps<'/'>) {
  const settings = await getPublicSettings();
  return (
    <html
      lang="zh-CN"
      className={`${inter.variable} ${notoSansSC.variable} ${jetbrainsMono.variable} dark h-full font-sans antialiased`}
      suppressHydrationWarning
    >
      <body className="flex min-h-full flex-col">
        <ThemeProvider>
          <SiteProvider settings={settings}>
            <AuthProvider>{children}</AuthProvider>
          </SiteProvider>
          <Toaster />
        </ThemeProvider>
      </body>
    </html>
  );
}
