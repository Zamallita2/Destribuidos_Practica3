"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { Plane, Users, CalendarSync, Banknote, Database, ArrowRight, Ticket, Map as MapIcon, Server } from "lucide-react";
import FlightMap from "@/components/FlightMap";
import { formatFlightLocalTime } from "@/lib/flightTime";
import { useLanguage } from "@/context/LanguageContext";
import { FlightStatusBadge, KpiCard } from "@/components/ui";

type City = { id: number; codigo: string; pais: string; time_zone: string };
type Flight = { id: number; id_origen: number; id_destino: number; id_estado_vuelo: number; salida_programada: number; llegada_programada: number };
type Departure = { flight: Flight; origin?: City; destination?: City; available: number | null };
type Summary = { vuelos: number; manifiestos_generados: number; vendidos: number; reservados: number; ingresos_primera: number; ingresos_turistica: number };
type Health = { nodes: { postgres_america: boolean; postgres_europa_asia: boolean; mongodb: boolean } };

const integer = (value: number) => value.toLocaleString("es-BO");
const money = (value: number) => `$${Math.round(value).toLocaleString("es-BO")}`;

export default function Dashboard() {
  const { t, language } = useLanguage();
  const [summary, setSummary] = useState<Summary | null>(null);
  const [departures, setDepartures] = useState<Departure[]>([]);
  const [health, setHealth] = useState<Health | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);

  useEffect(() => {
    let active = true;
    const load = async () => {
      try {
        const country = JSON.parse(localStorage.getItem("airres-country") || "{}");
        const headers = { "X-User-Country": country.name || "Estados Unidos", "X-Region": country.region || "America" };
        const [cityResponse, flightResponse, summaryResponse, healthResponse] = await Promise.all([
          fetch("/api/ciudades", { headers }), fetch("/api/vuelos?limit=10", { headers }),
          fetch("/api/dashboard", { headers }), fetch("/api/health"),
        ]);
        if (!cityResponse.ok || !flightResponse.ok || !summaryResponse.ok || !healthResponse.ok) throw new Error("Panel no disponible");
        const [cities, flights, nextSummary, nextHealth]: [City[], Flight[], Summary, Health] = await Promise.all([
          cityResponse.json(), flightResponse.json(), summaryResponse.json(), healthResponse.json(),
        ]);
        const availability = await Promise.all(flights.map(async (flight) => {
          try {
            const response = await fetch(`/api/dashboard/vuelos/${flight.id}`, { headers });
            return response.ok ? (await response.json()).disponibles as number : null;
          } catch { return null; }
        }));
        if (!active) return;
        setSummary(nextSummary);
        setHealth(nextHealth);
        setDepartures(flights.map((flight, index) => ({
          flight, origin: cities.find((city) => city.id === flight.id_origen),
          destination: cities.find((city) => city.id === flight.id_destino), available: availability[index],
        })));
        setError(false);
      } catch { if (active) setError(true); }
      finally { if (active) setLoading(false); }
    };
    load();
    const interval = setInterval(load, 60000);
    return () => { active = false; clearInterval(interval); };
  }, []);

  const stats = [
    { label: "Vuelos registrados", value: summary ? integer(summary.vuelos) : "—", icon: Plane, note: "CSV histórico y vuelos de demostración", tone: "navy" as const },
    { label: "Vuelos con asientos cargados", value: summary ? integer(summary.manifiestos_generados) : "—", icon: Database, note: "Manifiestos generados", tone: "navy" as const },
    { label: t("dashboard.stats.sold"), value: summary ? integer(summary.vendidos) : "—", icon: Users, note: "Asientos vendidos en todos los vuelos", tone: "green" as const },
    { label: t("dashboard.stats.reserved"), value: summary ? integer(summary.reservados) : "—", icon: CalendarSync, note: "Asientos reservados", tone: "amber" as const },
    { label: t("dashboard.stats.income_first"), value: summary ? money(summary.ingresos_primera) : "—", icon: Banknote, note: "Estimación del modelo", tone: "gold" as const },
    { label: t("dashboard.stats.income_regular"), value: summary ? money(summary.ingresos_turistica) : "—", icon: Banknote, note: "Estimación del modelo", tone: "gold" as const },
  ];
  const nodes = [
    { name: "PostgreSQL América", online: health?.nodes.postgres_america },
    { name: "PostgreSQL Europa/Asia", online: health?.nodes.postgres_europa_asia },
    { name: "MongoDB", online: health?.nodes.mongodb },
  ];
  const quickLinks = [
    { href: "/boletos", label: "Comprar un boleto", icon: Ticket },
    { href: "/vuelos", label: "Ver todos los vuelos", icon: Plane },
    { href: "/sugerencias", label: "Planificar una ruta", icon: MapIcon },
  ];
  const boardColumns = "grid-cols-[4rem_minmax(0,1fr)_auto] sm:grid-cols-[4rem_8.5rem_minmax(0,1fr)_3.5rem_7.5rem]";
  const time = (epoch: number, zone?: string) => new Date(epoch * 1000).toLocaleTimeString(language === "en" ? "en-US" : "es-BO", { hour: "2-digit", minute: "2-digit", hourCycle: "h23", timeZone: zone || "UTC" });

  return (
    <div className="fade-up space-y-6">
      <section className="relative overflow-hidden rounded-3xl bg-navy-900 px-6 py-8 text-white shadow-lift sm:px-10 sm:py-10">
        <div aria-hidden="true" className="pointer-events-none absolute -right-16 -top-24 h-72 w-72 rounded-full bg-gold-400/20 blur-3xl" />
        <div aria-hidden="true" className="pointer-events-none absolute -bottom-24 left-1/3 h-64 w-64 rounded-full bg-navy-500/30 blur-3xl" />
        <Plane aria-hidden="true" className="pointer-events-none absolute right-8 top-8 hidden h-28 w-28 -rotate-12 text-white/5 md:block" />
        <div className="relative">
          <p className="text-xs font-semibold uppercase tracking-[0.2em] text-gold-300">Aerolíneas Rafael Pabón</p>
          <h1 className="mt-2 text-3xl font-bold text-white sm:text-4xl">Panel de la aerolínea</h1>
          <p className="mt-2 max-w-2xl text-navy-100">Resumen de los vuelos cargados y las próximas salidas.</p>
          <div className="mt-6 flex flex-wrap gap-3">
            {quickLinks.map(({ href, label, icon: Icon }, index) => <Link key={href} href={href} className={index === 0 ? "btn-gold" : "btn border border-white/20 bg-white/10 text-white hover:bg-white/20"}>
              <Icon className="h-4 w-4" aria-hidden="true" />{label}
            </Link>)}
          </div>
        </div>
      </section>

      {error && <p role="alert" className="alert alert-warning">No se pudo actualizar el panel. Los datos mostrados pueden estar desactualizados.</p>}

      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
        {stats.map((stat) => <KpiCard key={stat.label} {...stat} />)}
      </div>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,2fr)_minmax(260px,1fr)]">
        <section className="min-w-0">
          <div className="mb-3 flex flex-wrap items-end justify-between gap-3">
            <div><h2 className="section-title">Próximas salidas</h2><p className="section-subtitle">Hasta 10 vuelos futuros. El catálogo completo incluye los vuelos históricos.</p></div>
            <Link href="/vuelos" className="btn-ghost btn-sm">Abrir catálogo <ArrowRight className="h-4 w-4" aria-hidden="true" /></Link>
          </div>
          <div className="fids">
            <div className={`fids-head ${boardColumns}`}>
              <span>{t("dashboard.table.time")}</span>
              <span className="hidden sm:block">{t("dashboard.table.flight")}</span>
              <span>{t("dashboard.table.destination")}</span>
              <span className="hidden text-right sm:block">{t("dashboard.table.avail")}</span>
              <span className="text-right">{t("dashboard.table.remarks")}</span>
            </div>
            {loading && <p className="px-5 py-10 text-center text-navy-200">Cargando salidas…</p>}
            {!loading && departures.length === 0 && <p className="px-5 py-10 text-center text-navy-200">No hay salidas futuras. Los vuelos históricos están en el catálogo.</p>}
            {departures.map(({ flight, origin, destination, available }) => <Link key={flight.id} href={`/dashboard/vuelos/${flight.id}`} className={`fids-row ${boardColumns} transition hover:bg-white/5`}>
              <span className="text-lg font-semibold text-gold-300">{time(flight.salida_programada, origin?.time_zone)}</span>
              <span className="hidden whitespace-nowrap text-navy-100 sm:block">AP {flight.id}</span>
              <span className="min-w-0">
                <span className="block truncate font-sans text-base font-semibold text-white">{origin?.codigo || "?"} → {destination?.codigo || "?"}</span>
                <span className="block truncate font-sans text-xs text-navy-300">{destination?.pais || "Destino"} · {formatFlightLocalTime(flight.salida_programada, origin?.time_zone || "UTC", language)}</span>
              </span>
              <span className="hidden text-right text-navy-100 sm:block">{available === null ? "—" : integer(available)}</span>
              <span className="text-right font-sans"><FlightStatusBadge state={flight.id_estado_vuelo} /></span>
            </Link>)}
          </div>
        </section>

        <section className="card card-body h-max">
          <div className="flex items-center gap-2"><Server className="h-5 w-5 text-navy-600" aria-hidden="true" /><h2 className="section-title">Estado de servidores</h2></div>
          <p className="section-subtitle">Indica si cada base de datos responde. No representa el avance de la sincronización.</p>
          <ul className="mt-5 space-y-2.5">{nodes.map((node) => <li key={node.name} className="flex items-center justify-between gap-3 rounded-xl border border-slate-200 bg-slate-50 px-3.5 py-3">
            <span className="flex items-center gap-2.5 text-sm font-medium text-navy-900">
              <span className={`h-2.5 w-2.5 rounded-full ${node.online === undefined ? "bg-slate-300" : node.online ? "bg-emerald-500 shadow-[0_0_0_3px_rgba(16,185,129,0.2)]" : "bg-red-500 shadow-[0_0_0_3px_rgba(239,68,68,0.2)]"}`} aria-hidden="true" />
              {node.name}
            </span>
            <span className={`badge ${node.online === undefined ? "badge-slate" : node.online ? "badge-green" : "badge-red"}`}>{node.online === undefined ? "Consultando" : node.online ? "Disponible" : "Sin conexión"}</span>
          </li>)}</ul>
          <Link href="/sincronizacion" className="btn-secondary mt-5 w-full">Ver sincronización <ArrowRight className="h-4 w-4" aria-hidden="true" /></Link>
        </section>
      </div>

      <section>
        <h2 className="section-title">Mapa de rutas próximas</h2>
        <p className="section-subtitle mb-4">Vista ilustrativa de los próximos vuelos consultados; no muestra los 29.000 registros a la vez.</p>
        <FlightMap />
      </section>
    </div>
  );
}
