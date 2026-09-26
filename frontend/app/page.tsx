"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { Plane, Users, CalendarSync, Banknote, Database, Armchair, ArrowRight } from "lucide-react";
import FlightMap from "@/components/FlightMap";
import { formatFlightLocalTime } from "@/lib/flightTime";
import { useLanguage } from "@/context/LanguageContext";

type City = { id: number; codigo: string; pais: string; time_zone: string };
type Flight = { id: number; id_origen: number; id_destino: number; id_estado_vuelo: number; salida_programada: number; llegada_programada: number };
type Departure = { flight: Flight; origin?: City; destination?: City; available: number | null };
type Summary = { vuelos: number; manifiestos_generados: number; vendidos: number; reservados: number; ingresos_primera: number; ingresos_turistica: number };
type Health = { nodes: { postgres_america: boolean; postgres_europa_asia: boolean; mongodb: boolean } };

const stateNames: Record<number, string> = { 1: "Programado", 2: "Embarcando", 3: "Despegó", 4: "En vuelo", 5: "Aterrizó", 6: "Llegó", 7: "Cancelado", 8: "Retrasado" };
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
    { label: "Vuelos registrados", value: summary ? integer(summary.vuelos) : "—", icon: Plane, note: "CSV histórico y vuelos de demostración" },
    { label: "Vuelos con asientos cargados", value: summary ? integer(summary.manifiestos_generados) : "—", icon: Database, note: "Manifiestos generados" },
    { label: t("dashboard.stats.sold"), value: summary ? integer(summary.vendidos) : "—", icon: Users, note: "Asientos vendidos en todos los vuelos" },
    { label: t("dashboard.stats.reserved"), value: summary ? integer(summary.reservados) : "—", icon: CalendarSync, note: "Asientos reservados" },
    { label: t("dashboard.stats.income_first"), value: summary ? money(summary.ingresos_primera) : "—", icon: Banknote, note: "Estimación del modelo" },
    { label: t("dashboard.stats.income_regular"), value: summary ? money(summary.ingresos_turistica) : "—", icon: Banknote, note: "Estimación del modelo" },
  ];
  const nodes = [
    { name: "PostgreSQL América", online: health?.nodes.postgres_america },
    { name: "PostgreSQL Europa/Asia", online: health?.nodes.postgres_europa_asia },
    { name: "MongoDB", online: health?.nodes.mongodb },
  ];

  return (
    <div className="space-y-8 pb-12">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div><h2 className="text-3xl font-bold text-white">Panel de la aerolínea</h2><p className="mt-2 text-gray-400">Resumen de los vuelos cargados y las próximas salidas.</p></div>
        <Link href="/vuelos" className="inline-flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2 font-semibold text-white hover:bg-blue-500">Ver todos los vuelos <ArrowRight className="h-4 w-4" /></Link>
      </div>
      {error && <p role="alert" className="rounded-xl border border-amber-500/30 bg-amber-500/10 p-4 text-sm text-amber-200">No se pudo actualizar el panel. Los datos mostrados pueden estar desactualizados.</p>}
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
        {stats.map(({ label, value, icon: Icon, note }) => <div key={label} className="glass-card p-5">
          <Icon className="mb-3 h-6 w-6 text-blue-300" aria-hidden="true" /><p className="text-2xl font-bold text-white">{value}</p>
          <p className="mt-1 text-sm font-semibold text-gray-200">{label}</p><p className="mt-1 text-xs text-gray-400">{note}</p>
        </div>)}
      </div>
      <div className="grid gap-6 xl:grid-cols-[minmax(0,2fr)_minmax(260px,1fr)]">
        <section className="glass-panel min-w-0 p-5 sm:p-6">
          <div className="mb-5 flex flex-wrap items-start justify-between gap-3">
            <div><h3 className="text-xl font-bold text-white">Próximas salidas</h3><p className="mt-1 text-sm text-gray-400">Hasta 10 vuelos futuros. El catálogo completo incluye los vuelos históricos.</p></div>
            <Link href="/vuelos" className="text-sm font-semibold text-blue-300 hover:underline">Abrir catálogo</Link>
          </div>
          <div className="space-y-2">
            {loading && <p className="py-10 text-center text-gray-400">Cargando salidas…</p>}
            {!loading && departures.length === 0 && <p className="py-10 text-center text-gray-400">No hay salidas futuras. Los vuelos históricos están en el catálogo.</p>}
            {departures.map(({ flight, origin, destination, available }) => <div key={flight.id} className="grid gap-3 rounded-xl border border-white/10 bg-white/[0.03] p-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.2fr)_auto] lg:items-center">
              <div><p className="text-xs text-gray-400">Vuelo AP {flight.id}</p><p className="font-semibold text-white">{origin?.codigo || "?"} → {destination?.codigo || "?"}</p><p className="text-xs text-gray-400">{origin?.pais || "Origen"} → {destination?.pais || "Destino"}</p></div>
              <div className="text-sm text-gray-200"><p><span className="text-gray-400">Sale:</span> {formatFlightLocalTime(flight.salida_programada, origin?.time_zone || "UTC", language)}</p><p><span className="text-gray-400">Llega:</span> {formatFlightLocalTime(flight.llegada_programada, destination?.time_zone || "UTC", language)}</p></div>
              <div className="text-sm lg:text-right"><p className="font-medium text-blue-200"><Armchair className="mr-1 inline h-4 w-4" aria-hidden="true" />{available === null ? "Cupos sin consultar" : `${integer(available)} asientos disponibles`}</p><p className="text-gray-400">{stateNames[flight.id_estado_vuelo] || "Sin estado"}</p></div>
            </div>)}
          </div>
        </section>
        <section className="glass-panel p-5 sm:p-6">
          <h3 className="text-xl font-bold text-white">Estado de servidores</h3>
          <p className="mt-1 text-sm text-gray-400">Indica si cada base de datos responde. No representa el avance de la sincronización.</p>
          <div className="mt-5 space-y-3">{nodes.map((node) => <div key={node.name} className="flex items-center justify-between gap-3 rounded-lg border border-white/10 bg-white/5 p-3">
            <span className="text-sm text-gray-200">{node.name}</span>
            <span className={`rounded-md px-2 py-1 text-xs font-semibold ${node.online === undefined ? "bg-gray-500/20 text-gray-300" : node.online ? "bg-emerald-500/15 text-emerald-300" : "bg-red-500/15 text-red-300"}`}>{node.online === undefined ? "Consultando" : node.online ? "Disponible" : "Sin conexión"}</span>
          </div>)}</div>
        </section>
      </div>
      <section><h3 className="mb-2 text-xl font-bold text-white">Mapa de rutas próximas</h3><p className="mb-4 text-sm text-gray-400">Vista ilustrativa de los próximos vuelos consultados; no muestra los 29.000 registros a la vez.</p><FlightMap /></section>
    </div>
  );
}
