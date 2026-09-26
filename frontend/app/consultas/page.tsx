"use client";

import { useEffect, useState } from "react";
import { Database, Play, Search } from "lucide-react";
import Link from "next/link";

type Result = { columns: string[]; rows: Record<string, unknown>[]; note?: string };

const reports = [
  { id: "1", title: "Vuelos realizados por pasajero", description: "Cuenta los vuelos anteriores a hoy durante el año indicado.", fields: ["pasajero", "anio"] },
  { id: "2", title: "Lista de embarque", description: "Pasajeros con asiento vendido en un vuelo.", fields: ["vuelo"] },
  { id: "3", title: "Historial de vuelos", description: "Fechas, rutas y clase de los vuelos de un pasajero.", fields: ["pasajero"] },
  { id: "4", title: "Ingresos por clase de un vuelo", description: "Asientos vendidos e ingresos de ejecutiva y turística.", fields: ["vuelo"] },
  { id: "5", title: "Ingresos por trimestre", description: "Ventas trimestrales de toda la aerolínea.", fields: [] },
  { id: "7", title: "Conexiones por servidor", description: "Conexiones abiertas en este momento en los tres nodos.", fields: [] },
] as const;

const labels: Record<string, string> = {
  pasajero: "Pasajero", anio: "Año", vuelos_realizados: "Vuelos realizados",
  asiento: "Asiento", fecha: "Fecha", ruta: "Ruta", clase: "Clase",
  asientos_vendidos: "Asientos vendidos", ingresos: "Ingresos", trimestre: "Trimestre",
  servidor: "Servidor", conexiones: "Conexiones", vuelo: "Vuelo",
};

function displayValue(value: unknown, column: string) {
  if (value === null || value === undefined) return "No disponible";
  if (typeof value === "number") {
    return new Intl.NumberFormat("es-BO", { minimumFractionDigits: column === "ingresos" ? 2 : 0, maximumFractionDigits: column === "ingresos" ? 2 : 0 }).format(value);
  }
  return String(value);
}

export default function ConsultasPage() {
  const [selected, setSelected] = useState("1");
  const [passenger, setPassenger] = useState("Priya Sharma");
  const [year, setYear] = useState(String(new Date().getFullYear()));
  const [flight, setFlight] = useState("100000093");
  const [node, setNode] = useState("america");
  const [sql, setSql] = useState("SELECT id_boleto, nombre_pasajero, estado FROM boletos LIMIT 20");
  const [result, setResult] = useState<Result | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const active = reports.find((item) => item.id === selected);
  const pageSize = 100;
  const pageCount = Math.max(1, Math.ceil((result?.rows.length || 0) / pageSize));
  const visibleRows = result?.rows.slice((page - 1) * pageSize, page * pageSize) || [];

  useEffect(() => {
    fetch("/api/vuelos?scope=all&view=page&limit=1")
      .then((response) => response.json())
      .then((body) => { if (body.items?.[0]?.id) setFlight((current) => current || String(body.items[0].id)); })
      .catch(() => {});
  }, []);

  async function execute() {
    setLoading(true);
    setError("");
    setResult(null);
    setPage(1);
    try {
      const params = new URLSearchParams({ pasajero: passenger, anio: year });
      if (flight) params.set("vuelo", flight);
      const response = selected === "sql"
        ? await fetch("/api/consultas/sql", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ node, sql }) })
        : await fetch(`/api/consultas/${selected}?${params}`);
      const body = await response.json();
      if (!response.ok) throw new Error(body.error || "No se pudo ejecutar la consulta");
      setResult(body);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Error desconocido");
    } finally {
      setLoading(false);
    }
  }

  return <div className="space-y-7 text-white">
    <div>
      <div className="flex items-center gap-3"><Database className="h-8 w-8 text-blue-400" /><h2 className="text-3xl font-bold">Consultas</h2></div>
      <p className="mt-2 text-gray-400">Ejecuta los seis reportes principales o escribe una consulta SQL de lectura. Los filtros iniciales usan datos reales de esta instalación y se pueden cambiar.</p>
    </div>

    <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
      {reports.map((report) => <button key={report.id} type="button" onClick={() => { setSelected(report.id); setResult(null); setError(""); setPage(1); }}
        className={`rounded-2xl border p-4 text-left transition ${selected === report.id ? "border-blue-400 bg-blue-500/15" : "border-white/10 bg-white/5 hover:bg-white/10"}`}>
        <span className="text-xs font-bold uppercase tracking-wide text-blue-300">Consulta {report.id}</span>
        <h3 className="mt-1 font-semibold">{report.title}</h3>
        <p className="mt-1 text-sm text-gray-400">{report.description}</p>
      </button>)}
      <button type="button" onClick={() => { setSelected("sql"); setResult(null); setError(""); setPage(1); }}
        className={`rounded-2xl border p-4 text-left transition ${selected === "sql" ? "border-purple-400 bg-purple-500/15" : "border-white/10 bg-white/5 hover:bg-white/10"}`}>
        <span className="text-xs font-bold uppercase tracking-wide text-purple-300">Libre</span>
        <h3 className="mt-1 font-semibold">Editor SQL</h3>
        <p className="mt-1 text-sm text-gray-400">Una consulta SELECT por nodo PostgreSQL, con límite de 500 filas.</p>
      </button>
    </div>

    <section className="rounded-2xl border border-white/10 bg-white/5 p-5 md:p-6">
      <h3 className="text-xl font-semibold">{active?.title || "Editor SQL"}</h3>
      {selected === "sql" ? <div className="mt-5 space-y-4">
        <label className="block text-sm text-gray-300">Nodo
          <select value={node} onChange={(event) => setNode(event.target.value)} className="mt-2 block w-full rounded-lg border border-white/20 bg-[#1b2234] p-3 text-white sm:w-72">
            <option value="america">PostgreSQL América</option><option value="europa">PostgreSQL Europa/Asia</option>
          </select>
        </label>
        <label className="block text-sm text-gray-300">SQL de solo lectura
          <textarea value={sql} onChange={(event) => setSql(event.target.value)} spellCheck={false} rows={7}
            className="mt-2 w-full rounded-lg border border-white/20 bg-[#111827] p-3 font-mono text-sm text-white outline-none focus:border-blue-400" />
        </label>
        <p className="text-xs text-gray-400">Tablas disponibles: vuelos, boletos, asientos, ciudades y ocupaciones_vuelo. Las consultas globales están en los reportes preparados.</p>
      </div> : <div className="mt-5 flex flex-wrap gap-4">
        {active?.fields.some((field) => field === "pasajero") && <label className="text-sm text-gray-300">Pasajero<input value={passenger} onChange={(event) => setPassenger(event.target.value)} className="mt-2 block rounded-lg border border-white/20 bg-[#1b2234] p-3 text-white" /></label>}
        {active?.fields.some((field) => field === "anio") && <label className="text-sm text-gray-300">Año<input type="number" value={year} onChange={(event) => setYear(event.target.value)} className="mt-2 block rounded-lg border border-white/20 bg-[#1b2234] p-3 text-white" /></label>}
        {active?.fields.some((field) => field === "vuelo") && <div><label className="text-sm text-gray-300">ID del vuelo<input type="number" min="1" value={flight} onChange={(event) => setFlight(event.target.value)} className="mt-2 block rounded-lg border border-white/20 bg-[#1b2234] p-3 text-white" /></label><Link href="/vuelos" className="mt-2 block text-xs text-blue-300 hover:underline">Buscar ID en Vuelos</Link></div>}
        {active?.fields.length === 0 && <p className="text-sm text-gray-400">Esta consulta no necesita parámetros.</p>}
      </div>}
      <button type="button" onClick={execute} disabled={loading} className="mt-6 flex items-center gap-2 rounded-lg bg-blue-600 px-5 py-3 font-semibold text-white hover:bg-blue-500 disabled:opacity-50">
        {loading ? <Search className="h-4 w-4 animate-pulse" /> : <Play className="h-4 w-4" />}{loading ? "Consultando..." : "Ejecutar consulta"}
      </button>
      {error && <p role="alert" className="mt-4 rounded-lg border border-red-500/30 bg-red-500/10 p-3 text-red-200">{error}</p>}
    </section>

    {result && <section className="rounded-2xl border border-white/10 bg-white/5 p-5 md:p-6">
      <div className="flex flex-wrap items-end justify-between gap-2"><h3 className="text-xl font-semibold">Resultados</h3><span className="text-sm text-gray-400">{result.rows.length} fila(s)</span></div>
      {result.note && <p className="mt-2 text-sm text-gray-400">{result.note}</p>}
      <div className="mt-4 overflow-x-auto rounded-lg border border-white/10">
        <table className="min-w-full text-left text-sm"><thead className="bg-white/10"><tr>{result.columns.map((column, index) => <th key={`${column}-${index}`} className="whitespace-nowrap px-4 py-3 font-semibold">{labels[column] || column}</th>)}</tr></thead>
          <tbody>{result.rows.length === 0 ? <tr><td colSpan={Math.max(result.columns.length, 1)} className="px-4 py-6 text-center text-gray-400">Sin resultados para estos parámetros.</td></tr> : visibleRows.map((row, index) => <tr key={(page - 1) * pageSize + index} className="border-t border-white/10">{result.columns.map((column, position) => <td key={`${column}-${position}`} className="whitespace-nowrap px-4 py-3">{displayValue(row[column], column)}</td>)}</tr>)}</tbody>
        </table>
      </div>
      {pageCount > 1 && <div className="mt-4 flex items-center justify-end gap-3 text-sm"><button type="button" disabled={page === 1} onClick={() => setPage(page - 1)} className="rounded-lg border border-white/20 px-3 py-2 disabled:opacity-40">Anterior</button><span>Página {page} de {pageCount}</span><button type="button" disabled={page === pageCount} onClick={() => setPage(page + 1)} className="rounded-lg border border-white/20 px-3 py-2 disabled:opacity-40">Siguiente</button></div>}
    </section>}
  </div>;
}
