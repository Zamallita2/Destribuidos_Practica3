"use client";

import { useEffect, useState } from "react";
import { CheckCircle2, DatabaseZap, FileSpreadsheet, Loader2, UploadCloud, AlertTriangle } from "lucide-react";
import { translateUiText } from "@/lib/englishUi";
import { useLanguage } from "@/context/LanguageContext";
import { KpiCard, PageHeader } from "@/components/ui";

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
  const { language } = useLanguage();
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
    const warning = "Esto borrará todos los vuelos, boletos y reservas anteriores y los reemplazará usando los archivos seleccionados. ¿Continuar?";
    if (!window.confirm(language === "en" ? translateUiText(warning) : warning)) return;
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
    <div className="fade-up space-y-6">
      <PageHeader icon={DatabaseZap} eyebrow="Operación" title="Datos de entrada"
        subtitle="Carga un dataset nuevo, una matriz, varias matrices o todos los archivos. El sistema combinará lo nuevo con los archivos activos y validará el resultado antes de reemplazar los datos." />

      <section>
        <h2 className="section-title mb-3">Datos activos</h2>
        {current ? (
          <>
            <div className="grid gap-4 sm:grid-cols-3">
              <div className="card card-body"><p className="text-sm text-slate-500">Dataset</p><p className="mt-1 break-all font-mono text-sm font-semibold text-navy-900">{current.dataset}</p></div>
              <KpiCard label="Filas de origen" value={current.rows?.toLocaleString("es-BO")} />
              <KpiCard label="Aeropuertos en matrices" value={current.airports} />
            </div>
            {current.last_import && <p className="mt-3 text-sm text-slate-500">Última carga: {new Date(current.last_import.updated_at).toLocaleString("es-BO")} · {current.last_import.files.join(" · ")}</p>}
          </>
        ) : <p className="text-slate-500">Cargando información actual…</p>}
      </section>

      <section className="card card-body">
        <h2 className="section-title flex items-center gap-2"><UploadCloud className="h-5 w-5 text-navy-500" /> Seleccionar archivos</h2>
        <p className="section-subtitle mb-5">Para Excel de matrices, pon los aeropuertos de destino en la primera fila y los de origen en la primera columna. Las celdas contienen horas o precios. Solo necesitas seleccionar lo que deseas cambiar.</p>
        <div className="grid gap-4 lg:grid-cols-2">
          {fields.map((field) => {
            const chosen = files[field.key];
            return <label key={field.key} className={`block cursor-pointer rounded-2xl border-2 border-dashed p-4 transition ${chosen ? "border-emerald-300 bg-emerald-50/60" : "border-slate-200 hover:border-navy-300 hover:bg-navy-50/40"}`}>
              <span className="flex items-center gap-2 font-semibold text-navy-900">
                {chosen ? <CheckCircle2 className="h-4 w-4 text-emerald-600" /> : <FileSpreadsheet className="h-4 w-4 text-navy-500" />}{field.title}
              </span>
              <span className="mt-1 block min-h-8 text-xs text-slate-500">{field.help}</span>
              <input type="file" accept={field.accept} disabled={running || busy}
                className="mt-3 block w-full text-sm text-slate-600 file:mr-3 file:rounded-lg file:border-0 file:bg-navy-900 file:px-3 file:py-2 file:text-sm file:font-semibold file:text-white hover:file:bg-navy-700"
                onChange={(event) => {
                  const selected = event.target.files?.[0];
                  setFiles((previous) => ({ ...previous, [field.key]: selected }));
                  setJob(null);
                  window.localStorage.removeItem("airres-input-job");
                }} />
              {chosen && <span className="mt-2 block text-xs font-medium text-emerald-700">{chosen.name}</span>}
            </label>;
          })}
        </div>
        <p className="mt-4 text-xs text-slate-500">Los aeropuertos de las matrices deben tener región, país y zona horaria en el catálogo actual. La vista previa cuenta como no importables las filas con fecha, avión o ruta no válidos.</p>
        <button onClick={preview} disabled={!selectedCount || busy || running} className="btn-primary mt-5">
          {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <UploadCloud className="h-4 w-4" />} Validar y preparar
        </button>
      </section>

      {error && <div role="alert" className="alert alert-error">{error}</div>}

      {job && <section className="card card-body space-y-5">
        <div className="flex items-center gap-3">
          <span className={`flex h-10 w-10 items-center justify-center rounded-xl ${job.state === "completed" ? "bg-emerald-50 text-emerald-600" : job.state === "failed" ? "bg-red-50 text-red-600" : "bg-navy-50 text-navy-600"}`}>
            {job.state === "completed" ? <CheckCircle2 /> : job.state === "failed" ? <AlertTriangle /> : <DatabaseZap />}
          </span>
          <div>
            <h2 className="section-title">{job.state === "preview" ? "Vista previa" : job.state === "running" ? "Procesando datos" : job.state === "completed" ? "Datos listos" : "Proceso interrumpido"}</h2>
            <p className="text-sm text-slate-500">{job.step}</p>
          </div>
        </div>
        {job.error && <p role="alert" className="alert alert-error">{job.error}</p>}
        <div className="grid gap-3 sm:grid-cols-3">
          <div className="rounded-xl bg-slate-50 p-4 text-sm text-slate-600">Filas: <strong className="block text-2xl text-navy-900">{job.rows.toLocaleString("es-BO")}</strong></div>
          <div className="rounded-xl bg-emerald-50 p-4 text-sm text-emerald-800">Vuelos admitidos: <strong className="block text-2xl">{job.eligible.toLocaleString("es-BO")}</strong></div>
          <div className="rounded-xl bg-amber-50 p-4 text-sm text-amber-900">Filas no importables: <strong className="block text-2xl">{job.rejected.toLocaleString("es-BO")}</strong></div>
        </div>
        <ul className="flex flex-wrap gap-2">{job.files.map((file) => <li key={file} className="badge badge-slate">{file}</li>)}</ul>
        {job.state !== "preview" && <div>
          <div className="mb-2 flex justify-between text-sm font-medium text-slate-600"><span>Progreso</span><span>{job.percent}%</span></div>
          <div role="progressbar" aria-valuenow={job.percent} aria-valuemin={0} aria-valuemax={100} className="h-3 overflow-hidden rounded-full bg-slate-100">
            <div className="h-full rounded-full bg-gradient-to-r from-navy-700 to-gold-400 transition-all duration-500" style={{ width: `${job.percent}%` }} />
          </div>
        </div>}
        {job.state === "preview" && <>
          <div className="alert alert-warning"><AlertTriangle className="h-5 w-5 shrink-0" />Al procesar se borrarán todos los vuelos, boletos y reservas anteriores. Los archivos actuales no se modifican durante esta vista previa.</div>
          <button onClick={process} disabled={busy} className="btn-primary">Procesar y reemplazar datos</button>
        </>}
        {job.state === "completed" && <div className="alert alert-success flex-col"><p>Importados: {job.america.toLocaleString("es-BO")} vuelos de América y {job.other.toLocaleString("es-BO")} de Europa/Asia. Ya puedes usar el sistema.</p><a className="font-semibold underline" href="/api/entradas/rechazos.csv">Descargar rechazos por ruta (CSV)</a></div>}
      </section>}
    </div>
  );
}
