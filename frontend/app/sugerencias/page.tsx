"use client";

import { useState, useEffect } from "react";
import { Search, Activity, MapPin, Loader2, Map as MapIcon, Clock, DollarSign, Plane, ArrowLeftRight } from "lucide-react";
import { EmptyState, PageHeader } from "@/components/ui";

interface Ciudad {
  id: number;
  codigo: string;
  pais: string;
  region: string;
}

interface RouteDetails {
  salida: string;
  llegada: string;
  cost: number;
  time: number;
}

interface SuggestedRoute {
  ruta: string[];
  costo: number;
  tiempo: number;
  vuelos: RouteDetails[];
}

export default function DijkstraSugerencias() {
  const [origen, setOrigen] = useState("");
  const [destino, setDestino] = useState("");
  const [clase, setClase] = useState("regular");
  const [criterio, setCriterio] = useState("tiempo"); // tiempo o costo
  
  const [ciudades, setCiudades] = useState<Ciudad[]>([]);
  const [resultados, setResultados] = useState<SuggestedRoute[] | null>(null);
  const [loading, setLoading] = useState(false);
  const [activeTab, setActiveTab] = useState(0);

  useEffect(() => {
    fetch("/api/ciudades", {
      headers: { "X-User-Country": "CO" }
    })
      .then(res => res.json())
      .then(data => {
        if (Array.isArray(data)) {
          // Remover duplicados si los hay y ordenar alfabetico
          const uniques = data.filter((v,i,a)=>a.findIndex(v2=>(v2.codigo===v.codigo))===i);
          uniques.sort((a,b) => a.codigo.localeCompare(b.codigo));
          setCiudades(uniques);
        }
      })
      .catch(err => console.error("Error cargando ciudades:", err));
  }, []);

  const simulateDijkstra = async () => {
    setLoading(true);
    setResultados(null);
    setActiveTab(0);
    try {
      const res = await fetch(`/api/sugerencias/${criterio}?origen=${origen}&destino=${destino}&clase=${clase}`, {
        headers: { "X-User-Country": "CO" }
      });
      const data = await res.json();
      setResultados(data);
    } catch(err) {
      console.error(err);
      setResultados([]);
    } finally {
      setLoading(false);
    }
  };

  const currentRoute = resultados && resultados.length > activeTab ? resultados[activeTab] : null;

  const money = (value: number) => `$${Math.round(value).toLocaleString("es-BO")}`;

  return (
    <div className="fade-up space-y-6">
      <PageHeader icon={MapIcon} eyebrow="Planificador" title="Sugeridor de Rutas Inteligente"
        subtitle="Implementación de Algoritmo K-Shortest Paths (Dijkstra) para obtener el Top 3 de la ruta óptima minimizando costo o tiempo de vuelo." />

      <section className="card card-body">
        <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)] lg:items-end">
          <div>
            <label htmlFor="route-origin" className="field-label flex items-center gap-1.5"><MapPin className="h-3.5 w-3.5" /> Aeropuerto Origen</label>
            <select id="route-origin" value={origen} onChange={e => setOrigen(e.target.value)} className="field">
              <option value="">Seleccione Origen...</option>
              {ciudades.map(c => <option key={c.codigo} value={c.codigo}>{c.codigo} - {c.pais}</option>)}
            </select>
          </div>
          <button type="button" aria-label="Intercambiar origen y destino" onClick={() => { setOrigen(destino); setDestino(origen); }}
            className="mx-auto flex h-10 w-10 items-center justify-center rounded-full border border-slate-300 bg-white text-navy-700 shadow-sm transition hover:bg-navy-50 lg:mb-0.5">
            <ArrowLeftRight className="h-4 w-4" />
          </button>
          <div>
            <label htmlFor="route-destination" className="field-label flex items-center gap-1.5"><MapPin className="h-3.5 w-3.5" /> Aeropuerto Destino</label>
            <select id="route-destination" value={destino} onChange={e => setDestino(e.target.value)} className="field">
              <option value="">Seleccione Destino...</option>
              {ciudades.map(c => <option key={c.codigo} value={c.codigo}>{c.codigo} - {c.pais}</option>)}
            </select>
          </div>
        </div>
        <div className="mt-5 grid gap-4 md:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto] md:items-end">
          <div>
            <p className="field-label">Asiento Deseado</p>
            <div className="segmented" role="group" aria-label="Asiento Deseado">
              <button type="button" data-active={clase === "regular"} onClick={() => setClase("regular")}>Regular</button>
              <button type="button" data-active={clase === "vip"} onClick={() => setClase("vip")}>Primera Clase</button>
            </div>
          </div>
          <div>
            <p className="field-label">Criterio de Optimización</p>
            <div className="segmented" role="group" aria-label="Criterio de Optimización">
              <button type="button" data-active={criterio === "tiempo"} onClick={() => setCriterio("tiempo")}>Menor Tiempo</button>
              <button type="button" data-active={criterio === "costo"} onClick={() => setCriterio("costo")}>Menor Costo</button>
            </div>
          </div>
          <button onClick={simulateDijkstra} disabled={!origen || !destino || loading || origen === destino} className="btn-gold btn-lg">
            {loading ? <Loader2 className="h-5 w-5 animate-spin" /> : <Search className="h-5 w-5" />}
            {loading ? "Procesando Grafo..." : "Buscar Top 3 Rutas"}
          </button>
        </div>
      </section>

      {resultados && (resultados.length === 0 ? (
        <EmptyState icon={Activity} title="Sin Rutas Posibles">No hemos podido encontrar una ruta que conecte estos destinos matemáticamente en estas condiciones.</EmptyState>
      ) : (
        <section className="space-y-5">
          <h2 className="section-title">Rutas Óptimas (Top {resultados.length})</h2>
          <div className="grid gap-3 md:grid-cols-3" role="tablist" aria-label="Rutas encontradas">
            {resultados.map((route, idx) => <button key={idx} type="button" role="tab" aria-selected={activeTab === idx} onClick={() => setActiveTab(idx)}
              className={`rounded-2xl border bg-white p-4 text-left shadow-card transition hover:-translate-y-0.5 hover:shadow-lift ${activeTab === idx ? "border-navy-800 ring-2 ring-navy-800" : "border-slate-200"}`}>
              <div className="flex items-center justify-between gap-2">
                <span className={`badge ${idx === 0 ? "badge-gold" : "badge-slate"}`}>{idx === 0 ? "Ruta Óptima" : `Alterna ${idx}`}</span>
                <span className="text-xs text-slate-500">{route.ruta.length - 2 <= 0 ? "Directo" : `${route.ruta.length - 2} escala(s)`}</span>
              </div>
              <p className="mt-3 truncate font-mono text-sm font-semibold text-navy-900">{route.ruta.join(" → ")}</p>
              <div className="mt-3 flex items-end justify-between">
                <span className="text-2xl font-bold text-navy-900">{money(route.costo)}</span>
                <span className="flex items-center gap-1 text-sm font-semibold text-slate-600"><Clock className="h-4 w-4" />{route.tiempo} Hrs</span>
              </div>
            </button>)}
          </div>

          {currentRoute && <div className="card overflow-hidden">
            <div className="grid gap-px bg-slate-200 sm:grid-cols-2">
              <div className="flex items-center justify-between bg-white p-5">
                <div><p className="eyebrow">Costo Total Estimado</p><p className="text-xs text-slate-500">{clase === 'vip' ? 'Primera Clase' : 'Clase Regular'}</p></div>
                <p className="flex items-center text-3xl font-bold text-navy-900"><DollarSign className="h-6 w-6 text-gold-500" />{Math.round(currentRoute.costo).toLocaleString("es-BO")}</p>
              </div>
              <div className="flex items-center justify-between bg-white p-5">
                <div><p className="eyebrow">Tiempo de Viaje Estimado</p><p className="text-xs text-slate-500">Incluyendo escalas</p></div>
                <p className="text-3xl font-bold text-navy-900">{currentRoute.tiempo} Hrs</p>
              </div>
            </div>
            <ol className="relative space-y-0 p-5 sm:p-6" aria-label="Itinerario">
              {currentRoute.ruta.map((nodo: string, i: number) => {
                const last = i === currentRoute.ruta.length - 1;
                const leg = currentRoute.vuelos[i];
                return <li key={i} className="relative flex gap-4 pb-6 last:pb-0">
                  {!last && <span aria-hidden="true" className="absolute left-5 top-10 h-[calc(100%-2.5rem)] w-px border-l-2 border-dashed border-navy-200" />}
                  <span className={`z-10 flex h-10 w-10 shrink-0 items-center justify-center rounded-full text-xs font-bold ${i === 0 || last ? "bg-navy-900 text-gold-300" : "border-2 border-navy-200 bg-white text-navy-700"}`}>{nodo}</span>
                  <div className="min-w-0 flex-1 pt-1">
                    <p className="text-sm font-semibold text-navy-900">{nodo} <span className="ml-1 text-xs font-medium uppercase tracking-wide text-slate-400">{i === 0 ? "Origen" : last ? "Destino" : "Escala"}</span></p>
                    {!last && leg && <p className="mt-1 flex flex-wrap items-center gap-3 text-xs text-slate-500">
                      <span className="flex items-center gap-1"><Plane className="h-3.5 w-3.5 rotate-90 text-gold-500" /> {leg.salida} → {leg.llegada}</span>
                      <span className="badge badge-slate">{money(leg.cost)}</span>
                      <span className="badge badge-slate">{leg.time} Hrs</span>
                    </p>}
                  </div>
                </li>;
              })}
            </ol>
          </div>}
        </section>
      ))}
    </div>
  );
}
