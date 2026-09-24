"use client";

import { Plane, LayoutDashboard, Ticket, Map, Settings } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { motion } from "framer-motion";
import { useLanguage } from "@/context/LanguageContext";

export default function Navbar() {
  const pathname = usePathname();
  const { language, setLanguage, t } = useLanguage();

  const links = [
    { href: "/", label: t("nav.dashboard"), icon: LayoutDashboard },
    { href: "/vuelos", label: t("nav.flights"), icon: Plane },
    { href: "/boletos", label: t("nav.tickets"), icon: Ticket },
    { href: "/sugerencias", label: t("nav.routes"), icon: Map },
    { href: "/tsp", label: t("nav.tsp"), icon: Map },
    { href: "/configuracion", label: t("nav.settings"), icon: Settings },
  ];

  return (
    <nav className="w-64 glass-panel border-y-0 border-l-0 rounded-none h-full flex flex-col p-4 relative z-50">
      <div className="flex items-center gap-3 mb-10 px-2 mt-4 animate-float">
        <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-blue-500 to-purple-600 flex items-center justify-center shadow-[0_0_15px_rgba(59,130,246,0.5)]">
          <Plane className="text-white w-6 h-6" />
        </div>
        <span className="text-xl font-bold font-heading tracking-wide text-transparent bg-clip-text bg-gradient-to-r from-white to-gray-400">
          AirRes
        </span>
      </div>

      <div className="flex flex-col gap-2 flex-1 relative">
        {links.map((link) => {
          const Icon = link.icon;
          const isActive = pathname === link.href;

          return (
            <Link
              key={link.href}
              href={link.href}
              className={`relative px-4 py-3 rounded-xl flex items-center gap-3 transition-all duration-300 ${
                isActive ? "text-white" : "text-gray-400 hover:text-white hover:bg-white/5"
              }`}
            >
              {isActive && (
                <motion.div
                  layoutId="activeTab"
                  className="absolute inset-0 bg-gradient-to-r from-blue-500/20 to-purple-500/20 border border-blue-500/30 rounded-xl"
                  transition={{ type: "spring", stiffness: 300, damping: 30 }}
                />
              )}
              <Icon className={`w-5 h-5 relative z-10 ${isActive ? "text-blue-400" : ""}`} />
              <span className="font-medium relative z-10">{link.label}</span>
            </Link>
          );
        })}
      </div>

      <div className="mt-auto pt-4 px-2">
        <div className="flex gap-2 mb-6 bg-black/40 p-1 rounded-lg border border-white/5">
          <button
            onClick={() => setLanguage("es")}
            className={`flex-1 py-1 text-xs font-bold rounded-md transition-all ${
              language === "es" ? "bg-blue-600 text-white" : "text-gray-400 hover:text-white"
            }`}
          >
            ES
          </button>
          <button
            onClick={() => setLanguage("en")}
            className={`flex-1 py-1 text-xs font-bold rounded-md transition-all ${
              language === "en" ? "bg-blue-600 text-white" : "text-gray-400 hover:text-white"
            }`}
          >
            EN
          </button>
        </div>

        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-full bg-gray-800 border border-gray-700 flex items-center justify-center overflow-hidden">
             <img src="https://api.dicebear.com/7.x/avataaars/svg?seed=Admin" alt="Admin" className="w-full h-full object-cover" />
          </div>
          <div>
            <p className="text-sm font-semibold">Admin Panel</p>
            <p className="text-xs text-gray-500">v1.0.0</p>
          </div>
        </div>
      </div>
    </nav>
  );
}
