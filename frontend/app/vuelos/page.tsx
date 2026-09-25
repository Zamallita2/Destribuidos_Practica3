"use client";

import { useEffect, useState } from "react";
import { Ticket, Info, Loader2, ArrowRight, X } from "lucide-react";

export default function Vuelos() {
  const [vuelos, setVuelos] = useState([]);
  const [ciudades, setCiudades] = useState([]);
  const [aviones, setAviones] = useState([]);
  const [matrix, setMatrix] = useState<any>(null);
  const [precios, setPrecios] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [showAddModal, setShowAddModal] = useState(false);
  const [errorVuelo, setErrorVuelo] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [selectedVuelo, setSelectedVuelo] = useState<any>(null);
  const [currentRegion, setCurrentRegion] = useState("");

  useEffect(() => {
    const country = JSON.parse(localStorage.getItem("airres-country") || "{}");
    setCurrentRegion(country.region || "América (Global)");
  }, []);

  const fetchVuelos = async () => {
    try {
      const countryData = JSON.parse(localStorage.getItem("airres-country") || "{}");
      const countryHeaders = {
        "X-User-Country": countryData.name || "Estados Unidos",
        "X-Region": countryData.region || "America"
      };
      
      const res = await fetch("http://localhost:8080/api/vuelos", { headers: countryHeaders });
      if (res.ok) setVuelos(await res.json());

      const resCiudades = await fetch("http://localhost:8080/api/ciudades", { headers: countryHeaders });
      if (resCiudades.ok) setCiudades(await resCiudades.json());

      const resAviones = await fetch("http://localhost:8080/api/aviones", { headers: countryHeaders });
      if (resAviones.ok) setAviones(await resAviones.json());

      const resMatrix = await fetch("http://localhost:8080/api/tiempos", { headers: countryHeaders });
      if (resMatrix.ok) setMatrix(await resMatrix.json());

      const resPrecios = await fetch("http://localhost:8080/api/precios", { headers: countryHeaders });
      if (resPrecios.ok) setPrecios(await resPrecios.json());
      
    } catch(e) {
      console.error(e);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    fetchVuelos();
  }, []);

  // Utility to convert epoch to readable string
  const toDate = (epoch: number) => {
    if(!epoch) return "N/A";
    const d = new Date(epoch * 1000);
    return d.toLocaleString("es-ES", { day:"2-digit", month:"short", hour:"2-digit", minute:"2-digit" });
  }

  const changeState = async (id: number, nextStateId: number) => {
     await fetch(`http://localhost:8080/api/vuelos/${id}/estado?id_estado=${nextStateId}`, { 
       method: "PUT",
       headers: {
        "X-User-Country": JSON.parse(localStorage.getItem("airres-country") || "{}").name || "Estados Unidos",
        "X-Region": JSON.parse(localStorage.getItem("airres-country") || "{}").region || "America"
       }
     });
     fetchVuelos();
  }

  const validateRoute = () => {
    setErrorVuelo(null);
    const org = (document.getElementById("org") as HTMLSelectElement).value;
    const dst = (document.getElementById("dst") as HTMLSelectElement).value;
    if (org && dst && precios) {
       const orgCity: any = ciudades.find((c: any) => c.id === parseInt(org));
       const dstCity: any = ciudades.find((c: any) => c.id === parseInt(dst));
       
       if (orgCity && dstCity) {
          const price = precios.matriz_precios_regular?.[orgCity.codigo]?.[dstCity.codigo];
          if (org === dst || price === null || price === undefined) {
             setErrorVuelo("🚫 Esta ruta no está permitida. Por favor selecciona un destino habilitado.");
             (document.getElementById("dst") as HTMLSelectElement).value = "";
             return false;
          }
       }
    }
    return true;
  };

  const [currentPage, setCurrentPage] = useState(1);
  const [itemsPerPage, setItemsPerPage] = useState(10);
  const [creationMode, setCreationMode] = useState<"direct" | "advanced">("direct");
  const [stopoverRoute, setStopoverRoute] = useState<number[]>([]);

  // Pagination calculations
  const totalPages = Math.ceil(vuelos.length / itemsPerPage) || 1;
  const startIndex = (currentPage - 1) * itemsPerPage;
  const paginatedVuelos = vuelos.slice(startIndex, startIndex + itemsPerPage);

  const handlePageChange = (page: number) => {
    if (page >= 1 && page <= totalPages) {
      setCurrentPage(page);
    }
  };

  const getValidNextDestinations = (currentCityId: number) => {
    if (!currentCityId || !ciudades.length || !matrix) return [];
    const currentCity: any = ciudades.find((c: any) => c.id === currentCityId);
    if (!currentCity) return [];

    const originCode = currentCity.codigo;
    const destCodes = matrix[originCode];
    if (!destCodes) return [];

    return ciudades.filter((c: any) => {
      if (c.id === currentCityId) return false;
      const val = destCodes[c.codigo];
      return val !== null && val !== undefined && val > 0;
    });
  };

  const handleAddStopoverCity = (cityIdStr: string) => {
    const cityId = parseInt(cityIdStr);
    if (!cityId) return;
    setStopoverRoute((prev) => [...prev, cityId]);
  };

  const handleRemoveLastStopover = () => {
    setStopoverRoute((prev) => prev.slice(0, -1));
  };

  return (
    <div className="animate-in fade-in slide-in-from-bottom-5 duration-500">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h2 className="text-3xl font-bold bg-clip-text text-transparent bg-gradient-to-r from-blue-400 to-purple-400 flex items-center gap-3">
            <Ticket className="w-8 h-8 text-blue-500" />
            Gestión de Vuelos
          </h2>
          <p className="text-gray-400 mt-2">Control total sobre cronograma, embarque y despegue.</p>
        </div>
        <button onClick={() => { fetchVuelos(); setShowAddModal(true); setErrorVuelo(null); setStopoverRoute([]); setCreationMode("direct"); }} className="btn-primary flex justify-center gap-2 items-center">
            Nuevo Vuelo
        </button>
      </div>

      <div className="glass-panel p-6">
        {/* Modals and forms */}
        {showAddModal && (
          <div className="fixed inset-0 bg-black/60 z-50 flex items-center justify-center p-4 animate-in fade-in">
             <div className="bg-[#1a1d2d] border border-gray-700/50 p-6 rounded-2xl w-full max-w-lg shadow-2xl relative overflow-hidden max-h-[90vh] overflow-y-auto">
                <div className="absolute top-0 left-0 w-full h-1 bg-gradient-to-r from-blue-500 to-purple-500" />
                
                <div className="flex justify-between items-center mb-4">
                    <h3 className="text-2xl font-bold font-heading text-white">Programar Vuelo</h3>
                    <div className="px-2 py-0.5 rounded bg-blue-500/10 border border-blue-500/20 text-[10px] text-blue-400 font-bold uppercase tracking-wider">{currentRegion}</div>
                 </div>

                {/* MODE TOGGLE */}
                <div className="flex bg-white/5 p-1 rounded-xl mb-6 border border-white/10">
                   <button 
                     type="button"
                     onClick={() => { setCreationMode("direct"); setErrorVuelo(null); setStopoverRoute([]); }}
                     className={`flex-1 py-1.5 text-xs font-bold rounded-lg transition ${creationMode === "direct" ? "bg-blue-600 text-white" : "text-gray-400 hover:text-white"}`}
                   >
                     Vuelo Directo
                   </button>
                   <button 
                     type="button"
                     onClick={() => { setCreationMode("advanced"); setErrorVuelo(null); setStopoverRoute([]); }}
                     className={`flex-1 py-1.5 text-xs font-bold rounded-lg transition ${creationMode === "advanced" ? "bg-purple-600 text-white" : "text-gray-400 hover:text-white"}`}
                   >
                     Avanzado (Escalas)
                   </button>
                </div>
                
                {errorVuelo && (
                  <div className="mb-6 p-4 bg-red-500/10 border border-red-500/30 rounded-xl flex items-center gap-3 text-red-400 text-sm animate-in zoom-in slide-in-from-top-2 duration-300">
                     <Info className="w-5 h-5 shrink-0" />
                     <p>{errorVuelo}</p>
                  </div>
                )}

                {creationMode === "direct" ? (
                  <div className="space-y-4">
                     <div>
                       <label className="text-sm font-semibold text-gray-400">Origen</label>
                       <select 
                          id="org" 
                          className="w-full mt-1 bg-white/5 p-2 rounded border border-gray-700 text-white outline-none"
                          onChange={validateRoute}
                        >
                          <option value="">Seleccione Origen...</option>
                          {ciudades.map((c: any) => (
                             <option key={c.id} value={c.id} className="text-black">{c.codigo} - {c.pais}</option>
                          ))}
                       </select>
                     </div>
                     <div>
                       <label className="text-sm font-semibold text-gray-400">Destino</label>
                      <select 
                          id="dst" 
                          className="w-full mt-1 bg-white/5 p-2 rounded border border-gray-700 text-white outline-none"
                          onChange={validateRoute}
                        >
                           <option value="">Seleccione Destino...</option>
                           {ciudades.map((c: any) => (
                              <option key={c.id} value={c.id} className="text-black">{c.codigo} - {c.pais}</option>
                           ))}
                        </select>
                     </div>
                     <div>
                       <label className="text-sm font-semibold text-gray-400">Avión</label>
                       <select id="avion" className="w-full mt-1 bg-white/5 p-2 rounded border border-gray-700 text-white outline-none">
                          <option value="">Seleccione Avión...</option>
                          {aviones.map((a: any) => (
                             <option key={a.id} value={a.id} className="text-black">{a.nombre} ({a.fabricante})</option>
                          ))}
                       </select>
                     </div>
                     <div>
                       <label className="text-sm font-semibold text-gray-400">Salida Programada (Fecha y Hora)</label>
                       <input type="datetime-local" id="fechaOut" className="w-full mt-1 bg-white/5 p-2 rounded border border-gray-700 text-white" />
                     </div>
                  </div>
                ) : (
                  <div className="space-y-4">
                     {/* ADVANCED STOPOVER MODE */}
                     <div>
                        <label className="text-sm font-semibold text-purple-400">Paso 1: Ciudad de Inicio</label>
                        <select
                          disabled={stopoverRoute.length > 0}
                          onChange={(e) => {
                            const val = parseInt(e.target.value);
                            if (val) setStopoverRoute([val]);
                            else setStopoverRoute([]);
                          }}
                          value={stopoverRoute[0] || ""}
                          className="w-full mt-1 bg-white/5 p-2 rounded border border-purple-500/40 text-white outline-none disabled:opacity-60"
                        >
                          <option value="">Seleccione Origen Inicial...</option>
                          {ciudades.map((c: any) => (
                             <option key={c.id} value={c.id} className="text-black">{c.codigo} - {c.pais}</option>
                          ))}
                        </select>
                     </div>

                     {stopoverRoute.length > 0 && (
                        <div>
                           <label className="text-sm font-semibold text-blue-400">
                             Paso {stopoverRoute.length + 1}: Agregar Siguiente Tramo / Escala desde {(ciudades.find((c: any) => c.id === stopoverRoute[stopoverRoute.length - 1]) as any)?.codigo}
                           </label>
                           <select
                              key={`next-dest-${stopoverRoute.length}`}
                              onChange={(e) => {
                                handleAddStopoverCity(e.target.value);
                                e.target.value = "";
                              }}
                              className="w-full mt-1 bg-white/5 p-2 rounded border border-blue-500/40 text-white outline-none"
                           >
                              <option value="">+ Seleccionar Destino Conectado...</option>
                              {getValidNextDestinations(stopoverRoute[stopoverRoute.length - 1]).map((c: any) => (
                                 <option key={c.id} value={c.id} className="text-black">{c.codigo} - {c.pais}</option>
                              ))}
                           </select>
                        </div>
                     )}

                     {/* STOPOVER ROUTE PREVIEW */}
                     {stopoverRoute.length > 0 && (
                        <div className="p-4 bg-white/5 rounded-xl border border-white/10 space-y-2">
                           <div className="flex justify-between items-center">
                              <span className="text-xs font-bold text-gray-400 uppercase tracking-wider">Ruta Programada ({stopoverRoute.length - 1} Escalas/Tramos)</span>
                              {stopoverRoute.length > 1 && (
                                <button type="button" onClick={handleRemoveLastStopover} className="text-xs text-red-400 hover:text-red-300">
                                  Quitar Último
                                </button>
                              )}
                           </div>
                           <div className="flex flex-wrap items-center gap-2 pt-2">
                              {stopoverRoute.map((cId, idx) => {
                                 const city: any = ciudades.find((c: any) => c.id === cId);
                                 return (
                                    <div key={cId} className="flex items-center gap-2">
                                       <span className="px-2.5 py-1 bg-purple-500/20 border border-purple-500/40 text-purple-300 rounded-lg text-xs font-bold">
                                          {city?.codigo || cId}
                                       </span>
                                       {idx < stopoverRoute.length - 1 && (
                                          <ArrowRight className="w-3.5 h-3.5 text-gray-500" />
                                       )}
                                    </div>
                                 );
                              })}
                           </div>
                        </div>
                     )}

                     <div>
                       <label className="text-sm font-semibold text-gray-400">Avión</label>
                       <select id="avion" className="w-full mt-1 bg-white/5 p-2 rounded border border-gray-700 text-white outline-none">
                          <option value="">Seleccione Avión...</option>
                          {aviones.map((a: any) => (
                             <option key={a.id} value={a.id} className="text-black">{a.nombre} ({a.fabricante})</option>
                          ))}
                       </select>
                     </div>

                     <div>
                       <label className="text-sm font-semibold text-gray-400">Salida Inicial (Fecha y Hora)</label>
                       <input type="datetime-local" id="fechaOut" className="w-full mt-1 bg-white/5 p-2 rounded border border-gray-700 text-white" />
                     </div>
                  </div>
                )}

                <div className="mt-6 flex justify-end gap-3">
                   <button onClick={() => setShowAddModal(false)} disabled={submitting} className="px-4 py-2 rounded text-gray-400 hover:bg-white/5 disabled:opacity-50">Cancelar</button>
                   <button id="saveBtn" disabled={submitting} onClick={async () => {
                      const avionId = parseInt((document.getElementById("avion") as HTMLInputElement).value);
                      const fechaStr = (document.getElementById("fechaOut") as HTMLInputElement).value;

                      if (creationMode === "direct") {
                         const origen = parseInt((document.getElementById("org") as HTMLInputElement).value);
                         const destino = parseInt((document.getElementById("dst") as HTMLInputElement).value);

                         if (!origen || !destino || !avionId || !fechaStr) {
                            setErrorVuelo("Por favor completa todos los campos del formulario.");
                            return;
                         }

                         if (!validateRoute()) return;
                         
                         setSubmitting(true);
                         try {
                           const orgCity: any = ciudades.find((c: any) => c.id === origen);
                           const dstCity: any = ciudades.find((c: any) => c.id === destino);
                           
                           let travelTimeHours = 0;
                           if (orgCity && dstCity && matrix && matrix[orgCity.codigo] && matrix[orgCity.codigo][dstCity.codigo] !== undefined) {
                              travelTimeHours = matrix[orgCity.codigo][dstCity.codigo];
                           } else {
                              setErrorVuelo("⚠️ Error: La ruta seleccionada no tiene un tiempo de viaje definido.");
                              setSubmitting(false);
                              return;
                           }

                           const epoch = Math.floor(new Date(fechaStr).getTime() / 1000);
                           const arrivalEpoch = epoch + (travelTimeHours * 3600);
                           const gateId = 1;

                           const res = await fetch("http://localhost:8080/api/vuelos", {
                              method: "POST",
                              headers: {
                                "Content-Type": "application/json",
                                "X-User-Country": JSON.parse(localStorage.getItem("airres-country") || "{}").name || "Estados Unidos",
                                "X-Region": JSON.parse(localStorage.getItem("airres-country") || "{}").region || "America"
                              },
                              body: JSON.stringify({
                                 id_origen: origen, id_destino: destino, id_avion: avionId, id_estado_vuelo: 1, id_puerta: gateId,
                                 salida_programada: epoch, llegada_programada: arrivalEpoch, 
                                 fecha_salida: epoch, fecha_llegada: arrivalEpoch
                              })
                           });

                           if (res.ok) {
                              setShowAddModal(false);
                              fetchVuelos();
                           } else {
                              const errData = await res.json();
                              setErrorVuelo(`Error del servidor: ${errData.error || 'No se pudo crear el vuelo'}`);
                           }
                         } catch (e) {
                            setErrorVuelo("Error de red. Asegúrate de que el servidor esté corriendo.");
                         } finally {
                           setSubmitting(false);
                         }
                      } else {
                         // ADVANCED STOPOVER MULTI-FLIGHT CREATION
                         if (stopoverRoute.length < 2 || !avionId || !fechaStr) {
                            setErrorVuelo("Por favor selecciona al menos un origen y un destino final con escalas, además del avión y la fecha.");
                            return;
                         }

                         setSubmitting(true);
                         try {
                           let currentEpoch = Math.floor(new Date(fechaStr).getTime() / 1000);
                           const countryData = JSON.parse(localStorage.getItem("airres-country") || "{}");
                           const headers = {
                             "Content-Type": "application/json",
                             "X-User-Country": countryData.name || "Estados Unidos",
                             "X-Region": countryData.region || "America"
                           };

                           for (let i = 0; i < stopoverRoute.length - 1; i++) {
                              const oId = stopoverRoute[i];
                              const dId = stopoverRoute[i + 1];

                              const oCity: any = ciudades.find((c: any) => c.id === oId);
                              const dCity: any = ciudades.find((c: any) => c.id === dId);
                              const travelHours = matrix?.[oCity.codigo]?.[dCity.codigo] || 2;

                              const arrivalEpoch = currentEpoch + (travelHours * 3600);

                              const res = await fetch("http://localhost:8080/api/vuelos", {
                                 method: "POST",
                                 headers,
                                 body: JSON.stringify({
                                    id_origen: oId, id_destino: dId, id_avion: avionId, id_estado_vuelo: 1, id_puerta: 1,
                                    salida_programada: currentEpoch, llegada_programada: arrivalEpoch,
                                    fecha_salida: currentEpoch, fecha_llegada: arrivalEpoch
                                 })
                              });

                              if (!res.ok) {
                                 const errData = await res.json();
                                 setErrorVuelo(`Error al crear el tramo ${oCity.codigo} -> ${dCity.codigo}: ${errData.error || 'Fallo de creación'}`);
                                 setSubmitting(false);
                                 return;
                              }

                              // 2 hours layover time before next flight segment
                              currentEpoch = arrivalEpoch + (2 * 3600);
                           }

                           setShowAddModal(false);
                           fetchVuelos();
                         } catch (e) {
                            setErrorVuelo("Error de red procesando las escalas.");
                         } finally {
                            setSubmitting(false);
                         }
                      }
                   }} className="px-4 py-2 bg-blue-600 hover:bg-blue-500 rounded text-white shadow-lg shadow-blue-500/20 disabled:opacity-50 flex items-center gap-2">
                      {submitting && <Loader2 className="w-4 h-4 animate-spin" />}
                      {submitting ? 'Guardando...' : 'Guardar Vuelo(s)'}
                   </button>
                </div>
             </div>
          </div>
        )}

        {selectedVuelo && (
          <div className="fixed inset-0 bg-black/70 z-50 flex items-center justify-center p-4 animate-in fade-in backdrop-blur-sm">
             <div className="bg-[#0f111a] border border-blue-500/30 p-8 rounded-3xl w-full max-w-2xl shadow-[0_0_50px_rgba(59,130,246,0.2)] relative overflow-hidden animate-in zoom-in duration-300">
                <div className="absolute top-0 left-0 w-full h-1.5 bg-gradient-to-r from-blue-600 via-purple-500 to-blue-400" />
                
                <div className="flex justify-between items-start mb-8">
                    <div>
                        <h3 className="text-3xl font-bold font-heading text-white flex items-center gap-3">
                            <Info className="text-blue-400 w-8 h-8" /> Detalle del Vuelo
                        </h3>
                        <p className="text-gray-500 font-mono mt-1">ID: VUELO-{selectedVuelo.id}</p>
                    </div>
                    <button onClick={() => setSelectedVuelo(null)} className="p-2 hover:bg-white/10 rounded-full transition text-gray-400">
                        <X className="w-6 h-6" />
                    </button>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
                    <div className="space-y-6">
                        <div className="bg-white/5 p-4 rounded-2xl border border-white/10">
                            <p className="text-[10px] uppercase text-blue-400 font-bold tracking-widest mb-4">Ruta del Vuelo</p>
                            <div className="flex items-center justify-between">
                                <div className="text-center">
                                    <p className="text-3xl font-bold text-white">{ciudades.find((c: any) => c.id === selectedVuelo.id_origen)?.codigo || "???"}</p>
                                    <p className="text-xs text-gray-500">{ciudades.find((c: any) => c.id === selectedVuelo.id_origen)?.pais || "Desconocido"}</p>
                                </div>
                                <div className="flex-1 flex flex-col items-center px-4">
                                    <div className="w-full h-px bg-blue-500/30 relative">
                                        <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-[#0f111a] px-2 text-blue-400">
                                            <ArrowRight className="w-4 h-4" />
                                        </div>
                                    </div>
                                </div>
                                <div className="text-center">
                                    <p className="text-3xl font-bold text-white">{ciudades.find((c: any) => c.id === selectedVuelo.id_destino)?.codigo || "???"}</p>
                                    <p className="text-xs text-gray-500">{ciudades.find((c: any) => c.id === selectedVuelo.id_destino)?.pais || "Desconocido"}</p>
                                </div>
                            </div>
                        </div>

                        <div className="bg-white/5 p-4 rounded-2xl border border-white/10">
                            <p className="text-[10px] uppercase text-purple-400 font-bold tracking-widest mb-3">Aeronave</p>
                            <p className="text-white font-bold">{aviones.find((a: any) => a.id === selectedVuelo.id_avion)?.nombre || "No asignado"}</p>
                            <p className="text-xs text-gray-500">{aviones.find((a: any) => a.id === selectedVuelo.id_avion)?.fabricante || "Fabricante desconocido"}</p>
                        </div>
                    </div>

                    <div className="space-y-6">
                        <div className="bg-white/5 p-4 rounded-2xl border border-white/10">
                            <p className="text-[10px] uppercase text-emerald-400 font-bold tracking-widest mb-4">Cronograma (Región {currentRegion})</p>
                            <div className="space-y-3">
                                <div>
                                    <p className="text-xs text-gray-500">Salida Programada</p>
                                    <p className="text-white font-semibold">{toDate(selectedVuelo.salida_programada)}</p>
                                </div>
                                <div>
                                    <p className="text-xs text-gray-500">Llegada Estimada</p>
                                    <p className="text-white font-semibold">{toDate(selectedVuelo.llegada_programada)}</p>
                                </div>
                            </div>
                        </div>

                        <div className="bg-white/5 p-4 rounded-2xl border border-white/10">
                            <p className="text-[10px] uppercase text-yellow-400 font-bold tracking-widest mb-3">Precios Sugeridos</p>
                            <div className="flex justify-between items-center">
                                <div>
                                    <p className="text-xs text-gray-500">Regular</p>
                                    <p className="text-lg font-bold text-emerald-400">
                                        ${precios?.matriz_precios_regular?.[ciudades.find((c: any) => c.id === selectedVuelo.id_origen)?.codigo]?.[ciudades.find((c: any) => c.id === selectedVuelo.id_destino)?.codigo] || "N/A"}
                                    </p>
                                </div>
                                <div>
                                    <p className="text-xs text-gray-500 text-right">VIP</p>
                                    <p className="text-lg font-bold text-yellow-400">
                                        ${precios?.matriz_precios_vip?.[ciudades.find((c: any) => c.id === selectedVuelo.id_origen)?.codigo]?.[ciudades.find((c: any) => c.id === selectedVuelo.id_destino)?.codigo] || "N/A"}
                                    </p>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>

                <div className="mt-10 flex justify-end items-center gap-4">
                    <div className="flex-1 flex gap-2">
                         <span className={`px-3 py-1.5 rounded-xl text-xs font-bold border ${selectedVuelo.id_estado_vuelo === 1 ? 'bg-gray-500/20 text-gray-400' : 'bg-emerald-500/20 text-emerald-400 border-emerald-500/30'}`}>
                            {selectedVuelo.id_estado_vuelo === 6 ? 'VUELO COMPLETADO' : 'EN PROCESO'}
                         </span>
                    </div>
                    <button onClick={() => setSelectedVuelo(null)} className="px-8 py-3 bg-white/5 hover:bg-white/10 border border-white/10 rounded-2xl font-bold transition">
                        Cerrar
                    </button>
                    <button 
                        onClick={() => {
                            if (selectedVuelo.id_estado_vuelo < 6) changeState(selectedVuelo.id, selectedVuelo.id_estado_vuelo + 1);
                            setSelectedVuelo(null);
                        }} 
                        className="px-8 py-3 bg-blue-600 hover:bg-blue-500 rounded-2xl font-bold shadow-lg shadow-blue-500/30 transition"
                    >
                        Siguiente Fase
                    </button>
                </div>
             </div>
          </div>
        )}

        <div className="w-full rounded-xl border border-white/10 overflow-hidden">
          {loading ? (
             <div className="p-10 flex justify-center"><Loader2 className="animate-spin text-blue-500" /></div>
          ) : vuelos.length === 0 ? (
             <div className="p-10 text-center text-gray-500">
               No hay vuelos registrados en la base de datos de esta región ({currentRegion}). 
               Asegúrate de estar en la región correcta o crea uno nuevo usando el botón "Nuevo Vuelo".
             </div>
          ) : (
            <>
              <table className="w-full text-sm text-left">
                <thead className="text-xs text-gray-400 uppercase bg-white/5 border-b border-white/10">
                  <tr>
                    <th className="px-6 py-4">ID Vuelo</th>
                    <th className="px-6 py-4">Status</th>
                    <th className="px-6 py-4">Región DB</th>
                    <th className="px-6 py-4">Salida Prog.</th>
                    <th className="px-6 py-4">Llegada Prog.</th>
                    <th className="px-6 py-4">Detalles</th>
                    <th className="px-6 py-4">Acción Estado</th>
                  </tr>
                </thead>
                <tbody>
                  {paginatedVuelos.map((v: any) => (
                    <tr key={v.id} className="border-b border-white/5 outline-none hover:bg-white/5 transition-colors">
                      <td className="px-6 py-4 font-semibold">VUELO-{v.id}</td>
                      <td className="px-6 py-4">
                        {v.id_estado_vuelo === 1 && <span className="px-2 py-1 rounded bg-gray-500/20 text-gray-400 border border-gray-500/30">SCHEDULED</span>}
                        {v.id_estado_vuelo === 2 && <span className="px-2 py-1 rounded bg-yellow-500/20 text-yellow-400 border border-yellow-500/50">BOARDING</span>}
                        {v.id_estado_vuelo === 3 && <span className="px-2 py-1 rounded bg-blue-500/20 text-blue-400 border border-blue-500/30">DEPARTED</span>}
                        {v.id_estado_vuelo === 4 && <span className="px-2 py-1 rounded bg-indigo-500/20 text-indigo-400 border border-indigo-500/30">IN_FLIGHT</span>}
                        {v.id_estado_vuelo === 5 && <span className="px-2 py-1 rounded bg-orange-500/20 text-orange-400 border border-orange-500/30">LANDED</span>}
                        {v.id_estado_vuelo === 6 && <span className="px-2 py-1 rounded bg-emerald-500/20 text-emerald-400 border border-emerald-500/50">ARRIVED</span>}
                      </td>
                      <td className="px-6 py-4">
                         <span className="text-[10px] text-gray-500 font-bold uppercase">{currentRegion}</span>
                      </td>
                      <td className="px-6 py-4 opacity-70">{toDate(v.salida_programada)}</td>
                      <td className="px-6 py-4 opacity-70">{toDate(v.llegada_programada)}</td>
                      <td className="px-6 py-4">
                         <button 
                           onClick={() => setSelectedVuelo(v)}
                           className="p-2 hover:bg-blue-500/20 rounded-lg text-blue-400 transition border border-transparent hover:border-blue-500/30"
                         >
                           <Info className="w-5 h-5"/>
                         </button>
                      </td>
                      <td className="px-6 py-4">
                         {v.id_estado_vuelo < 6 && (
                           <button onClick={() => changeState(v.id, v.id_estado_vuelo + 1)} className="text-xs bg-blue-600 hover:bg-blue-500 text-white px-3 py-1.5 rounded-lg flex gap-2 items-center transition shadow-[0_0_10px_rgba(59,130,246,0.5)]">
                             Siguiente Fase <ArrowRight className="w-3 h-3"/>
                           </button>
                         )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>

              {/* PAGINATION CONTROLS */}
              <div className="p-4 bg-white/5 border-t border-white/10 flex flex-wrap items-center justify-between gap-4">
                <div className="flex items-center gap-2 text-xs text-gray-400">
                  <span>Mostrando {startIndex + 1} - {Math.min(startIndex + itemsPerPage, vuelos.length)} de {vuelos.length} vuelos</span>
                  <span className="mx-2">|</span>
                  <span>Filas por página:</span>
                  <select 
                    value={itemsPerPage} 
                    onChange={(e) => { setItemsPerPage(Number(e.target.value)); setCurrentPage(1); }}
                    className="bg-white/5 border border-white/10 rounded px-2 py-1 text-white outline-none"
                  >
                    <option value={10} className="text-black">10</option>
                    <option value={20} className="text-black">20</option>
                    <option value={50} className="text-black">50</option>
                  </select>
                </div>

                <div className="flex items-center gap-2">
                  <button 
                    onClick={() => handlePageChange(currentPage - 1)}
                    disabled={currentPage === 1}
                    className="px-3 py-1.5 rounded bg-white/5 hover:bg-white/10 text-xs font-semibold text-white disabled:opacity-30 disabled:cursor-not-allowed transition"
                  >
                    Anterior
                  </button>

                  <div className="flex items-center gap-1">
                    {Array.from({ length: totalPages }, (_, i) => i + 1)
                      .filter(p => p === 1 || p === totalPages || Math.abs(p - currentPage) <= 1)
                      .map((p, idx, arr) => {
                        const showEllipsis = idx > 0 && p - arr[idx - 1] > 1;
                        return (
                          <div key={p} className="flex items-center gap-1">
                            {showEllipsis && <span className="text-gray-500 text-xs px-1">...</span>}
                            <button
                              onClick={() => handlePageChange(p)}
                              className={`px-3 py-1.5 rounded text-xs font-bold transition ${currentPage === p ? 'bg-blue-600 text-white' : 'bg-white/5 hover:bg-white/10 text-gray-400'}`}
                            >
                              {p}
                            </button>
                          </div>
                        );
                      })
                    }
                  </div>

                  <button 
                    onClick={() => handlePageChange(currentPage + 1)}
                    disabled={currentPage === totalPages}
                    className="px-3 py-1.5 rounded bg-white/5 hover:bg-white/10 text-xs font-semibold text-white disabled:opacity-30 disabled:cursor-not-allowed transition"
                  >
                    Siguiente
                  </button>
                </div>
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
