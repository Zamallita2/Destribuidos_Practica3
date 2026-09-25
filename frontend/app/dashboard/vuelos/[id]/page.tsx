"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";

type Stats = {
  capacidad: number;
  vendidos: number;
  reservados: number;
  disponibles: number;
  ingresos_primera: number;
  ingresos_turistica: number;
};

export default function FlightDashboard() {
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

  if (error) return <div className="p-8 text-red-400">{error}</div>;
  if (!flight || !stats) return <div className="p-8 text-gray-400">Cargando panel del vuelo…</div>;

  const origin = cities.find((city) => city.id === flight.id_origen)?.codigo || flight.id_origen;
  const destination = cities.find((city) => city.id === flight.id_destino)?.codigo || flight.id_destino;
  const cards = [
    ["Capacidad", stats.capacidad],
    ["Vendidos", stats.vendidos],
    ["Reservados", stats.reservados],
    ["Disponibles", stats.disponibles],
    ["Ingresos Primera", `$${stats.ingresos_primera.toFixed(2)}`],
    ["Ingresos Turística", `$${stats.ingresos_turistica.toFixed(2)}`],
  ];
  return (
    <main className="space-y-8 p-8">
      <Link href="/vuelos" className="text-blue-400 hover:underline">← Volver a vuelos</Link>
      <div>
        <h1 className="text-3xl font-bold">Panel del vuelo AP {id}</h1>
        <p className="text-gray-400">{origin} → {destination} · {new Date(flight.salida_programada * 1000).toLocaleString()} · Estado {flight.id_estado_vuelo}</p>
      </div>
      <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
        {cards.map(([label, value]) => (
          <div key={label} className="glass-card p-6">
            <p className="text-sm text-gray-400">{label}</p>
            <p className="mt-2 text-3xl font-semibold">{value}</p>
          </div>
        ))}
      </div>
    </main>
  );
}
