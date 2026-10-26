import type { Metadata } from "next";
import { Inter, JetBrains_Mono, Noto_Sans_SC } from "next/font/google";
import { AuthProvider } from "@/components/common/auth/AuthProvider";
import { ThemeProvider } from "@/components/common/layout/ThemeProvider";
import { Toaster } from "@/components/ui/sonner";
import "./globals.css";

const inter = Inter({
  variable: "--font-inter",
  subsets: ["latin"],
  display: "swap",
});

const notoSansSC = Noto_Sans_SC({
  variable: "--font-noto-sans-sc",
  subsets: ["latin"],
  display: "swap",
  weight: ["400", "500", "600", "700"],
});

// 数字(价格、倒计时)用等宽字体, 跳动时不会左右晃
const jetbrainsMono = JetBrains_Mono({
  variable: "--font-jetbrains-mono",
  subsets: ["latin"],
  display: "swap",
});

export const metadata: Metadata = {
  title: {
    template: "%s - BlindCrate",
    default: "BlindCrate",
  },
  description: "盲盒开箱 · 保险柜竞拍对战平台",
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html
      lang="zh-CN"
      className={`${inter.variable} ${notoSansSC.variable} ${jetbrainsMono.variable} dark h-full font-sans antialiased`}
      suppressHydrationWarning
    >
      <body className="flex min-h-full flex-col">
        <ThemeProvider>
          <AuthProvider>{children}</AuthProvider>
          <Toaster />
        </ThemeProvider>
      </body>
    </html>
  );
}
