"use client";

import { Plane, LayoutDashboard, Ticket, Map, Settings, FileText, DatabaseZap, Activity, Route } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useLanguage } from "@/context/LanguageContext";

export function BrandMark({ compact = false }: { compact?: boolean }) {
  return <Link href="/" className="flex items-center gap-3">
    <span className="flex h-10 w-10 items-center justify-center rounded-xl bg-gold-400 text-navy-950 shadow-lg">
      <Plane className="h-5 w-5 -rotate-45" aria-hidden="true" />
    </span>
    {!compact && <span className="leading-tight">
      <span className="block text-[11px] font-semibold uppercase tracking-[0.2em] text-gold-300">Aerolíneas</span>
      <span className="block text-lg font-bold text-white">Rafael Pabón</span>
    </span>}
  </Link>;
}

export default function Navbar() {
  const pathname = usePathname();
  const { t } = useLanguage();

  const groups = [
    { title: "Viajar", links: [
      { href: "/", label: t("nav.dashboard"), icon: LayoutDashboard },
      { href: "/vuelos", label: t("nav.flights"), icon: Plane },
      { href: "/boletos", label: t("nav.tickets"), icon: Ticket },
      { href: "/gestion-boletos", label: t("nav.ticketManagement"), icon: FileText },
    ] },
    { title: "Rutas", links: [
      { href: "/sugerencias", label: t("nav.routes"), icon: Map },
      { href: "/tsp", label: t("nav.tsp"), icon: Route },
    ] },
    { title: "Operación", links: [
      { href: "/sincronizacion", label: t("nav.sync"), icon: Activity },
      { href: "/entradas", label: t("nav.inputs"), icon: DatabaseZap },
      { href: "/configuracion", label: t("nav.settings"), icon: Settings },
    ] },
  ];
  const isActive = (href: string) => href === "/" ? pathname === "/" : pathname.startsWith(href);

  return <>
    <nav aria-label="Navegación principal" className="sticky top-0 hidden h-screen w-64 shrink-0 flex-col bg-navy-950 px-4 py-6 text-white lg:flex">
      <div className="px-2"><BrandMark /></div>
      <div className="mt-8 flex-1 space-y-6 overflow-y-auto">
        {groups.map((group) => <div key={group.title}>
          <p className="px-3 text-[11px] font-semibold uppercase tracking-[0.18em] text-navy-300">{group.title}</p>
          <ul className="mt-2 space-y-1">
            {group.links.map(({ href, label, icon: Icon }) => {
              const active = isActive(href);
              return <li key={href}>
                <Link href={href} aria-current={active ? "page" : undefined}
                  className={`flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium transition ${active
                    ? "bg-white/10 text-white shadow-inner ring-1 ring-white/10"
                    : "text-navy-200 hover:bg-white/5 hover:text-white"}`}>
                  <Icon className={`h-[18px] w-[18px] ${active ? "text-gold-300" : ""}`} aria-hidden="true" />
                  {label}
                </Link>
              </li>;
            })}
          </ul>
        </div>)}
      </div>
      <div className="mt-6 rounded-xl border border-white/10 bg-white/5 p-3 text-xs text-navy-200">
        <p className="font-semibold text-white">Sistema distribuido</p>
        <p className="mt-0.5">3 servidores · 3 bases de datos</p>
      </div>
    </nav>

    <nav aria-label="Navegación móvil" className="fixed inset-x-0 bottom-0 z-50 flex gap-1 overflow-x-auto border-t border-slate-200 bg-white/95 p-2 backdrop-blur lg:hidden">
      {groups.flatMap((group) => group.links).map(({ href, label, icon: Icon }) => {
        const active = isActive(href);
        return <Link key={href} href={href} aria-current={active ? "page" : undefined}
          className={`flex min-w-[76px] flex-col items-center gap-1 rounded-lg px-2 py-1.5 text-[10px] font-medium ${active ? "bg-navy-50 text-navy-800" : "text-slate-500"}`}>
          <Icon className="h-5 w-5" aria-hidden="true" /><span className="whitespace-nowrap">{label}</span>
        </Link>;
      })}
    </nav>
  </>;
}
