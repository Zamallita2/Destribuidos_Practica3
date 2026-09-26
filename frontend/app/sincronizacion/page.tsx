"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { Activity, ArrowRight, Database, RefreshCw, ServerCrash } from "lucide-react";
import { useLanguage } from "@/context/LanguageContext";
import { translateUiText } from "@/lib/englishUi";

type NodeStatus = {
  state: "UP" | "DOWN" | "DEGRADED";
  flights: number | null;
  tickets: number | null;
  pending?: number | null;
  last_seen_at?: string;
  latency_ms: number;
};
type SyncEvent = { id: number; at: string; type: string; message: string };
type SyncStatus = {
  observed_at: string;
  nodes: Record<"pg_am" | "pg_eu" | "mongo", NodeStatus>;
  events: SyncEvent[];
  read_source: string;
  pending: number;
  count_difference: Record<string, number | null>;
};

const nodeNames = { pg_am: "PostgreSQL América", pg_eu: "PostgreSQL Europa/Asia", mongo: "MongoDB" };
const nodeKeys = ["pg_am", "pg_eu", "mongo"] as const;

function formatTime(value: string | undefined, language: "es" | "en") {
  return value ? new Date(value).toLocaleTimeString(language === "en" ? "en-US" : "es-BO") : "—";
}

function count(value: number | null | undefined, language: "es" | "en") {
  return value == null ? (language === "en" ? "No reading" : "Sin lectura") : value.toLocaleString(language === "en" ? "en-US" : "es-BO");
}

export default function SincronizacionPage() {
  const { language } = useLanguage();
  const [status, setStatus] = useState<SyncStatus | null>(null);
  const [error, setError] = useState("");
  const [selection, setSelection] = useState({ name: "Estados Unidos", region: "America" });

  useEffect(() => {
    let active = true;
    let selected = { name: "Estados Unidos", region: "America" };
    try {
      const saved = JSON.parse(window.localStorage.getItem("airres-country") || "null");
      if (saved && typeof saved.name === "string" && typeof saved.region === "string") selected = saved;
    } catch { /* El país por defecto sigue disponible. */ }
    setSelection(selected);
    async function refresh() {
      try {
        const response = await fetch("/api/sync/status", { cache: "no-store", headers: { "X-Region": selected.region } });
        if (!response.ok) throw new Error("El monitor no responde");
        const data: SyncStatus = await response.json();
        if (active) { setStatus(data); setError(""); }
      } catch {
        if (active) setError("La API no responde. El panel se actualizará cuando vuelva.");
      }
    }
    refresh();
    const timer = window.setInterval(refresh, 4000);
    return () => { active = false; window.clearInterval(timer); };
  }, []);

  const nodes = status?.nodes;
  const allUp = nodes && nodeKeys.every((key) => nodes[key].state === "UP");
  const source = status?.read_source;
  const sourceLabel = source === "pg_am" ? "PostgreSQL América" : source === "pg_eu" ? "PostgreSQL Europa/Asia" : source === "mongo_snapshot" ? "snapshot de MongoDB" : "ninguno";

  return <div className="space-y-6 text-white">
    <div className="flex flex-wrap items-start justify-between gap-4">
      <div>
        <div className="flex items-center gap-3"><Activity className="text-blue-400" /><h1 className="text-2xl font-bold">Sincronización de servidores</h1></div>
        <p className="mt-2 text-sm text-gray-400">Observación de conexiones reales. Detén o inicia un servidor y mira cómo cambia el sistema.</p>
      </div>
      <div className="rounded-lg border border-white/10 bg-white/5 px-3 py-2 text-xs text-gray-300">
        <RefreshCw className="mr-2 inline h-3 w-3" />Actualización automática · última lectura {formatTime(status?.observed_at, language)}
      </div>
    </div>

    {error && <div role="alert" className="rounded-xl border border-red-500/40 bg-red-500/10 p-4 text-red-200">{error}</div>}

    <div className={`rounded-xl border p-5 ${!status || error ? "border-white/10 bg-white/5" : allUp ? "border-green-500/30 bg-green-500/10" : "border-amber-500/30 bg-amber-500/10"}`}>
      <p className="text-lg font-semibold">{error ? "Sin observación actual" : !status ? "Consultando servidores…" : allUp ? "Los tres servidores responden" : "Funcionamiento con un nodo no disponible"}</p>
      <p className="mt-1 text-sm text-gray-300">País seleccionado: <strong>{language === "en" ? translateUiText(selection.name) : selection.name}</strong>. Fuente para consultar listas: <strong>{sourceLabel}</strong>.</p>
      <p className="mt-2 text-xs text-gray-400">{source === "mongo_snapshot" ? "El PostgreSQL principal no responde; MongoDB ofrece una copia global de lectura. Las reservas requieren PostgreSQL." : "La ruta cambia a MongoDB si el PostgreSQL principal deja de responder. Las reservas se guardan según el vuelo y su réplica."} El país elige la fuente de la lista; no filtra los vuelos por origen.</p>
    </div>

    <div className="grid gap-4 lg:grid-cols-3">
      {nodeKeys.map((key) => {
        const node = nodes?.[key];
        const online = node?.state === "UP";
        return <section key={key} className="rounded-xl border border-white/10 bg-[#202530] p-5">
          <div className="flex items-start justify-between gap-2">
            <div className="flex items-center gap-2">{online ? <Database className="h-5 w-5 text-blue-400" /> : <ServerCrash className="h-5 w-5 text-amber-400" />}<h2 className="font-semibold">{nodeNames[key]}</h2></div>
            <span className={`rounded-full px-2 py-1 text-xs font-bold ${online ? "bg-green-500/15 text-green-300" : node?.state === "DEGRADED" ? "bg-amber-500/15 text-amber-300" : "bg-red-500/15 text-red-300"}`}>{node?.state ?? "CARGANDO"}</span>
          </div>
          <div className="mt-5 grid grid-cols-2 gap-3">
            <div><p className="text-xs text-gray-400">Vuelos</p><p className="text-xl font-bold">{count(node?.flights, language)}</p></div>
            <div><p className="text-xs text-gray-400">Boletos</p><p className="text-xl font-bold">{count(node?.tickets, language)}</p></div>
          </div>
          <p className="mt-4 text-xs text-gray-400">{online ? (language === "en" ? `Responds in ${node.latency_ms} ms` : `Responde en ${node.latency_ms} ms`) : `Última vez activo: ${formatTime(node?.last_seen_at, language)}`}</p>
          {key !== "mongo" && <p className="mt-1 text-xs text-gray-400">Cola pendiente: <strong className="text-white">{count(node?.pending, language)}</strong></p>}
        </section>;
      })}
    </div>

    <div className="grid gap-4 lg:grid-cols-2">
      <section className="rounded-xl border border-white/10 bg-[#202530] p-5">
        <h2 className="font-semibold">Diferencias observables</h2>
        <p className="mt-1 text-xs text-gray-400">Comparación de cantidades. Una diferencia de cero no verifica que cada versión sea idéntica.</p>
        <div className="mt-4 space-y-3 text-sm">
          <div className="flex justify-between"><span>Vuelos AM ↔ EU</span><strong>{count(status?.count_difference.am_eu_flights, language)}</strong></div>
          <div className="flex justify-between"><span>Boletos AM ↔ EU</span><strong>{count(status?.count_difference.am_eu_tickets, language)}</strong></div>
          <div className="flex justify-between"><span>Vuelos AM ↔ Mongo</span><strong>{count(status?.count_difference.am_mongo_flights, language)}</strong></div>
          <div className="flex justify-between"><span>Vuelos EU ↔ Mongo</span><strong>{count(status?.count_difference.eu_mongo_flights, language)}</strong></div>
        </div>
        <div className="mt-5 rounded-lg bg-blue-500/10 p-3 text-sm">Operaciones esperando entrega completa: <strong>{count(status?.pending, language)}</strong></div>
        <p className="mt-2 text-xs text-gray-400">Si una PostgreSQL está caída, su cola local no puede consultarse y el total visible puede ser parcial.</p>
        <div className="mt-4 flex gap-4 text-sm text-blue-300"><Link href="/vuelos" className="flex items-center gap-1 hover:underline">Ver vuelos <ArrowRight size={14} /></Link><Link href="/boletos" className="flex items-center gap-1 hover:underline">Ver boletos <ArrowRight size={14} /></Link></div>
      </section>
      <section className="rounded-xl border border-white/10 bg-[#202530] p-5">
        <h2 className="font-semibold">Eventos detectados</h2>
        <p className="mt-1 text-xs text-gray-400">Horas locales. Se registran los cambios mientras la API está encendida.</p>
        <ol className="mt-4 max-h-80 space-y-3 overflow-y-auto">
          {[...(status?.events ?? [])].reverse().map((event) => <li key={event.id} className="border-l-2 border-blue-500/50 pl-3 text-sm"><time className="text-xs text-gray-400">{formatTime(event.at, language)}</time><p>{language === "en" ? translateUiText(event.message) : event.message}</p></li>)}
          {!status?.events?.length && <li className="text-sm text-gray-400">Esperando cambios de estado reales…</li>}
        </ol>
      </section>
    </div>
  </div>;
}
