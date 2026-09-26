"use client";

import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { Info, Loader2, ArrowRight, X, Search, Plane, Plus, CalendarClock, Armchair, BarChart3 } from "lucide-react";
import Link from "next/link";
import { FlightStatusBadge, PageHeader, EmptyState, RouteCodes } from "@/components/ui";
import { formatFlightLocalTime } from "@/lib/flightTime";
import { useLanguage } from "@/context/LanguageContext";

const flightStates: Record<number, string> = {
  1: "Programado", 2: "Embarcando", 3: "Despegó",
  4: "En vuelo", 5: "Aterrizó", 6: "Llegó", 7: "Cancelado", 8: "Retrasado",
};

const cityLabel = (ciudades: any[], id: number) => ciudades.find((c: any) => c.id === id);

export default function Vuelos() {
  const { language } = useLanguage();
  const [vuelos, setVuelos] = useState<any[]>([]);
  const [totalVuelos, setTotalVuelos] = useState(0);
  const [catalogCounts, setCatalogCounts] = useState({ imported: 0, demo: 0 });
  const [scope, setScope] = useState<"all" | "upcoming">("all");
  const [flightIDInput, setFlightIDInput] = useState("");
  const [flightIDFilter, setFlightIDFilter] = useState("");
  const [originFilter, setOriginFilter] = useState("");
  const [destinationFilter, setDestinationFilter] = useState("");
  const [currentPage, setCurrentPage] = useState(1);
  const [itemsPerPage, setItemsPerPage] = useState(25);
  const [pageJump, setPageJump] = useState("");
  const [listError, setListError] = useState<string | null>(null);
  const requestID = useRef(0);
  const [ciudades, setCiudades] = useState<any[]>([]);
  const [aviones, setAviones] = useState<any[]>([]);
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
    const thisRequest = ++requestID.current;
    setLoading(true);
    setListError(null);
    try {
      const countryData = JSON.parse(localStorage.getItem("airres-country") || "{}");
      const countryHeaders = {
        "X-User-Country": countryData.name || "Estados Unidos",
        "X-Region": countryData.region || "America"
      };
      
      const params = new URLSearchParams({
        view: "page", scope, limit: String(itemsPerPage),
        offset: String((currentPage - 1) * itemsPerPage),
      });
      if (flightIDFilter) params.set("id", flightIDFilter);
      if (originFilter) params.set("origin", originFilter);
      if (destinationFilter) params.set("destination", destinationFilter);
      const [res, resCiudades, resAviones, resMatrix, resPrecios] = await Promise.all([
        fetch(`/api/vuelos?${params}`, { headers: countryHeaders }),
        fetch("/api/ciudades", { headers: countryHeaders }),
        fetch("/api/aviones", { headers: countryHeaders }),
        fetch("/api/tiempos", { headers: countryHeaders }),
        fetch("/api/precios", { headers: countryHeaders }),
      ]);
      if (!res.ok) throw new Error("No se pudo consultar el catálogo de vuelos.");
      const page = await res.json();
      if (thisRequest !== requestID.current) return;
      setVuelos(page.items || []);
      setTotalVuelos(page.total || 0);
      setCatalogCounts(page.catalog || { imported: 0, demo: 0 });
      if (resCiudades.ok) setCiudades(await resCiudades.json());
      if (resAviones.ok) setAviones(await resAviones.json());
      if (resMatrix.ok) setMatrix(await resMatrix.json());
      if (resPrecios.ok) setPrecios(await resPrecios.json());
    } catch (e) {
      if (thisRequest === requestID.current) setListError(e instanceof Error ? e.message : "No se pudo cargar la lista.");
    } finally {
      if (thisRequest === requestID.current) setLoading(false);
    }
  }

  useEffect(() => {
    fetchVuelos();
  }, [scope, currentPage, itemsPerPage, flightIDFilter, originFilter, destinationFilter]);

  // Utility to convert epoch to readable string
  const toDate = (epoch: number, cityID: number) => {
    if(!epoch) return "N/A";
    const city = ciudades.find((c: any) => c.id === cityID);
    return formatFlightLocalTime(epoch, city?.time_zone || "UTC", language);
  }

  const changeState = async (id: number, nextStateId: number) => {
     await fetch(`/api/vuelos/${id}/estado?id_estado=${nextStateId}`, {
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
           const economy = precios.matriz_precios_regular?.[orgCity.codigo]?.[dstCity.codigo];
           const first = precios.matriz_precios_vip?.[orgCity.codigo]?.[dstCity.codigo];
           if (org === dst || !((economy != null && economy > 0) || (first != null && first > 0))) {
             setErrorVuelo("🚫 Esta ruta no está permitida. Por favor selecciona un destino habilitado.");
             (document.getElementById("dst") as HTMLSelectElement).value = "";
             return false;
          }
       }
    }
    return true;
  };

  const [creationMode, setCreationMode] = useState<"direct" | "advanced">("direct");
  const [stopoverRoute, setStopoverRoute] = useState<number[]>([]);

  // Pagination calculations
  const totalPages = Math.ceil(totalVuelos / itemsPerPage) || 1;
  const startIndex = (currentPage - 1) * itemsPerPage;
  const paginatedVuelos = vuelos;

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
      const economy = precios?.matriz_precios_regular?.[originCode]?.[c.codigo];
      const first = precios?.matriz_precios_vip?.[originCode]?.[c.codigo];
      return val > 0 && ((economy != null && economy > 0) || (first != null && first > 0));
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

  const saveFlights = async () => {
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

                           const res = await fetch("/api/vuelos", {
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

                              const res = await fetch("/api/vuelos", {
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
                   };

  const selectedOrigin = selectedVuelo ? cityLabel(ciudades, selectedVuelo.id_origen) : null;
  const selectedDestination = selectedVuelo ? cityLabel(ciudades, selectedVuelo.id_destino) : null;
  const filtersActive = Boolean(flightIDFilter || originFilter || destinationFilter);

  return (
    <div className="fade-up">
      <PageHeader icon={Plane} eyebrow="Operaciones de vuelo" title="Catálogo de vuelos"
        subtitle="Consulta los vuelos importados del CSV y los vuelos de demostración. Los registros históricos también aparecen aquí."
        actions={<button onClick={() => { setShowAddModal(true); setErrorVuelo(null); setStopoverRoute([]); setCreationMode("direct"); }} className="btn-primary">
          <Plus className="h-4 w-4" aria-hidden="true" /> Nuevo Vuelo
        </button>} />

      {showAddModal && createPortal(
        <div role="dialog" aria-modal="true" aria-label="Programar vuelo" className="fixed inset-0 z-50 flex items-center justify-center overflow-y-auto bg-navy-950/60 p-4 backdrop-blur-sm">
          <div className="max-h-[90vh] w-full max-w-lg overflow-y-auto rounded-2xl bg-white shadow-2xl">
            <div className="flex items-start justify-between gap-3 bg-navy-900 px-6 py-5 text-white">
              <div>
                <p className="text-xs font-semibold uppercase tracking-[0.18em] text-gold-300">{currentRegion}</p>
                <h3 className="mt-1 text-xl font-bold text-white">Programar Vuelo</h3>
              </div>
              <button type="button" onClick={() => setShowAddModal(false)} aria-label="Cerrar" className="rounded-full p-1.5 text-navy-200 hover:bg-white/10 hover:text-white"><X className="h-5 w-5" /></button>
            </div>
            <div className="space-y-5 p-6">
              <div className="segmented" role="group" aria-label="Tipo de vuelo">
                <button type="button" data-active={creationMode === "direct"} onClick={() => { setCreationMode("direct"); setErrorVuelo(null); setStopoverRoute([]); }}>Vuelo Directo</button>
                <button type="button" data-active={creationMode === "advanced"} onClick={() => { setCreationMode("advanced"); setErrorVuelo(null); setStopoverRoute([]); }}>Avanzado (Escalas)</button>
              </div>

              {errorVuelo && <div role="alert" className="alert alert-error"><Info className="h-5 w-5 shrink-0" /><p>{errorVuelo}</p></div>}

              {creationMode === "direct" ? (
                <div className="grid gap-4 sm:grid-cols-2">
                  <div>
                    <label htmlFor="org" className="field-label">Origen</label>
                    <select id="org" className="field" onChange={validateRoute}>
                      <option value="">Seleccione Origen...</option>
                      {ciudades.map((c: any) => <option key={c.id} value={c.id}>{c.codigo} - {c.pais}</option>)}
                    </select>
                  </div>
                  <div>
                    <label htmlFor="dst" className="field-label">Destino</label>
                    <select id="dst" className="field" onChange={validateRoute}>
                      <option value="">Seleccione Destino...</option>
                      {ciudades.map((c: any) => <option key={c.id} value={c.id}>{c.codigo} - {c.pais}</option>)}
                    </select>
                  </div>
                  <div className="sm:col-span-2">
                    <label htmlFor="avion" className="field-label">Avión</label>
                    <select id="avion" className="field">
                      <option value="">Seleccione Avión...</option>
                      {aviones.map((a: any) => <option key={a.id} value={a.id}>ID {a.id} · {a.nombre} ({a.fabricante})</option>)}
                    </select>
                  </div>
                  <div className="sm:col-span-2">
                    <label htmlFor="fechaOut" className="field-label">Salida Programada (Fecha y Hora)</label>
                    <input type="datetime-local" id="fechaOut" className="field" />
                  </div>
                </div>
              ) : (
                <div className="space-y-4">
                  <div>
                    <label className="field-label">Paso 1: Ciudad de Inicio</label>
                    <select disabled={stopoverRoute.length > 0} value={stopoverRoute[0] || ""} className="field"
                      onChange={(e) => { const val = parseInt(e.target.value); if (val) setStopoverRoute([val]); else setStopoverRoute([]); }}>
                      <option value="">Seleccione Origen Inicial...</option>
                      {ciudades.map((c: any) => <option key={c.id} value={c.id}>{c.codigo} - {c.pais}</option>)}
                    </select>
                  </div>

                  {stopoverRoute.length > 0 && (
                    <div>
                      <label className="field-label">
                        Paso {stopoverRoute.length + 1}: Agregar Siguiente Tramo / Escala desde {cityLabel(ciudades, stopoverRoute[stopoverRoute.length - 1])?.codigo}
                      </label>
                      <select key={`next-dest-${stopoverRoute.length}`} className="field" onChange={(e) => { handleAddStopoverCity(e.target.value); e.target.value = ""; }}>
                        <option value="">+ Seleccionar Destino Conectado...</option>
                        {getValidNextDestinations(stopoverRoute[stopoverRoute.length - 1]).map((c: any) => <option key={c.id} value={c.id}>{c.codigo} - {c.pais}</option>)}
                      </select>
                    </div>
                  )}

                  {stopoverRoute.length > 0 && (
                    <div className="rounded-xl border border-slate-200 bg-slate-50 p-4">
                      <div className="flex items-center justify-between">
                        <span className="eyebrow">Ruta Programada ({stopoverRoute.length - 1} Escalas/Tramos)</span>
                        {stopoverRoute.length > 1 && <button type="button" onClick={handleRemoveLastStopover} className="text-xs font-semibold text-red-600 hover:underline">Quitar Último</button>}
                      </div>
                      <div className="mt-3 flex flex-wrap items-center gap-2">
                        {stopoverRoute.map((cId, idx) => <div key={cId} className="flex items-center gap-2">
                          <span className="rounded-lg bg-navy-900 px-2.5 py-1 text-xs font-bold text-white">{cityLabel(ciudades, cId)?.codigo || cId}</span>
                          {idx < stopoverRoute.length - 1 && <ArrowRight className="h-3.5 w-3.5 text-slate-400" />}
                        </div>)}
                      </div>
                    </div>
                  )}

                  <div>
                    <label htmlFor="avion" className="field-label">Avión</label>
                    <select id="avion" className="field">
                      <option value="">Seleccione Avión...</option>
                      {aviones.map((a: any) => <option key={a.id} value={a.id}>ID {a.id} · {a.nombre} ({a.fabricante})</option>)}
                    </select>
                  </div>
                  <div>
                    <label htmlFor="fechaOut" className="field-label">Salida Inicial (Fecha y Hora)</label>
                    <input type="datetime-local" id="fechaOut" className="field" />
                  </div>
                </div>
              )}
            </div>
            <div className="flex justify-end gap-3 border-t border-slate-100 bg-slate-50 px-6 py-4">
              <button onClick={() => setShowAddModal(false)} disabled={submitting} className="btn-ghost">Cancelar</button>
              <button id="saveBtn" disabled={submitting} onClick={saveFlights} className="btn-primary">
                {submitting && <Loader2 className="h-4 w-4 animate-spin" />}
                {submitting ? 'Guardando...' : 'Guardar Vuelo(s)'}
              </button>
            </div>
          </div>
        </div>, document.body
      )}

      {selectedVuelo && createPortal(
        <div role="dialog" aria-modal="true" aria-label="Detalle del Vuelo" className="fixed inset-0 z-50 flex items-center justify-center overflow-y-auto bg-navy-950/60 p-4 backdrop-blur-sm">
          <div className="w-full max-w-2xl overflow-hidden rounded-2xl bg-white shadow-2xl">
            <div className="relative bg-navy-900 px-6 py-6 text-white sm:px-8">
              <button onClick={() => setSelectedVuelo(null)} aria-label="Cerrar" className="absolute right-4 top-4 rounded-full p-1.5 text-navy-200 hover:bg-white/10 hover:text-white"><X className="h-5 w-5" /></button>
              <p className="text-xs font-semibold uppercase tracking-[0.18em] text-gold-300">Detalle del Vuelo · AP {selectedVuelo.id}</p>
              <div className="mt-4 flex items-center justify-between gap-4">
                <div><p className="text-4xl font-bold text-white">{selectedOrigin?.codigo || "???"}</p><p className="text-sm text-navy-200">{selectedOrigin?.pais || "Desconocido"}</p></div>
                <div className="flex flex-1 items-center gap-2 px-2" aria-hidden="true"><span className="h-px flex-1 border-t border-dashed border-navy-400" /><Plane className="h-5 w-5 rotate-45 text-gold-300" /><span className="h-px flex-1 border-t border-dashed border-navy-400" /></div>
                <div className="text-right"><p className="text-4xl font-bold text-white">{selectedDestination?.codigo || "???"}</p><p className="text-sm text-navy-200">{selectedDestination?.pais || "Desconocido"}</p></div>
              </div>
            </div>
            <div className="grid gap-4 p-6 sm:grid-cols-2 sm:p-8">
              <div className="rounded-xl border border-slate-200 p-4">
                <p className="eyebrow mb-3">Horario local de cada aeropuerto</p>
                <p className="text-xs text-slate-500">Salida Programada</p>
                <p className="font-semibold text-navy-900">{toDate(selectedVuelo.salida_programada, selectedVuelo.id_origen)}</p>
                <p className="mt-3 text-xs text-slate-500">Llegada Estimada</p>
                <p className="font-semibold text-navy-900">{toDate(selectedVuelo.llegada_programada, selectedVuelo.id_destino)}</p>
              </div>
              <div className="space-y-4">
                <div className="rounded-xl border border-slate-200 p-4">
                  <p className="eyebrow mb-2">Aeronave</p>
                  <p className="font-semibold text-navy-900">{aviones.find((a: any) => a.id === selectedVuelo.id_avion)?.nombre || "No asignado"}</p>
                  <p className="text-xs text-slate-500">{aviones.find((a: any) => a.id === selectedVuelo.id_avion)?.fabricante || "Fabricante desconocido"}</p>
                </div>
                <div className="rounded-xl border border-gold-200 bg-gold-50 p-4">
                  <p className="eyebrow mb-2">Precios Sugeridos</p>
                  <div className="flex items-center justify-between">
                    <div><p className="text-xs text-slate-500">Regular</p><p className="text-lg font-bold text-navy-900">${precios?.matriz_precios_regular?.[selectedOrigin?.codigo]?.[selectedDestination?.codigo] || "N/A"}</p></div>
                    <div className="text-right"><p className="text-xs text-slate-500">VIP</p><p className="text-lg font-bold text-gold-700">${precios?.matriz_precios_vip?.[selectedOrigin?.codigo]?.[selectedDestination?.codigo] || "N/A"}</p></div>
                  </div>
                </div>
              </div>
            </div>
            <div className="flex flex-wrap items-center gap-3 border-t border-slate-100 bg-slate-50 px-6 py-4 sm:px-8">
              <FlightStatusBadge state={selectedVuelo.id_estado_vuelo} />
              <Link href={`/dashboard/vuelos/${selectedVuelo.id}`} className="btn-ghost btn-sm"><BarChart3 className="h-4 w-4" /> Ver panel del vuelo</Link>
              <div className="ml-auto flex flex-wrap gap-2">
                <button onClick={() => setSelectedVuelo(null)} className="btn-secondary">Cerrar</button>
                {selectedVuelo.id_estado_vuelo < 6 && <button onClick={() => { changeState(selectedVuelo.id, selectedVuelo.id_estado_vuelo + 1); setSelectedVuelo(null); }} className="btn-primary">
                  Cambiar estado a «{flightStates[selectedVuelo.id_estado_vuelo + 1]}»
                </button>}
              </div>
            </div>
          </div>
        </div>, document.body
      )}

      <div className="mb-6 grid gap-4 md:grid-cols-3">
        <div className="card card-body bg-navy-900 text-white">
          <p className="text-sm text-navy-200">{filtersActive ? "Vuelos que coinciden" : scope === "all" ? "Vuelos registrados" : "Vuelos próximos"}</p>
          <p className="mt-1 text-3xl font-bold text-white">{totalVuelos.toLocaleString("es-BO")}</p>
          <p className="mt-1 text-xs text-navy-200">{filtersActive ? "Resultado de los filtros actuales." : scope === "all" ? "CSV histórico y vuelos de demostración." : "Solo salidas futuras; los históricos están en Todos."}</p>
        </div>
        <div className="card card-body">
          <p className="text-sm text-slate-500">Importados del CSV</p>
          <p className="mt-1 text-3xl font-bold text-navy-900">{catalogCounts.imported.toLocaleString("es-BO")}</p>
          <p className="mt-1 text-xs text-slate-500">Registros válidos conservados, incluso históricos.</p>
        </div>
        <div className="card card-body">
          <p className="text-sm text-slate-500">Vuelos de demostración</p>
          <p className="mt-1 text-3xl font-bold text-navy-900">{catalogCounts.demo.toLocaleString("es-BO")}</p>
          <p className="mt-1 text-xs text-slate-500">Programados para poder probar compras.</p>
        </div>
      </div>

      <section className="card">
        <div className="space-y-4 border-b border-slate-100 p-5 sm:p-6">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div className="segmented w-auto" role="group" aria-label="Tipo de vuelos">
              <button type="button" data-active={scope === "all"} onClick={() => { setScope("all"); setCurrentPage(1); }}>Todos los vuelos</button>
              <button type="button" data-active={scope === "upcoming"} onClick={() => { setScope("upcoming"); setCurrentPage(1); }}>Solo próximos</button>
            </div>
            <p className="flex items-center gap-1.5 text-xs text-slate-500"><CalendarClock className="h-4 w-4" aria-hidden="true" />Horario local de cada aeropuerto</p>
          </div>
          <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_minmax(0,1fr)_auto]">
            <form className="flex min-w-0 gap-2" onSubmit={(event) => { event.preventDefault(); setFlightIDFilter(flightIDInput.trim()); setCurrentPage(1); }}>
              <input type="number" min="1" value={flightIDInput} onChange={(event) => setFlightIDInput(event.target.value)} placeholder="ID del vuelo" aria-label="Buscar por ID de vuelo" className="field min-w-0 flex-1" />
              <button type="submit" className="btn-primary px-3" aria-label="Buscar vuelo"><Search className="h-4 w-4" /></button>
            </form>
            <select value={originFilter} onChange={(event) => { setOriginFilter(event.target.value); setCurrentPage(1); }} aria-label="Filtrar por origen" className="field">
              <option value="">Todos los orígenes</option>
              {ciudades.map((city: any) => <option key={city.id} value={city.id}>{city.codigo} · {city.pais}</option>)}
            </select>
            <select value={destinationFilter} onChange={(event) => { setDestinationFilter(event.target.value); setCurrentPage(1); }} aria-label="Filtrar por destino" className="field">
              <option value="">Todos los destinos</option>
              {ciudades.map((city: any) => <option key={city.id} value={city.id}>{city.codigo} · {city.pais}</option>)}
            </select>
            <button type="button" onClick={() => { setFlightIDInput(""); setFlightIDFilter(""); setOriginFilter(""); setDestinationFilter(""); setCurrentPage(1); }} className="btn-secondary">Limpiar</button>
          </div>
        </div>

        {listError && <div role="alert" className="alert alert-error m-5">{listError} <button onClick={fetchVuelos} className="font-semibold underline">Reintentar</button></div>}
        {loading ? (
          <div className="flex justify-center p-12"><Loader2 className="h-8 w-8 animate-spin text-navy-500" /></div>
        ) : totalVuelos === 0 ? (
          <div className="p-5"><EmptyState icon={Plane} title="Sin resultados">No hay vuelos que coincidan con estos filtros. Prueba «Todos los vuelos» o limpia la búsqueda.</EmptyState></div>
        ) : (
          <>
            <div className="space-y-3 p-4 xl:hidden">
              {paginatedVuelos.map((v: any) => <article key={v.id} className="rounded-xl border border-slate-200 p-4">
                <div className="flex flex-wrap items-start justify-between gap-2">
                  <div><p className="text-xs font-semibold text-slate-500">Vuelo AP {v.id}</p><RouteCodes from={cityLabel(ciudades, v.id_origen)?.codigo} to={cityLabel(ciudades, v.id_destino)?.codigo} /></div>
                  <FlightStatusBadge state={v.id_estado_vuelo} />
                </div>
                <div className="mt-3 grid gap-1 text-sm text-slate-600">
                  <p><span className="text-slate-400">Sale:</span> {toDate(v.salida_programada, v.id_origen)}</p>
                  <p><span className="text-slate-400">Llega:</span> {toDate(v.llegada_programada, v.id_destino)}</p>
                </div>
                <div className="mt-3 flex items-center justify-between gap-2 text-xs text-slate-500">
                  <span>{v.demo ? "Demostración" : "CSV importado"}</span>
                  <button onClick={() => setSelectedVuelo(v)} className="btn-secondary btn-sm">Ver detalle</button>
                </div>
              </article>)}
            </div>
            <div className="hidden overflow-x-auto xl:block">
              <table className="table-clean">
                <thead>
                  <tr>
                    <th>Vuelo</th><th>Ruta</th><th>Estado</th><th>Salida local</th><th>Llegada local</th><th>Origen de datos</th><th className="text-right">Detalle</th>
                  </tr>
                </thead>
                <tbody>
                  {paginatedVuelos.map((v: any) => <tr key={v.id}>
                    <td className="whitespace-nowrap font-mono font-semibold text-navy-900">AP {v.id}</td>
                    <td className="whitespace-nowrap"><RouteCodes from={cityLabel(ciudades, v.id_origen)?.codigo} to={cityLabel(ciudades, v.id_destino)?.codigo} /></td>
                    <td><FlightStatusBadge state={v.id_estado_vuelo} /></td>
                    <td>{toDate(v.salida_programada, v.id_origen)}</td>
                    <td>{toDate(v.llegada_programada, v.id_destino)}</td>
                    <td className="text-slate-500">{v.demo ? "Demostración" : "CSV importado"}</td>
                    <td className="text-right">
                      <div className="flex justify-end gap-2">
                        <Link href={`/boletos?vuelo=${v.id}`} className="btn-ghost btn-sm" aria-label={`Comprar en el vuelo ${v.id}`}><Armchair className="h-4 w-4" /></Link>
                        <button onClick={() => setSelectedVuelo(v)} aria-label={`Ver detalle del vuelo ${v.id}`} className="btn-secondary btn-sm">Ver detalle</button>
                      </div>
                    </td>
                  </tr>)}
                </tbody>
              </table>
            </div>

            <div className="flex flex-wrap items-center justify-between gap-4 border-t border-slate-100 p-4 text-xs text-slate-500">
              <div className="flex flex-wrap items-center gap-2">
                <span>Mostrando {startIndex + 1}–{Math.min(startIndex + itemsPerPage, totalVuelos)} de {totalVuelos.toLocaleString("es-BO")} vuelos</span>
                <span className="text-slate-300">|</span>
                <span>Filas por página:</span>
                <select value={itemsPerPage} onChange={(e) => { setItemsPerPage(Number(e.target.value)); setCurrentPage(1); }} className="rounded-lg border border-slate-300 bg-white px-2 py-1 text-slate-700">
                  <option value={25}>25</option><option value={50}>50</option><option value={100}>100</option>
                </select>
              </div>
              <div className="flex flex-wrap items-center gap-2">
                <button onClick={() => handlePageChange(currentPage - 1)} disabled={currentPage === 1} className="btn-secondary btn-sm">Anterior</button>
                <div className="flex items-center gap-1">
                  {Array.from({ length: totalPages }, (_, i) => i + 1)
                    .filter(p => p === 1 || p === totalPages || Math.abs(p - currentPage) <= 1)
                    .map((p, idx, arr) => <div key={p} className="flex items-center gap-1">
                      {idx > 0 && p - arr[idx - 1] > 1 && <span className="px-1 text-slate-400">...</span>}
                      <button onClick={() => handlePageChange(p)} aria-current={currentPage === p ? "page" : undefined}
                        className={`h-8 min-w-8 rounded-lg px-2 font-semibold transition ${currentPage === p ? "bg-navy-900 text-white" : "text-slate-600 hover:bg-slate-100"}`}>{p}</button>
                    </div>)}
                </div>
                <button onClick={() => handlePageChange(currentPage + 1)} disabled={currentPage === totalPages} className="btn-secondary btn-sm">Siguiente</button>
                <form onSubmit={(event) => { event.preventDefault(); handlePageChange(Number(pageJump)); setPageJump(""); }} className="flex items-center gap-2">
                  <label htmlFor="jump-to-page">Ir a página</label>
                  <input id="jump-to-page" type="number" min="1" max={totalPages} value={pageJump} onChange={(event) => setPageJump(event.target.value)} className="w-20 rounded-lg border border-slate-300 px-2 py-1.5 text-slate-700" />
                  <button type="submit" className="btn-secondary btn-sm">Ir</button>
                </form>
              </div>
            </div>
          </>
        )}
      </section>
    </div>
  );
}
