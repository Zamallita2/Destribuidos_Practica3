import type { Metadata } from 'next';
import './globals.css';
import CountrySelector from '@/components/CountrySelector';
import Navbar from '@/components/Navbar';
import { LanguageProvider } from '@/context/LanguageContext';

export const metadata: Metadata = {
  title: 'Aerolíneas Pabón | Panel de vuelos',
  description: 'Sistema de administración de aerolíneas con sincronización en tiempo real.',
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="es">
      <body className="font-sans">
        <LanguageProvider>
          <div className="flex h-screen overflow-hidden">
            {/* Sidebar Navbar */}
            <Navbar />
            
            {/* Main Content Area */}
            <main className="min-w-0 flex-1 overflow-y-auto relative flex flex-col">
              {/* Topbar */}
              <header className="sticky top-0 z-40 w-full glass-panel border-x-0 border-t-0 rounded-none px-4 md:px-6 py-4 flex items-center justify-between">
                <h1 className="text-lg md:text-2xl font-bold bg-clip-text text-transparent bg-gradient-to-r from-blue-400 to-purple-500">
                  Aerolíneas Pabón
                </h1>
                <div className="hidden md:flex items-center gap-4">
                  <CountrySelector />
                </div>
              </header>
              
              <div className="p-4 md:p-8 pb-20 w-full max-w-7xl mx-auto">
                {children}
              </div>
            </main>
          </div>
        </LanguageProvider>
      </body>
    </html>
  );
}
