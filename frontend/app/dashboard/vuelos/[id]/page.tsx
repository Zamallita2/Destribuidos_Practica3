"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import { formatFlightLocalTime } from "@/lib/flightTime";
import { useLanguage } from "@/context/LanguageContext";
import { ArrowLeft, Armchair, Banknote, CalendarSync, Loader2, Plane, Ticket, Users } from "lucide-react";
import { FlightStatusBadge, KpiCard } from "@/components/ui";

type Stats = {
  capacidad: number;
  vendidos: number;
  reservados: number;
  disponibles: number;
  ingresos_primera: number;
  ingresos_turistica: number;
};

export default function FlightDashboard() {
  const { language } = useLanguage();
  const params = useParams();
  const id = String(params.id);
  const [flight, setFlight] = useState<any>(null);
  const [stats, setStats] = useState<Stats | null>(null);
  const [cities, setCities] = useState<any[]>([]);
  const [error, setError] = useState("");

  useEffect(() => {
    const headers = {
      "X-User-Country": JSON.parse(localStorage.getItem("airres-country") || "{}").name || "Estados Unidos",
    };
    Promise.all([
      fetch(`/api/vuelos/${id}`, { headers }),
      fetch(`/api/dashboard/vuelos/${id}`, { headers }),
      fetch("/api/ciudades", { headers }),
    ]).then(async ([flightResponse, statsResponse, citiesResponse]) => {
      if (!flightResponse.ok || !statsResponse.ok || !citiesResponse.ok) {
        throw new Error("No se pudo cargar el panel del vuelo.");
      }
      setFlight(await flightResponse.json());
      setStats(await statsResponse.json());
      setCities(await citiesResponse.json());
    }).catch((reason) => setError(String(reason)));
  }, [id]);

  if (error) return <div role="alert" className="alert alert-error">{error}</div>;
  if (!flight || !stats) return <div className="flex h-[50vh] items-center justify-center gap-3 text-slate-500"><Loader2 className="h-6 w-6 animate-spin text-navy-500" />Cargando panel del vuelo…</div>;

  const originCity = cities.find((city) => city.id === flight.id_origen);
  const destinationCity = cities.find((city) => city.id === flight.id_destino);
  const origin = originCity?.codigo || flight.id_origen;
  const destination = destinationCity?.codigo || flight.id_destino;
  const capacity = stats.capacidad || 1;
  const percent = (value: number) => Math.round((value / capacity) * 1000) / 10;
  const money = (value: number) => `$${value.toLocaleString("es-BO", { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
  const occupancy = [
    { label: "Vendidos", value: stats.vendidos, color: "bg-navy-800" },
    { label: "Reservados", value: stats.reservados, color: "bg-gold-400" },
    { label: "Disponibles", value: stats.disponibles, color: "bg-slate-200" },
  ];
  return (
    <div className="fade-up space-y-6">
      <Link href="/vuelos" className="btn-ghost btn-sm -ml-2"><ArrowLeft className="h-4 w-4" /> Volver a vuelos</Link>

      <section className="overflow-hidden rounded-3xl bg-navy-900 text-white shadow-lift">
        <div className="flex flex-wrap items-start justify-between gap-4 p-6 sm:p-8">
          <div>
            <p className="text-xs font-semibold uppercase tracking-[0.18em] text-gold-300">Panel del vuelo AP {id}</p>
            <div className="mt-3 flex items-center gap-4">
              <div><p className="text-4xl font-bold text-white sm:text-5xl">{origin}</p><p className="text-sm text-navy-200">{originCity?.pais}</p></div>
              <Plane className="h-8 w-8 rotate-45 text-gold-300" aria-hidden="true" />
              <div><p className="text-4xl font-bold text-white sm:text-5xl">{destination}</p><p className="text-sm text-navy-200">{destinationCity?.pais}</p></div>
            </div>
          </div>
          <div className="rounded-full bg-white p-0.5"><FlightStatusBadge state={flight.id_estado_vuelo} /></div>
        </div>
        <div className="grid gap-px bg-white/10 sm:grid-cols-2">
          <div className="bg-navy-900 px-6 py-4 sm:px-8"><p className="text-xs text-navy-300">Sale de {origin} (hora local)</p><p className="font-semibold text-white">{formatFlightLocalTime(flight.salida_programada, originCity?.time_zone || "UTC", language)}</p></div>
          <div className="bg-navy-900 px-6 py-4 sm:px-8"><p className="text-xs text-navy-300">Llega a {destination} (hora local)</p><p className="font-semibold text-white">{formatFlightLocalTime(flight.llegada_programada, destinationCity?.time_zone || "UTC", language)}</p></div>
        </div>
      </section>

      <section className="card card-body">
        <div className="flex flex-wrap items-end justify-between gap-2">
          <div><h2 className="section-title">Ocupación de la cabina</h2><p className="section-subtitle">Capacidad: {stats.capacidad} asientos</p></div>
          <p className="text-3xl font-bold text-navy-900">{percent(stats.vendidos + stats.reservados)}%</p>
        </div>
        <div className="mt-4 flex h-4 overflow-hidden rounded-full bg-slate-100" role="img" aria-label={`${stats.vendidos} vendidos, ${stats.reservados} reservados, ${stats.disponibles} disponibles`}>
          {occupancy.map((part) => <div key={part.label} className={part.color} style={{ width: `${percent(part.value)}%` }} />)}
        </div>
        <ul className="mt-3 flex flex-wrap gap-x-6 gap-y-1 text-sm text-slate-600">
          {occupancy.map((part) => <li key={part.label} className="flex items-center gap-2"><span className={`h-3 w-3 rounded-sm ${part.color}`} />{part.label}: <strong className="text-navy-900">{part.value}</strong> ({percent(part.value)}%)</li>)}
        </ul>
      </section>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
        <KpiCard icon={Armchair} label="Capacidad" value={stats.capacidad} />
        <KpiCard icon={Users} label="Vendidos" value={stats.vendidos} tone="green" />
        <KpiCard icon={CalendarSync} label="Reservados" value={stats.reservados} tone="amber" />
        <KpiCard icon={Ticket} label="Disponibles" value={stats.disponibles} />
        <KpiCard icon={Banknote} label="Ingresos Primera" value={money(stats.ingresos_primera)} tone="gold" />
        <KpiCard icon={Banknote} label="Ingresos Turística" value={money(stats.ingresos_turistica)} tone="gold" />
      </div>

      <div className="flex flex-wrap gap-2">
        <Link href={`/boletos?vuelo=${id}`} className="btn-gold"><Ticket className="h-4 w-4" /> Comprar en este vuelo</Link>
      </div>
    </div>
  );
}
