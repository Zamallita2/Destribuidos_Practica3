"use client";

import { useState, useEffect } from "react";
import { ArrowRight, Search, Activity, Check, MapPin, Loader2 } from "lucide-react";

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

  return (
    <div className="max-w-4xl mx-auto space-y-8 animate-in slide-in-from-bottom-5 fade-in duration-500">
      <div>
        <h2 className="text-3xl font-bold bg-clip-text text-transparent bg-gradient-to-r from-blue-400 to-purple-400 mb-2">Sugeridor de Rutas Inteligente</h2>
        <p className="text-gray-400">Implementación de Algoritmo K-Shortest Paths (Dijkstra) para obtener el Top 3 de la ruta óptima minimizando costo o tiempo de vuelo.</p>
      </div>

      <div className="glass-panel p-6 flex flex-col md:flex-row gap-6">
        <div className="flex-1 space-y-4">
          <div>
            <label className="text-sm font-semibold text-gray-400 flex items-center gap-2 mb-1">
              <MapPin className="w-4 h-4 text-rose-400" />
              Aeropuerto Origen
            </label>
            <select value={origen} onChange={e=>setOrigen(e.target.value)} className="w-full glass-card p-3 rounded-xl focus:ring-2 focus:ring-blue-500 outline-none text-white appearance-none cursor-pointer">
              <option value="" className="text-black">Seleccione Origen...</option>
              {ciudades.map(c => (
                <option key={c.codigo} value={c.codigo} className="text-black">{c.codigo} - {c.pais}</option>
              ))}
            </select>
          </div>
          <div>
            <label className="text-sm font-semibold text-gray-400 flex items-center gap-2 mb-1">
              <MapPin className="w-4 h-4 text-blue-400" />
              Aeropuerto Destino
            </label>
            <select value={destino} onChange={e=>setDestino(e.target.value)} className="w-full glass-card p-3 rounded-xl focus:ring-2 focus:ring-blue-500 outline-none text-white appearance-none cursor-pointer">
              <option value="" className="text-black">Seleccione Destino...</option>
              {ciudades.map(c => (
                <option key={c.codigo} value={c.codigo} className="text-black">{c.codigo} - {c.pais}</option>
              ))}
            </select>
          </div>
        </div>

        <div className="flex-1 space-y-4">
           <div>
            <label className="text-sm font-semibold text-gray-400 mb-1 block">Asiento Deseado</label>
            <div className="flex bg-[#0f111a] rounded-xl p-1 border border-white/5">
              <button 
                onClick={()=>setClase("regular")} 
                className={`flex-1 p-2 rounded-lg text-sm font-medium transition-all duration-300 ${clase === 'regular' ? 'bg-blue-600 text-white shadow-lg shadow-blue-500/25' : 'text-gray-400 hover:text-white'}`}
              >
                Regular
              </button>
              <button 
                onClick={()=>setClase("vip")} 
                className={`flex-1 p-2 rounded-lg text-sm font-medium transition-all duration-300 ${clase === 'vip' ? 'bg-purple-600 text-white shadow-lg shadow-purple-500/25' : 'text-gray-400 hover:text-white'}`}
              >
                Primera Clase
              </button>
            </div>
          </div>

          <div>
            <label className="text-sm font-semibold text-gray-400 mb-1 block">Criterio de Optimización</label>
            <div className="flex bg-[#0f111a] rounded-xl p-1 border border-white/5">
              <button 
                onClick={()=>setCriterio("tiempo")} 
                className={`flex-1 p-2 rounded-lg text-sm font-medium transition-all duration-300 ${criterio === 'tiempo' ? 'bg-emerald-600/80 text-white shadow-lg shadow-emerald-500/25' : 'text-gray-400 hover:text-white'}`}
              >
                Menor Tiempo
              </button>
              <button 
                onClick={()=>setCriterio("costo")} 
                className={`flex-1 p-2 rounded-lg text-sm font-medium transition-all duration-300 ${criterio === 'costo' ? 'bg-orange-600/80 text-white shadow-lg shadow-orange-500/25' : 'text-gray-400 hover:text-white'}`}
              >
                Menor Costo
              </button>
            </div>
          </div>
        </div>
      </div>

      <div className="flex justify-center">
        <button 
          onClick={simulateDijkstra}
          disabled={!origen || !destino || loading || origen === destino}
          className="btn-primary w-full max-w-sm flex justify-center gap-2 items-center text-lg shadow-[0_0_20px_rgba(59,130,246,0.4)] disabled:opacity-50 disabled:cursor-not-allowed group"
        >
          {loading ? (
            <Loader2 className="w-5 h-5 animate-spin" />
          ) : (
            <Search className="w-5 h-5 group-hover:scale-110 transition-transform" />
          )}
          {loading ? "Procesando Grafo..." : "Buscar Top 3 Rutas"}
        </button>
      </div>

      {resultados && (
        <div className="glass-panel p-2 md:p-6 animate-in zoom-in-95 duration-500 relative border-t-2 border-t-blue-500 min-h-[400px]">
           <div className="absolute top-0 right-0 p-8 md:p-34 opacity-5 pointer-events-none">
             <Activity className="w-48 h-48 md:w-64 md:h-64" />
           </div>
           
           {resultados.length === 0 ? (
             <div className="flex flex-col items-center justify-center py-12 text-center">
               <Activity className="w-16 h-16 text-rose-500/50 mb-4" />
               <h3 className="text-2xl font-bold text-white mb-2">Sin Rutas Posibles</h3>
               <p className="text-gray-400">No hemos podido encontrar una ruta que conecte estos destinos matemáticamente en estas condiciones.</p>
             </div>
           ) : (
             <>
               <div className="flex flex-col md:flex-row items-center justify-between gap-4 mb-6 relative z-10 px-2 lg:px-6">
                 <h3 className="text-xl md:text-2xl font-bold flex items-center gap-3">
                   <Check className="text-emerald-400 w-8 h-8 p-1 bg-emerald-400/20 rounded-full" />
                   Rutas Óptimas (Top {resultados.length})
                 </h3>
                 
                 {/* Tabs para las 3 rutas */}
                 <div className="flex bg-[#0f111a]/80 backdrop-blur-md rounded-xl p-1 border border-white/10 shadow-lg">
                   {resultados.map((_, idx) => (
                      <button
                        key={idx}
                        onClick={() => setActiveTab(idx)}
                        className={`px-4 py-2 rounded-lg text-sm font-semibold transition-all duration-300 ${activeTab === idx ? 'bg-gradient-to-r from-blue-600 to-indigo-600 text-white shadow-md' : 'text-gray-400 hover:text-white hover:bg-white/5'}`}
                      >
                        {idx === 0 ? "Ruta Óptima" : `Alterna ${idx}`}
                      </button>
                   ))}
                 </div>
               </div>

               {currentRoute && (
                 <div className="animate-in fade-in slide-in-from-right-4 duration-500 p-2 md:p-6">
                    <div className="flex flex-wrap items-center justify-center gap-y-8 mb-12 relative w-full px-4 pt-10">
                       <div className="absolute top-16 left-0 w-full h-[2px] bg-gradient-to-r from-blue-500/10 via-blue-500/40 to-purple-500/10 z-0" />
                       
                       {currentRoute.ruta.map((nodo: string, i: number) => (
                         <div key={i} className="relative z-10 flex flex-col items-center flex-1 min-w-[70px]">
                           <div className={`w-12 h-12 md:w-14 md:h-14 rounded-xl flex items-center justify-center font-black shadow-lg mb-3 ${
                             i === 0 || i === currentRoute.ruta.length - 1 
                             ? 'bg-blue-600/20 border-2 border-blue-400 text-white shadow-blue-500/30' 
                             : 'bg-[#0f111a] border border-white/20 text-gray-300'
                           }`}>
                             {nodo}
                           </div>
                           <span className="text-xs uppercase tracking-widest font-bold text-gray-400">
                             {i === 0 ? "Origen" : i === currentRoute.ruta.length -1 ? "Destino" : "Escala"}
                           </span>
                           {i < currentRoute.ruta.length - 1 && (
                             <div className="absolute top-6 left-[calc(50%+1.5rem)] w-[calc(100%-3rem)] md:left-[calc(50%+2rem)] md:w-[calc(100%-4rem)] hidden md:flex flex-col items-center text-xs text-blue-300/80 -translate-y-6">
                                {/* Información de vuelo intermedio en hover/desktop */}
                                <span className="bg-[#0f111a] px-2 rounded-full border border-blue-500/30 shadow-lg relative z-20 py-0.5 text-[10px]">
                                   {criterio === 'costo' ? `$${currentRoute.vuelos[i].cost}` : `${currentRoute.vuelos[i].time} Hrs`}
                                </span>
                             </div>
                           )}
                         </div>
                       ))}
                    </div>

                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                       <div className="bg-gradient-to-br from-indigo-900/40 to-[#0f111a] border border-indigo-500/20 p-5 rounded-2xl flex items-center justify-between hover:border-indigo-500/50 transition-colors">
                         <div className="flex flex-col">
                           <span className="text-gray-400 font-bold uppercase text-[10px] tracking-wider mb-1">Costo Total Estimado</span>
                           <span className="text-xs text-indigo-300/80">{clase === 'vip' ? 'Primera Clase' : 'Clase Regular'}</span>
                         </div>
                         <span className="text-3xl font-black text-transparent bg-clip-text bg-gradient-to-r from-indigo-400 to-purple-400">
                           ${currentRoute.costo}
                         </span>
                       </div>

                       <div className="bg-gradient-to-br from-emerald-900/40 to-[#0f111a] border border-emerald-500/20 p-5 rounded-2xl flex items-center justify-between hover:border-emerald-500/50 transition-colors">
                         <div className="flex flex-col">
                           <span className="text-gray-400 font-bold uppercase text-[10px] tracking-wider mb-1">Tiempo de Viaje Estimado</span>
                           <span className="text-xs text-emerald-300/80">Incluyendo escalas</span>
                         </div>
                         <span className="text-3xl font-black text-transparent bg-clip-text bg-gradient-to-r from-emerald-400 to-teal-400">
                           {currentRoute.tiempo} Hrs
                         </span>
                       </div>
                    </div>
                 </div>
               )}
             </>
           )}
        </div>
      )}
    </div>
  );
}
