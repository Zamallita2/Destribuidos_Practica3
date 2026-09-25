"use client";

import { useEffect, useState } from "react";
import { CheckCircle2, DatabaseZap, FileSpreadsheet, Loader2, UploadCloud, AlertTriangle } from "lucide-react";

type FileKey = "dataset" | "matrices" | "travel_time" | "economy_fares" | "first_class_fares";
type InputJob = {
  id: string;
  state: "preview" | "running" | "completed" | "failed";
  step: string;
  percent: number;
  error?: string;
  rows: number;
  eligible: number;
  rejected: number;
  america: number;
  other: number;
  files: string[];
};
type CurrentInputs = { dataset: string; rows: number; airports: number; matrix: string; last_import?: { files: string[]; updated_at: string } };

const fields: { key: FileKey; title: string; help: string; accept: string }[] = [
  { key: "dataset", title: "Dataset de vuelos", help: "CSV o Excel con flight_date, flight_time, origin, destination, aircraft_id, status y gate.", accept: ".csv,.xlsx" },
  { key: "matrices", title: "Tres matrices en un JSON", help: "Opcional. Usa airports, travel_time, economy_fares y first_class_fares.", accept: ".json" },
  { key: "travel_time", title: "Matriz de tiempos", help: "JSON, CSV o Excel; valores en horas.", accept: ".json,.csv,.xlsx" },
  { key: "economy_fares", title: "Precios de clase turística", help: "JSON, CSV o Excel; vacío o null significa clase no disponible.", accept: ".json,.csv,.xlsx" },
  { key: "first_class_fares", title: "Precios de primera clase", help: "JSON, CSV o Excel; vacío o null significa clase no disponible.", accept: ".json,.csv,.xlsx" },
];

export default function EntradasPage() {
  const [files, setFiles] = useState<Partial<Record<FileKey, File>>>({});
  const [current, setCurrent] = useState<CurrentInputs | null>(null);
  const [job, setJob] = useState<InputJob | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    fetch("/api/entradas", { cache: "no-store" }).then((r) => r.json()).then(setCurrent).catch(() => {});
	const lastID = window.localStorage.getItem("airres-input-job");
	if (lastID) {
	  fetch(`/api/entradas/jobs/${lastID}`, { cache: "no-store" })
	    .then((response) => response.ok ? response.json() : null)
	    .then((previous) => { if (previous) setJob(previous); else window.localStorage.removeItem("airres-input-job"); })
	    .catch(() => {});
	}
  }, []);

  useEffect(() => {
    if (job?.state !== "running") return;
    const timer = window.setInterval(async () => {
      try {
        const response = await fetch(`/api/entradas/jobs/${job.id}`, { cache: "no-store" });
        if (!response.ok) throw new Error("No se pudo consultar el progreso");
        const next: InputJob = await response.json();
        setJob(next);
        if (next.state === "completed") {
          fetch("/api/entradas", { cache: "no-store" }).then((r) => r.json()).then(setCurrent).catch(() => {});
        }
      } catch (cause) {
        setError(cause instanceof Error ? cause.message : "No se pudo consultar el progreso");
      }
    }, 1500);
    return () => window.clearInterval(timer);
  }, [job?.id, job?.state]);

  async function preview() {
    setError("");
    setBusy(true);
    setJob(null);
    try {
      const body = new FormData();
      for (const field of fields) {
        const file = files[field.key];
        if (file) body.append(field.key, file);
      }
      const response = await fetch("/api/entradas/preview", { method: "POST", body });
      const result = await response.json();
      if (!response.ok) throw new Error(result.error || "No se pudieron validar los archivos");
      setJob(result);
      window.localStorage.setItem("airres-input-job", result.id);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Error al validar");
    } finally {
      setBusy(false);
    }
  }

  async function process() {
    if (!job || job.state !== "preview") return;
    if (!window.confirm("Esto borrará todos los vuelos, boletos y reservas anteriores y los reemplazará usando los archivos seleccionados. ¿Continuar?")) return;
    setError("");
    setBusy(true);
    try {
      const response = await fetch(`/api/entradas/jobs/${job.id}/start`, { method: "POST" });
      const result = await response.json();
      if (!response.ok) throw new Error(result.error || "No se pudo iniciar el procesamiento");
      setJob({ ...job, state: "running", step: "Preparando reemplazo", percent: 5 });
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Error al iniciar");
    } finally {
      setBusy(false);
    }
  }

  const running = job?.state === "running";
  const selectedCount = Object.values(files).filter(Boolean).length;

  return (
    <div className="space-y-8 pb-12">
      <div>
        <h2 className="text-3xl font-bold text-gradient flex items-center gap-3"><DatabaseZap className="w-8 h-8 text-blue-400" /> Datos de entrada</h2>
        <p className="text-gray-400 mt-2 max-w-3xl">Carga un dataset nuevo, una matriz, varias matrices o todos los archivos. El sistema combinará lo nuevo con los archivos activos y validará el resultado antes de reemplazar los datos.</p>
      </div>

      <section className="glass-panel p-6">
        <h3 className="font-semibold text-xl mb-4">Datos activos</h3>
        {current ? (
          <><div className="grid sm:grid-cols-3 gap-4 text-sm">
            <div className="glass-card p-4"><p className="text-gray-400">Dataset</p><p className="font-semibold mt-1 break-all">{current.dataset}</p></div>
            <div className="glass-card p-4"><p className="text-gray-400">Filas de origen</p><p className="font-semibold text-2xl mt-1">{current.rows?.toLocaleString("es-BO")}</p></div>
            <div className="glass-card p-4"><p className="text-gray-400">Aeropuertos en matrices</p><p className="font-semibold text-2xl mt-1">{current.airports}</p></div>
          </div>{current.last_import && <p className="text-sm text-gray-400 mt-4">Última carga: {new Date(current.last_import.updated_at).toLocaleString("es-BO")} · {current.last_import.files.join(" · ")}</p>}</>
        ) : <p className="text-gray-400">Cargando información actual…</p>}
      </section>

      <section className="glass-panel p-6">
        <h3 className="font-semibold text-xl flex items-center gap-2"><UploadCloud className="w-5 h-5 text-blue-400" /> Seleccionar archivos</h3>
        <p className="text-gray-400 text-sm mt-2 mb-5">Para Excel de matrices, pon los aeropuertos de destino en la primera fila y los de origen en la primera columna. Las celdas contienen horas o precios. Solo necesitas seleccionar lo que deseas cambiar.</p>
        <div className="grid lg:grid-cols-2 gap-4">
          {fields.map((field) => (
            <label key={field.key} className="glass-card p-4 block cursor-pointer hover:border-blue-400/50 transition">
              <span className="flex items-center gap-2 font-semibold"><FileSpreadsheet className="w-4 h-4 text-blue-400" />{field.title}</span>
              <span className="block text-xs text-gray-400 mt-1 min-h-8">{field.help}</span>
              <input type="file" accept={field.accept} disabled={running || busy} className="mt-3 block w-full text-sm text-gray-300 file:mr-3 file:rounded-lg file:border-0 file:bg-blue-500/20 file:px-3 file:py-2 file:text-blue-200 hover:file:bg-blue-500/30" onChange={(event) => {
                const selected = event.target.files?.[0];
                setFiles((previous) => ({ ...previous, [field.key]: selected }));
                setJob(null);
                window.localStorage.removeItem("airres-input-job");
              }} />
              {files[field.key] && <span className="text-xs text-emerald-300 block mt-2">{files[field.key]?.name}</span>}
            </label>
          ))}
        </div>
        <p className="text-xs text-gray-500 mt-4">Los aeropuertos de las matrices deben tener región, país y zona horaria en el catálogo actual. La vista previa cuenta como no importables las filas con fecha, avión o ruta no válidos.</p>
        <button onClick={preview} disabled={!selectedCount || busy || running} className="btn-primary mt-6 disabled:opacity-40 disabled:cursor-not-allowed flex items-center gap-2">
          {busy ? <Loader2 className="w-4 h-4 animate-spin" /> : <UploadCloud className="w-4 h-4" />} Validar y preparar
        </button>
      </section>

      {error && <div role="alert" className="rounded-xl border border-red-500/40 bg-red-500/10 p-4 text-red-200">{error}</div>}

      {job && <section className="glass-panel p-6 space-y-5">
        <div className="flex items-center gap-3">
          {job.state === "completed" ? <CheckCircle2 className="text-emerald-400" /> : job.state === "failed" ? <AlertTriangle className="text-red-400" /> : <DatabaseZap className="text-blue-400" />}
          <h3 className="text-xl font-semibold">{job.state === "preview" ? "Vista previa" : job.state === "running" ? "Procesando datos" : job.state === "completed" ? "Datos listos" : "Proceso interrumpido"}</h3>
        </div>
        <p className="text-gray-300">{job.step}</p>
        {job.error && <p role="alert" className="text-red-300">{job.error}</p>}
        <div className="grid sm:grid-cols-3 gap-3 text-sm">
          <div className="glass-card p-3">Filas: <strong>{job.rows.toLocaleString("es-BO")}</strong></div>
          <div className="glass-card p-3">Vuelos admitidos: <strong>{job.eligible.toLocaleString("es-BO")}</strong></div>
          <div className="glass-card p-3">Filas no importables: <strong>{job.rejected.toLocaleString("es-BO")}</strong></div>
        </div>
        <ul className="text-sm text-gray-400 list-disc pl-5">{job.files.map((file) => <li key={file}>{file}</li>)}</ul>
        {job.state !== "preview" && <div>
          <div className="flex justify-between text-sm mb-2"><span>Progreso</span><span>{job.percent}%</span></div>
          <div role="progressbar" aria-valuenow={job.percent} aria-valuemin={0} aria-valuemax={100} className="h-3 rounded-full bg-white/10 overflow-hidden"><div className="h-full bg-gradient-to-r from-blue-500 to-emerald-400 transition-all duration-500" style={{ width: `${job.percent}%` }} /></div>
        </div>}
        {job.state === "preview" && <>
          <div className="rounded-xl border border-amber-500/30 bg-amber-500/10 p-4 text-sm text-amber-100">Al procesar se borrarán todos los vuelos, boletos y reservas anteriores. Los archivos actuales no se modifican durante esta vista previa.</div>
          <button onClick={process} disabled={busy} className="btn-primary disabled:opacity-40">Procesar y reemplazar datos</button>
        </>}
        {job.state === "completed" && <div className="text-emerald-300 text-sm space-y-2"><p>Importados: {job.america.toLocaleString("es-BO")} vuelos de América y {job.other.toLocaleString("es-BO")} de Europa/Asia. Ya puedes usar el sistema.</p><a className="text-blue-300 underline" href="/api/entradas/rechazos.csv">Descargar rechazos por ruta (CSV)</a></div>}
      </section>}
    </div>
  );
}
