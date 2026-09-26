"use client";

import { useEffect, useState } from "react";
import { useLanguage } from "@/context/LanguageContext";
import { Route as RouteIcon, Loader2, Plane, Clock, DollarSign, Check, RotateCcw } from "lucide-react";
import { EmptyState, PageHeader } from "@/components/ui";

type City = { codigo: string; pais: string };
type Route = {
  ruta: string[];
  costo: number;
  tiempo: number;
  vuelos: { salida: string; llegada: string; cost: number; time: number }[];
};

export default function TSPPage() {
  const { language } = useLanguage();
  const es = language === "es";
  const [cities, setCities] = useState<City[]>([]);
  const [selected, setSelected] = useState<string[]>([]);
  const [criterion, setCriterion] = useState<"COST" | "TIME">("COST");
  const [seatClass, setSeatClass] = useState<"REGULAR" | "VIP">("REGULAR");
  const [returnToOrigin, setReturnToOrigin] = useState(false);
  const [result, setResult] = useState<Route | null>(null);
  const [message, setMessage] = useState("");
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    fetch("/api/ciudades", { headers: { "X-Region": "America" } })
      .then((response) => response.json())
      .then((items) => setCities(Array.isArray(items) ? items : []))
      .catch(() => setMessage(es ? "No se pudo cargar el catálogo de ciudades." : "Could not load cities."));
  }, [es]);

  const toggle = (code: string) => {
    setSelected((current) => current.includes(code) ? current.filter((item) => item !== code) : [...current, code]);
    setResult(null);
  };
  const calculate = async () => {
    setLoading(true);
    setMessage("");
    setResult(null);
    try {
      const response = await fetch("/api/tsp", {
        method: "POST",
        headers: { "Content-Type": "application/json", "X-Region": "America" },
        body: JSON.stringify({
          ciudades: selected,
          criterio: criterion,
          seat_class: seatClass,
          return_to_origin: returnToOrigin,
        }),
      });
      const payload = await response.json();
      if (!response.ok) {
        setMessage(payload.error || (es ? "No existe un recorrido válido." : "No valid route exists."));
      } else {
        setResult(payload);
      }
    } catch {
      setMessage(es ? "No se pudo consultar el servidor." : "Could not reach the server.");
    } finally {
      setLoading(false);
    }
  };

  const money = (value: number) => `$${value.toLocaleString(es ? "es-BO" : "en-US", { maximumFractionDigits: 2 })}`;

  return (
    <div className="fade-up space-y-6">
      <PageHeader icon={RouteIcon} eyebrow={es ? "Optimización" : "Optimization"} title={es ? "Agente Viajero (TSP)" : "Traveling Salesperson (TSP)"}
        subtitle={es
          ? "Ruta óptima de la red dirigida. Los tramos representan rutas de la matriz, no reservas con horario."
          : "Optimal path on the directed route network. Legs are matrix routes, not scheduled bookings."} />

      <section className="card card-body space-y-6">
        <div>
          <div className="mb-3 flex flex-wrap items-end justify-between gap-2">
            <div>
              <h2 className="section-title">{es ? "Ciudades a visitar" : "Cities to visit"}</h2>
              <p className="section-subtitle">{es ? "Elige al menos 2. La primera que elijas es el punto de partida si activas el regreso." : "Pick at least 2. The first one is the starting point when returning is enabled."}</p>
            </div>
            {selected.length > 0 && <button type="button" onClick={() => { setSelected([]); setResult(null); }} className="btn-ghost btn-sm">{es ? "Limpiar selección" : "Clear selection"}</button>}
          </div>
          <div className="grid grid-cols-3 gap-2 sm:grid-cols-5 lg:grid-cols-8">
            {cities.map((city) => {
              const order = selected.indexOf(city.codigo);
              const active = order >= 0;
              return <button key={city.codigo} type="button" onClick={() => toggle(city.codigo)} aria-pressed={active} title={city.pais}
                className={`relative rounded-xl border px-3 py-3 text-center transition ${active ? "border-navy-900 bg-navy-900 text-white shadow-lift" : "border-slate-200 bg-white text-navy-900 hover:border-navy-300 hover:bg-navy-50"}`}>
                {active && <span className="absolute -right-1.5 -top-1.5 flex h-5 w-5 items-center justify-center rounded-full bg-gold-400 text-[10px] font-bold text-navy-950">{order + 1}</span>}
                <span className="block font-mono text-base font-bold">{city.codigo}</span>
                <span className={`block truncate text-[10px] ${active ? "text-navy-200" : "text-slate-400"}`}>{city.pais}</span>
              </button>;
            })}
          </div>
        </div>

        <div className="grid gap-4 md:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto_auto] md:items-end">
          <div>
            <p className="field-label">{es ? "Optimizar" : "Optimize"}</p>
            <div className="segmented" role="group" aria-label={es ? "Optimizar" : "Optimize"}>
              <button type="button" data-active={criterion === "COST"} onClick={() => setCriterion("COST")}>{es ? "Costo" : "Cost"}</button>
              <button type="button" data-active={criterion === "TIME"} onClick={() => setCriterion("TIME")}>{es ? "Tiempo" : "Time"}</button>
            </div>
          </div>
          <div>
            <p className="field-label">{es ? "Clase" : "Class"}</p>
            <div className="segmented" role="group" aria-label={es ? "Clase" : "Class"}>
              <button type="button" data-active={seatClass === "REGULAR"} onClick={() => setSeatClass("REGULAR")}>{es ? "Turística" : "Economy"}</button>
              <button type="button" data-active={seatClass === "VIP"} onClick={() => setSeatClass("VIP")}>{es ? "Primera" : "First"}</button>
            </div>
          </div>
          <label className="flex cursor-pointer items-center gap-2 rounded-xl border border-slate-200 px-3.5 py-2.5 text-sm font-medium text-navy-900">
            <input type="checkbox" checked={returnToOrigin} onChange={(event) => setReturnToOrigin(event.target.checked)} className="h-4 w-4 accent-navy-800" />
            <RotateCcw className="h-4 w-4 text-slate-400" aria-hidden="true" />
            {es ? "Regresar al origen" : "Return to start"}
          </label>
          <button disabled={selected.length < 2 || loading} onClick={calculate} className="btn-gold btn-lg">
            {loading ? <Loader2 className="h-5 w-5 animate-spin" /> : <RouteIcon className="h-5 w-5" />}
            {loading ? (es ? "Calculando…" : "Calculating…") : (es ? "Calcular ruta" : "Calculate route")}
          </button>
        </div>
      </section>

      {message && <p role="alert" className="alert alert-error">{message}</p>}

      {result ? (
        <section className="card overflow-hidden">
          <div className="bg-navy-900 p-5 text-white sm:p-6">
            <p className="flex items-center gap-2 text-xs font-semibold uppercase tracking-[0.18em] text-gold-300"><Check className="h-4 w-4" /> {es ? "Recorrido óptimo" : "Optimal tour"}</p>
            <p className="mt-2 break-words font-mono text-xl font-bold text-white sm:text-2xl">{result.ruta.join(" → ")}</p>
          </div>
          <div className="grid gap-px bg-slate-200 sm:grid-cols-3">
            <div className="bg-white p-5"><p className="eyebrow">{es ? "Costo" : "Cost"}</p><p className="mt-1 flex items-center text-2xl font-bold text-navy-900"><DollarSign className="h-5 w-5 text-gold-500" />{money(result.costo).slice(1)}</p></div>
            <div className="bg-white p-5"><p className="eyebrow">{es ? "Tiempo" : "Time"}</p><p className="mt-1 flex items-center gap-1 text-2xl font-bold text-navy-900"><Clock className="h-5 w-5 text-slate-400" />{result.tiempo} h</p></div>
            <div className="bg-white p-5"><p className="eyebrow">{es ? "Tramos" : "Legs"}</p><p className="mt-1 text-2xl font-bold text-navy-900">{result.vuelos.length}</p></div>
          </div>
          <ol className="divide-y divide-slate-100">
            {result.vuelos.map((leg, index) => <li key={index} className="flex flex-wrap items-center gap-4 px-5 py-3.5 sm:px-6">
              <span className="flex h-7 w-7 items-center justify-center rounded-full bg-navy-50 text-xs font-bold text-navy-700">{index + 1}</span>
              <span className="flex items-center gap-2 font-mono font-semibold text-navy-900">{leg.salida}<Plane className="h-4 w-4 rotate-90 text-gold-500" aria-hidden="true" />{leg.llegada}</span>
              <span className="ml-auto flex gap-2"><span className="badge badge-slate">{money(leg.cost)}</span><span className="badge badge-slate">{leg.time} h</span></span>
            </li>)}
          </ol>
        </section>
      ) : !message && <EmptyState icon={RouteIcon} title={es ? "Aún no hay recorrido" : "No tour yet"}>
        {es ? "Selecciona las ciudades y presiona «Calcular ruta» para obtener el orden de visita óptimo." : "Select the cities and press Calculate route to get the optimal visiting order."}
      </EmptyState>}
    </div>
  );
}
