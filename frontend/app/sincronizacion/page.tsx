"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { Activity, ArrowRight, Database, RefreshCw, ServerCrash, CheckCircle2, AlertTriangle, Clock, Server, Crown } from "lucide-react";
import { PageHeader } from "@/components/ui";
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

type ClusterNode = {
  node_id: string; up: boolean; primary: boolean; time_zone: string; local_time: string;
  lamport_clock: number; vector_clock: Record<string, number> | null; latency_ms: number;
};
const serverNames: Record<string, { name: string; city: string }> = {
  america: { name: "Servidor América", city: "La Paz" },
  europa: { name: "Servidor Europa", city: "Berlín" },
  asia: { name: "Servidor Asia", city: "Pekín" },
};
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
  const [cluster, setCluster] = useState<{ served_by: string; nodes: ClusterNode[] } | null>(null);

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
        const clusterResponse = await fetch("/api/cluster", { cache: "no-store", headers: { "X-Region": selected.region } });
        if (clusterResponse.ok && active) setCluster(await clusterResponse.json());
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

  const healthy = !error && status && allUp;
  const stateStyle = (state?: string) => state === "UP"
    ? { dot: "bg-emerald-500", ring: "border-emerald-300 bg-emerald-50", badge: "badge-green" }
    : state === "DEGRADED" ? { dot: "bg-amber-500", ring: "border-amber-300 bg-amber-50", badge: "badge-amber" }
    : state ? { dot: "bg-red-500", ring: "border-red-300 bg-red-50", badge: "badge-red" }
    : { dot: "bg-slate-300", ring: "border-slate-200 bg-slate-50", badge: "badge-slate" };

  return <div className="fade-up space-y-6">
    <PageHeader icon={Activity} eyebrow="Operación" title="Sincronización de servidores"
      subtitle="Observación de conexiones reales. Detén o inicia un servidor y mira cómo cambia el sistema."
      actions={<span className="badge badge-slate py-1.5"><RefreshCw className="h-3 w-3 animate-spin [animation-duration:4s]" aria-hidden="true" />Actualización automática · última lectura {formatTime(status?.observed_at, language)}</span>} />

    {error && <div role="alert" className="alert alert-error">{error}</div>}

    <section className={`alert ${!status || error ? "alert-info" : healthy ? "alert-success" : "alert-warning"} p-5`}>
      {healthy ? <CheckCircle2 className="h-6 w-6 shrink-0" /> : <AlertTriangle className="h-6 w-6 shrink-0" />}
      <div>
        <p className="text-base font-semibold">{error ? "Sin observación actual" : !status ? "Consultando servidores…" : allUp ? "Los tres servidores responden" : "Funcionamiento con un nodo no disponible"}</p>
        <p className="mt-1">País seleccionado: <strong>{language === "en" ? translateUiText(selection.name) : selection.name}</strong>. Fuente para consultar listas: <strong>{sourceLabel}</strong>.</p>
        <p className="mt-1 text-xs opacity-80">{source === "mongo_snapshot" ? "El PostgreSQL principal no responde; MongoDB ofrece una copia global de lectura. Las reservas requieren PostgreSQL." : "La ruta cambia a MongoDB si el PostgreSQL principal deja de responder. Las reservas se guardan según el vuelo y su réplica."} El país elige la fuente de la lista; no filtra los vuelos por origen.</p>
      </div>
    </section>

    <section className="card card-body">
      <div className="flex flex-wrap items-end justify-between gap-2">
        <div>
          <h2 className="section-title">Servidores de aplicación</h2>
          <p className="section-subtitle">Tres servidores en zonas horarias distintas. Cada uno tiene su propio reloj de Lamport y su componente en el reloj vectorial.</p>
        </div>
        {cluster && <span className="badge badge-blue">Esta página la atiende: {serverNames[cluster.served_by]?.name || cluster.served_by}</span>}
      </div>
      <div className="mt-5 grid gap-4 lg:grid-cols-3">
        {(cluster?.nodes ?? []).map((server) => {
          const info = serverNames[server.node_id] || { name: server.node_id, city: "" };
          return <div key={server.node_id} className={`rounded-2xl border-2 p-5 ${server.up ? "border-emerald-200 bg-white" : "border-red-200 bg-red-50"}`}>
            <div className="flex items-start justify-between gap-2">
              <div className="flex items-center gap-3">
                <span className={`flex h-10 w-10 items-center justify-center rounded-xl ${server.up ? "bg-navy-900 text-gold-300" : "bg-white text-red-400"}`}><Server className="h-5 w-5" /></span>
                <div><p className="font-semibold text-navy-900">{info.name}</p><p className="text-xs text-slate-500">{info.city}{server.time_zone ? ` · ${server.time_zone}` : ""}</p></div>
              </div>
              <span className={`badge ${server.up ? "badge-green" : "badge-red"}`}>{server.up ? "UP" : "DOWN"}</span>
            </div>
            {server.primary && <p className="mt-3 inline-flex items-center gap-1 text-xs font-semibold text-gold-700"><Crown className="h-3.5 w-3.5" /> Principal: importación y mantenimiento</p>}
            {server.up ? <>
              <p className="mt-3 font-mono text-sm text-navy-900">{server.local_time}</p>
              <div className="mt-3 rounded-xl bg-slate-50 p-3">
                <p className="text-xs text-slate-500">Reloj de Lamport</p>
                <p className="font-mono text-lg font-bold text-navy-900">{server.lamport_clock.toLocaleString("es-BO")}</p>
              </div>
              <div className="mt-2 rounded-xl bg-navy-950 p-3">
                <p className="text-xs text-navy-300">Reloj vectorial</p>
                <div className="mt-1.5 flex flex-wrap gap-1.5 font-mono text-xs">
                  {Object.entries(server.vector_clock || {}).sort(([a], [b]) => a.localeCompare(b)).map(([node, value]) =>
                    <span key={node} className={`rounded-md px-2 py-1 ${node === server.node_id ? "bg-gold-400 font-bold text-navy-950" : "bg-white/10 text-white"}`}>{node}: {value}</span>)}
                </div>
              </div>
            </> : <p className="mt-3 text-sm text-red-700">Sin respuesta. Sus peticiones las atienden los otros servidores.</p>}
          </div>;
        })}
        {!cluster && <p className="text-sm text-slate-500">Consultando servidores…</p>}
      </div>
      <p className="mt-4 text-xs text-slate-500">En dorado, la componente propia de cada servidor. Dos vectores son concurrentes cuando ninguno es mayor o igual al otro en todas sus componentes.</p>
    </section>

    {/* Topology: two regional PostgreSQL nodes replicating to each other and to MongoDB */}
    <section className="card card-body">
      <h2 className="section-title">Bases de datos</h2>
      <p className="section-subtitle">Cada operación se guarda con relojes de Lamport y vectoriales y se replica a los otros nodos.</p>
      <div className="relative mt-6 grid gap-4 md:grid-cols-3">
        <div aria-hidden="true" className="absolute left-[16%] right-[16%] top-1/2 hidden border-t-2 border-dashed border-navy-200 md:block" />
        {nodeKeys.map((key) => {
          const node = nodes?.[key];
          const style = stateStyle(node?.state);
          const online = node?.state === "UP";
          return <div key={key} className={`relative rounded-2xl border-2 p-5 text-center transition ${style.ring}`}>
            <span className={`mx-auto flex h-14 w-14 items-center justify-center rounded-2xl ${online ? "bg-navy-900 text-gold-300" : "bg-white text-slate-400"}`}>
              {online ? <Database className="h-7 w-7" /> : <ServerCrash className="h-7 w-7" />}
            </span>
            <p className="mt-3 font-semibold text-navy-900">{nodeNames[key]}</p>
            <span className={`badge mt-2 ${style.badge}`}><span className={`h-1.5 w-1.5 rounded-full ${style.dot} ${online ? "animate-pulse" : ""}`} />{node?.state ?? "CARGANDO"}</span>
          </div>;
        })}
      </div>
    </section>

    <div className="grid gap-4 lg:grid-cols-3">
      {nodeKeys.map((key) => {
        const node = nodes?.[key];
        const online = node?.state === "UP";
        return <section key={key} className="card card-body">
          <div className="flex items-start justify-between gap-2">
            <h2 className="font-semibold text-navy-900">{nodeNames[key]}</h2>
            <span className={`badge ${stateStyle(node?.state).badge}`}>{node?.state ?? "CARGANDO"}</span>
          </div>
          <div className="mt-4 grid grid-cols-2 gap-3">
            <div className="rounded-xl bg-slate-50 p-3"><p className="text-xs text-slate-500">Vuelos</p><p className="text-xl font-bold text-navy-900">{count(node?.flights, language)}</p></div>
            <div className="rounded-xl bg-slate-50 p-3"><p className="text-xs text-slate-500">Boletos</p><p className="text-xl font-bold text-navy-900">{count(node?.tickets, language)}</p></div>
          </div>
          <p className="mt-4 flex items-center gap-1.5 text-xs text-slate-500"><Clock className="h-3.5 w-3.5" aria-hidden="true" />{online ? (language === "en" ? `Responds in ${node.latency_ms} ms` : `Responde en ${node.latency_ms} ms`) : `Última vez activo: ${formatTime(node?.last_seen_at, language)}`}</p>
          {key !== "mongo" && <p className="mt-1 text-xs text-slate-500">Cola pendiente: <strong className="text-navy-900">{count(node?.pending, language)}</strong></p>}
        </section>;
      })}
    </div>

    <div className="grid gap-4 lg:grid-cols-2">
      <section className="card card-body">
        <h2 className="section-title">Diferencias observables</h2>
        <p className="section-subtitle">Comparación de cantidades. Una diferencia de cero no verifica que cada versión sea idéntica.</p>
        <dl className="mt-4 divide-y divide-slate-100 text-sm">
          {[
            ["Vuelos AM ↔ EU", status?.count_difference.am_eu_flights],
            ["Boletos AM ↔ EU", status?.count_difference.am_eu_tickets],
            ["Vuelos AM ↔ Mongo", status?.count_difference.am_mongo_flights],
            ["Vuelos EU ↔ Mongo", status?.count_difference.eu_mongo_flights],
          ].map(([label, value]) => <div key={String(label)} className="flex items-center justify-between py-2.5">
            <dt className="text-slate-600">{label}</dt>
            <dd className={`font-mono font-bold ${value === 0 ? "text-emerald-700" : value == null ? "text-slate-400" : "text-amber-700"}`}>{count(value as number | null | undefined, language)}</dd>
          </div>)}
        </dl>
        <div className="alert alert-info mt-4">Operaciones esperando entrega completa: <strong>{count(status?.pending, language)}</strong></div>
        <p className="mt-2 text-xs text-slate-500">Si una PostgreSQL está caída, su cola local no puede consultarse y el total visible puede ser parcial.</p>
        <div className="mt-4 flex gap-2"><Link href="/vuelos" className="btn-ghost btn-sm">Ver vuelos <ArrowRight size={14} /></Link><Link href="/boletos" className="btn-ghost btn-sm">Ver boletos <ArrowRight size={14} /></Link></div>
      </section>
      <section className="card card-body">
        <h2 className="section-title">Eventos detectados</h2>
        <p className="section-subtitle">Horas locales. Se registran los cambios mientras la API está encendida.</p>
        <ol className="mt-4 max-h-80 space-y-3 overflow-y-auto pr-1">
          {[...(status?.events ?? [])].reverse().map((event) => <li key={event.id} className="relative border-l-2 border-navy-200 pl-4">
            <span aria-hidden="true" className="absolute -left-[5px] top-1.5 h-2 w-2 rounded-full bg-navy-600" />
            <time className="text-xs font-mono text-slate-500">{formatTime(event.at, language)}</time>
            <p className="text-sm text-navy-900">{language === "en" ? translateUiText(event.message) : event.message}</p>
          </li>)}
          {!status?.events?.length && <li className="text-sm text-slate-500">Esperando cambios de estado reales…</li>}
        </ol>
      </section>
    </div>
  </div>;
}
