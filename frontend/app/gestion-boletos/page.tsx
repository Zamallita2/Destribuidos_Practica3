"use client";

import { useEffect, useState } from "react";
import {
  Ticket,
  Search,
  Loader2,
  ArrowRight,
  User,
  XCircle,
  CreditCard,
  ChevronDown,
  Plane,
  FileText,
} from "lucide-react";
import { downloadBoardingPassPdf } from "@/lib/boardingPassPdf";
import QRNetworkInfo from "@/components/QRNetworkInfo";

export default function GestionBoletos() {
  const [boletos, setBoletos] = useState<any[]>([]);
  const [vuelos, setVuelos] = useState<any[]>([]);
  const [ciudades, setCiudades] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedBoleto, setSelectedBoleto] = useState<any>(null);
  const [selectedPass, setSelectedPass] = useState<any>(null);
  const [updating, setUpdating] = useState(false);
  const [searchTerm, setSearchTerm] = useState("");

  const fetchData = async () => {
    try {
      const countryData = JSON.parse(localStorage.getItem("airres-country") || "{}");
      const countryHeaders = {
        "X-User-Country": countryData.name || "Estados Unidos",
        "X-Region": countryData.region || "America",
      };

      const [cRes, vRes, americanTickets, europeanTickets, lakeTickets] = await Promise.all([
        fetch("/api/ciudades", { headers: countryHeaders }).catch(() => null),
        fetch("/api/vuelos", { headers: countryHeaders }).catch(() => null),
        fetch("/api/boletos", { headers: { "X-Region": "America" } }).catch(() => null),
        fetch("/api/boletos", { headers: { "X-Region": "Europa" } }).catch(() => null),
        fetch("/api/boletos", { headers: { "X-Region": "Asia" } }).catch(() => null),
      ]);

      if (cRes?.ok) setCiudades(await cRes.json());
      if (vRes?.ok) setVuelos(await vRes.json());
      const allTickets = new Map<number, any>();
      for (const response of [americanTickets, europeanTickets, lakeTickets]) {
        if (!response?.ok) continue;
        for (const ticket of await response.json()) {
          if (!allTickets.has(ticket.id_boleto)) allTickets.set(ticket.id_boleto, ticket);
        }
      }
      const loadedTickets = Array.from(allTickets.values()).sort((a, b) => b.id_boleto - a.id_boleto);
      setBoletos(loadedTickets);
      const requestedID = Number(new URLSearchParams(window.location.search).get("boleto"));
      if (requestedID) setSelectedBoleto(loadedTickets.find((ticket) => ticket.id_boleto === requestedID) || null);
    } catch (e) {
      console.error(e);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  useEffect(() => {
    let cancelled = false;
    setSelectedPass(null);
    if (selectedBoleto?.estado !== "SALED") return;
    fetch(`/api/boletos/${selectedBoleto.id_boleto}/pase`)
      .then((response) => {
        if (!response.ok) throw new Error("Pase no disponible");
        return response.json();
      })
      .then((pass) => { if (!cancelled) setSelectedPass(pass); })
      .catch((error) => { if (!cancelled) console.error(error); });
    return () => { cancelled = true; };
  }, [selectedBoleto?.id_boleto, selectedBoleto?.estado]);

  const changeTicketState = async (boletoId: number, newState: string) => {
    if (!confirm(`¿Estás seguro que deseas cambiar el estado a ${newState}?`)) return;

    setUpdating(true);
    try {
      const countryData = JSON.parse(localStorage.getItem("airres-country") || "{}");
      const countryHeaders = {
        "X-User-Country": countryData.name || "Estados Unidos",
        "X-Region": countryData.region || "America",
        "Content-Type": "application/json",
      };

      const res = await fetch(`/api/boletos/${boletoId}/estado`, {
        method: "PATCH",
        headers: countryHeaders,
        body: JSON.stringify({ estado: newState }),
      });

      if (res.ok) {
        await fetchData();
        if (selectedBoleto?.id_boleto === boletoId) {
          setSelectedBoleto({ ...selectedBoleto, estado: newState });
        }
      } else {
        const d = await res.json();
        alert(d.error || "Hubo un error actualizando el estado");
      }
    } catch (e) {
      console.error(e);
    } finally {
      setUpdating(false);
    }
  };

  const getFlightDetails = (idVuelo: number) => {
    const v: any = vuelos.find((item: any) => item.id === idVuelo);
    if (!v) return null;

    const org = ciudades.find((c: any) => c.id === v.id_origen) as any;
    const dst = ciudades.find((c: any) => c.id === v.id_destino) as any;

    return { flight: v, org, dst };
  };

  const filterBoletos = () => {
    if (!searchTerm) return boletos;

    return boletos.filter(
      (b: any) =>
        b.nombre_pasajero?.toLowerCase().includes(searchTerm.toLowerCase()) ||
        String(b.id_boleto).includes(searchTerm) ||
        b.email_pasajero?.toLowerCase().includes(searchTerm.toLowerCase()) ||
        b.pasaporte?.toLowerCase().includes(searchTerm.toLowerCase())
    );
  };

  const statusColors = (estado: string) => {
    switch (estado) {
      case "RESERVED":
        return "bg-yellow-500/20 text-yellow-500 border-yellow-500/50";
      case "SALED":
        return "bg-emerald-500/20 text-emerald-400 border-emerald-500/50";
      case "ANNULLED":
        return "bg-red-500/20 text-red-500 border-red-500/50";
      default:
        return "bg-gray-500/20 text-gray-500 border-gray-500/50";
    }
  };

  const exportarBoletoPDF = async () => {
    if (!selectedBoleto) return;
    setUpdating(true);
    try {
      await downloadBoardingPassPdf(selectedBoleto);
    } catch (error) {
      console.error(error);
      alert("No se pudo descargar el boleto visual en PDF");
    } finally {
      setUpdating(false);
    }
  };

  if (loading) {
    return (
      <div className="h-[600px] flex items-center justify-center">
        <Loader2 className="w-12 h-12 text-blue-500 animate-spin" />
      </div>
    );
  }

  const filtered = filterBoletos();

  return (
    <div className="animate-in fade-in slide-in-from-bottom-5 duration-500 max-w-7xl mx-auto pb-20">
      <div className="mb-10 flex flex-col md:flex-row md:items-end justify-between gap-6">
        <div>
          <h2 className="text-4xl font-bold bg-clip-text text-transparent bg-gradient-to-r from-blue-400 to-emerald-400 flex items-center gap-3">
            <Ticket className="text-blue-400 w-10 h-10" />
            Gestión de Boletos
          </h2>
          <p className="text-gray-400 mt-2 text-lg">Consulta tus boletos, descarga el boleto visual o abre el pase en una billetera compatible.</p>
        </div>

        <div className="relative min-w-[300px]">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500 w-5 h-5" />
          <input
            type="text"
            placeholder="Buscar número de boleto, pasajero, email o pasaporte..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="w-full bg-white/5 border border-white/10 rounded-xl py-3 pl-11 pr-4 text-white placeholder:text-gray-600 focus:border-blue-500 outline-none transition"
          />
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        <div className="lg:col-span-2 glass-panel p-6 h-[650px] flex flex-col">
          <div className="flex justify-between items-center mb-6">
            <h3 className="font-bold text-xl uppercase tracking-wider text-gray-300">
              Todos los Boletos ({filtered.length})
            </h3>
            <button
              onClick={() => fetchData()}
              className="text-sm text-blue-400 hover:text-blue-300"
            >
              Refrescar
            </button>
          </div>

          <div className="overflow-y-auto flex-1 pr-2 space-y-4 scrollbar-thin scrollbar-thumb-white/10">
            {filtered.length === 0 ? (
              <div className="h-full flex items-center justify-center text-gray-500 italic border-dashed border-2 border-white/5 rounded-2xl">
                No se encontraron boletos.
              </div>
            ) : (
              filtered.map((b: any) => {
                const details = getFlightDetails(b.id_vuelo);

                return (
                  <div
                    key={b.id_boleto}
                    onClick={() => setSelectedBoleto(b)}
                    className={`p-4 rounded-2xl bg-white/5 border cursor-pointer hover:border-blue-500/50 hover:bg-white/10 transition-all ${
                      selectedBoleto?.id_boleto === b.id_boleto
                        ? "border-blue-500 bg-blue-500/10 shadow-[0_0_15px_rgba(59,130,246,0.15)]"
                        : "border-white/10"
                    }`}
                  >
                    <div className="flex justify-between items-start mb-3">
                      <div className="flex items-center gap-3">
                        <div className="w-10 h-10 rounded-full bg-gray-800 border border-gray-700 flex items-center justify-center">
                          <User className="text-gray-400 w-5 h-5" />
                        </div>
                        <div>
                          <p className="font-bold text-white text-lg">{b.nombre_pasajero}</p>
                          <p className="text-xs text-gray-400">{b.email_pasajero}</p>
                        </div>
                      </div>

                      <span className={`px-3 py-1 rounded-full text-xs font-bold border ${statusColors(b.estado)}`}>
                        {b.estado}
                      </span>
                    </div>

                    <div className="flex items-center gap-6 text-sm">
                      <div>
                        <p className="text-[10px] text-gray-500 font-bold uppercase">Boleto ID</p>
                        <p className="font-mono text-gray-300">#{b.id_boleto}</p>
                      </div>

                      {details && (
                        <div className="flex items-center gap-2">
                          <div className="text-gray-400 font-medium">{details.org?.codigo}</div>
                          <ArrowRight className="w-3 h-3 text-gray-600" />
                          <div className="text-gray-400 font-medium">{details.dst?.codigo}</div>
                        </div>
                      )}

                      <div>
                        <p className="text-[10px] text-gray-500 font-bold uppercase">Costo</p>
                        <p className="font-bold text-green-400">${b.costo}</p>
                      </div>
                    </div>
                  </div>
                );
              })
            )}
          </div>
        </div>

        <div className="lg:col-span-1 glass-panel p-6 h-[650px] sticky top-32 overflow-y-auto">
          {selectedBoleto ? (
            <div className="animate-in zoom-in duration-300 fade-in">
              <div className="flex flex-wrap justify-between items-center gap-3 mb-6 border-b border-white/10 pb-4">
                <h3 className="font-bold text-xl uppercase tracking-wider text-white">
                  Detalle del Boleto
                </h3>

                <div className="flex flex-wrap gap-2">
                {selectedBoleto.estado === "SALED" && <a
                  href={`/pase/${selectedBoleto.id_boleto}/billetera`}
                  className="flex items-center gap-2 rounded-lg border border-emerald-500/30 bg-emerald-600/20 px-4 py-2 text-sm font-bold text-emerald-300 hover:bg-emerald-600/30"
                >Abrir pase para Passbook</a>}
                {selectedBoleto.estado === "SALED" && <button
                  onClick={exportarBoletoPDF}
                  disabled={updating}
                  className="flex items-center gap-2 bg-blue-600/20 text-blue-400 border border-blue-500/30 px-4 py-2 rounded-lg text-sm font-bold hover:bg-blue-600/30 transition-all hover:-translate-y-0.5"
                >
                  {updating ? (
                    <Loader2 className="w-4 h-4 animate-spin" />
                  ) : (
                    <FileText className="w-4 h-4" />
                  )}
                  Descargar boleto visual (PDF)
                </button>}
                </div>
              </div>

              <div className="space-y-6">
                {selectedPass && <section aria-label="Vista del pase de abordar" className="overflow-hidden rounded-2xl border border-blue-400/30 bg-gradient-to-br from-slate-800 to-blue-950 p-5 shadow-xl">
                  <p className="text-xs font-bold uppercase tracking-[0.2em] text-blue-300">Aerolíneas Pabón · Pase de abordar</p>
                  <div className="mt-5 flex items-center justify-between text-3xl font-black text-white"><span>{selectedPass.origen}</span><Plane className="h-6 w-6 text-blue-300" /><span>{selectedPass.destino}</span></div>
                  <div className="mt-5 grid grid-cols-2 gap-4 border-t border-white/20 pt-4 text-sm">
                    <div><p className="text-xs text-blue-200">Pasajero</p><p className="font-bold">{selectedPass.pasajero}</p></div>
                    <div><p className="text-xs text-blue-200">Vuelo</p><p className="font-bold">AP-{selectedPass.vuelo}</p></div>
                    <div><p className="text-xs text-blue-200">Asiento</p><p className="font-bold">{selectedPass.asiento}</p></div>
                    <div><p className="text-xs text-blue-200">Puerta</p><p className="font-bold">{selectedPass.puerta}</p></div>
                  </div>
                  <p className="mt-4 text-xs text-blue-100">Sale de {selectedPass.origen} (hora local): {selectedPass.salida_local?.replace("T", " ").slice(0, 16)} · {selectedPass.zona_salida}</p>
                  <p className="text-xs text-blue-100">Llega a {selectedPass.destino} (hora local): {selectedPass.llegada_local?.replace("T", " ").slice(0, 16)} · {selectedPass.zona_llegada}</p>
                  <figure className="mt-4 border-t border-white/20 pt-4 text-center"><img src={`/api/boletos/${selectedBoleto.id_boleto}/wallet/qr.png`} width={160} height={160} alt="QR para abrir la guía de descarga del pase" className="mx-auto rounded bg-white p-2" /><figcaption className="mt-2 text-xs">Escanea con la cámara del celular para abrir el pase</figcaption></figure>
                  <p className="mt-3 text-xs text-blue-100">El QR abre una página para descargar el pase y compartirlo desde Archivos a Passbook. Usa la cámara del celular, no el lector de códigos de la app. Ambos dispositivos deben estar en la misma red.</p>
                  <QRNetworkInfo />
                </section>}
                <div>
                  <label className="text-[10px] uppercase text-gray-500 font-bold tracking-widest block mb-1">
                    Nombre
                  </label>
                  <p className="text-lg font-bold">{selectedBoleto.nombre_pasajero}</p>
                </div>

                <div>
                  <label className="text-[10px] uppercase text-gray-500 font-bold tracking-widest block mb-1">
                    Datos de Contacto e Identificación
                  </label>
                  <p className="text-sm text-gray-300">Email: {selectedBoleto.email_pasajero}</p>
                  <p className="text-sm text-gray-300">Pasaporte: {selectedBoleto.pasaporte}</p>
                </div>

                <div className="h-px w-full bg-white/10 my-4" />

                {getFlightDetails(selectedBoleto.id_vuelo) && (
                  <div className="bg-white/5 p-4 rounded-xl border border-white/10 space-y-4">
                    <p className="text-[10px] uppercase text-blue-400 font-bold tracking-widest block">
                      Información del Vuelo
                    </p>

                    <div className="flex items-center justify-between">
                      <div>
                        <p className="text-2xl font-bold">
                          {getFlightDetails(selectedBoleto.id_vuelo)?.org?.codigo}
                        </p>
                        <p className="text-xs text-gray-400 line-clamp-1 max-w-[80px]">
                          {getFlightDetails(selectedBoleto.id_vuelo)?.org?.pais}
                        </p>
                      </div>

                      <div className="flex-1 px-4 flex flex-col items-center">
                        <p className="text-[10px] text-gray-500">
                          {selectedBoleto.tiempo_de_viaje} hrs
                        </p>
                        <div className="w-full h-px bg-white/20 relative my-2">
                          <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-background px-1">
                            <Plane className="w-3 h-3 text-blue-400" />
                          </div>
                        </div>
                      </div>

                      <div className="text-right">
                        <p className="text-2xl font-bold">
                          {getFlightDetails(selectedBoleto.id_vuelo)?.dst?.codigo}
                        </p>
                        <p className="text-xs text-gray-400 line-clamp-1 max-w-[80px]">
                          {getFlightDetails(selectedBoleto.id_vuelo)?.dst?.pais}
                        </p>
                      </div>
                    </div>

                    <div className="flex justify-between pt-2">
                      <div>
                        <p className="text-[10px] text-gray-500">Asiento</p>
                        <p className="font-bold text-white">#{selectedBoleto.id_asiento}</p>
                      </div>
                      <div className="text-right">
                        <p className="text-[10px] text-gray-500">Precio Pagado</p>
                        <p className="font-bold text-green-400">${selectedBoleto.costo}</p>
                      </div>
                    </div>
                  </div>
                )}

                <div className="pt-6 border-t border-white/10 mt-6">
                  <p className="text-[10px] uppercase text-gray-500 font-bold tracking-widest block mb-4">
                    Acciones de Estado
                  </p>

                  <div className="flex flex-col gap-3">
                    <button
                      onClick={() => changeTicketState(selectedBoleto.id_boleto, "RESERVED")}
                      disabled={updating || selectedBoleto.estado === "RESERVED"}
                      className={`py-3 rounded-xl font-bold flex items-center justify-center gap-2 transition border ${
                        selectedBoleto.estado === "RESERVED"
                          ? "bg-yellow-500/20 text-yellow-500 border-yellow-500/50 cursor-not-allowed hidden"
                          : "bg-transparent border-yellow-500/50 text-yellow-500 hover:bg-yellow-500/10"
                      }`}
                    >
                      {updating ? (
                        <Loader2 className="w-4 h-4 animate-spin" />
                      ) : (
                        <ChevronDown className="w-4 h-4" />
                      )}
                      Mover a Reserva
                    </button>

                    <button
                      onClick={() => changeTicketState(selectedBoleto.id_boleto, "SALED")}
                      disabled={updating || selectedBoleto.estado === "SALED"}
                      className={`py-3 rounded-xl font-bold flex items-center justify-center gap-2 transition shadow-lg ${
                        selectedBoleto.estado === "SALED"
                          ? "bg-emerald-600/50 text-white cursor-not-allowed border-emerald-500/50 hidden"
                          : "bg-emerald-600 hover:bg-emerald-500 text-white border-transparent"
                      }`}
                    >
                      {updating ? (
                        <Loader2 className="w-4 h-4 animate-spin" />
                      ) : (
                        <CreditCard className="w-4 h-4" />
                      )}
                      Confirmar Compra
                    </button>

                    <button
                      onClick={() => changeTicketState(selectedBoleto.id_boleto, "ANNULLED")}
                      disabled={updating || selectedBoleto.estado === "ANNULLED"}
                      className={`py-3 rounded-xl font-bold flex items-center justify-center gap-2 transition border ${
                        selectedBoleto.estado === "ANNULLED"
                          ? "bg-red-500/20 text-red-500 border-red-500/50 cursor-not-allowed hidden"
                          : "bg-transparent border-red-500/50 text-red-500 hover:bg-red-500/10"
                      }`}
                    >
                      {updating ? (
                        <Loader2 className="w-4 h-4 animate-spin" />
                      ) : (
                        <XCircle className="w-4 h-4" />
                      )}
                      Anular Boleto
                    </button>
                  </div>
                </div>
              </div>
            </div>
          ) : (
            <div className="h-full flex flex-col justify-center items-center text-center opacity-50 border-2 border-dashed border-white/10 rounded-2xl p-6">
              <Ticket className="w-16 h-16 text-gray-600 mb-4" />
              <h4 className="text-xl font-bold mb-2">Ningún Boleto Seleccionado</h4>
              <p className="text-gray-400 text-sm">
                Haz clic en un boleto de la lista de la izquierda para ver los detalles y actualizar su estado.
              </p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
