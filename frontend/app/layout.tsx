import type { Metadata } from 'next';
import localFont from 'next/font/local';
import './globals.css';
import CountrySelector from '@/components/CountrySelector';
import LanguageToggle from '@/components/LanguageToggle';
import Navbar, { BrandMark } from '@/components/Navbar';
import { LanguageProvider } from '@/context/LanguageContext';

const geistSans = localFont({ src: './fonts/GeistVF.woff', variable: '--font-geist-sans', weight: '100 900' });
const geistMono = localFont({ src: './fonts/GeistMonoVF.woff', variable: '--font-geist-mono', weight: '100 900' });

export const metadata: Metadata = {
  title: 'Aerolíneas Rafael Pabón',
  description: 'Reservas, venta y gestión de vuelos con sincronización distribuida.',
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="es" className={`${geistSans.variable} ${geistMono.variable}`}>
      <body className="font-sans">
        <LanguageProvider>
          <div className="flex min-h-screen">
            <Navbar />
            <div className="flex min-w-0 flex-1 flex-col">
              <header className="sticky top-0 z-40 border-b border-slate-200 bg-white/90 backdrop-blur">
                <div className="flex items-center justify-between gap-3 px-4 py-3 sm:px-6 lg:px-8">
                  <div className="rounded-xl bg-navy-950 p-1.5 lg:hidden"><BrandMark compact /></div>
                  <p className="hidden text-sm text-slate-500 lg:block">Sistema de reservas distribuido</p>
                  <div className="flex items-center gap-2 sm:gap-3">
                    <CountrySelector />
                    <LanguageToggle />
                  </div>
                </div>
              </header>
              <main className="mx-auto w-full max-w-7xl flex-1 px-4 pb-28 pt-6 sm:px-6 lg:px-8 lg:pb-12">
                {children}
              </main>
            </div>
          </div>
        </LanguageProvider>
      </body>
    </html>
  );
}
