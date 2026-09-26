"use client";

import { useEffect, useState } from "react";
import {
  Ticket,
  Search,
  Loader2,
  XCircle,
  CreditCard,
  ChevronDown,
  Plane,
  FileText,
  RefreshCw,
  Wallet,
  Mail,
  IdCard,
} from "lucide-react";
import { EmptyState, PageHeader, RouteCodes, TicketStatusBadge } from "@/components/ui";
import { downloadBoardingPassPdf } from "@/lib/boardingPassPdf";
import { translateUiText } from "@/lib/englishUi";
import { useLanguage } from "@/context/LanguageContext";
import QRNetworkInfo from "@/components/QRNetworkInfo";

export default function GestionBoletos() {
  const { language } = useLanguage();
  const localized = (message: string) => language === "en" ? translateUiText(message) : message;
  const [boletos, setBoletos] = useState<any[]>([]);
  const [vuelos, setVuelos] = useState<any[]>([]);
  const [ciudades, setCiudades] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedBoleto, setSelectedBoleto] = useState<any>(null);
  const [selectedPass, setSelectedPass] = useState<any>(null);
  const [updating, setUpdating] = useState(false);
  const [searchTerm, setSearchTerm] = useState("");
  const [statusFilter, setStatusFilter] = useState("ALL");

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
    if (!confirm(localized(`¿Estás seguro que deseas cambiar el estado a ${newState}?`))) return;

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
        alert(localized(d.error || "Hubo un error actualizando el estado"));
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

  const exportarBoletoPDF = async () => {
    if (!selectedBoleto) return;
    setUpdating(true);
    try {
      await downloadBoardingPassPdf(selectedBoleto);
    } catch (error) {
      console.error(error);
      alert(localized("No se pudo descargar el boleto visual en PDF"));
    } finally {
      setUpdating(false);
    }
  };

  if (loading) {
    return (
      <div className="flex h-[60vh] items-center justify-center">
        <Loader2 className="h-10 w-10 animate-spin text-navy-500" />
      </div>
    );
  }

  const searched = filterBoletos();
  const filtered = statusFilter === "ALL" ? searched : searched.filter((b: any) => b.estado === statusFilter);
  const statusFilters = [
    ["ALL", "Todos"], ["SALED", "Vendidos"], ["RESERVED", "Reservados"], ["REFUNDED", "En reembolso"], ["ANNULLED", "Anulados"],
  ] as const;
  const selectedDetails = selectedBoleto ? getFlightDetails(selectedBoleto.id_vuelo) : null;

  return (
    <div className="fade-up pb-10">
      <PageHeader icon={Ticket} eyebrow="Mis viajes" title="Gestión de Boletos"
        subtitle="Consulta tus boletos, descarga el boleto visual o abre el pase en una billetera compatible."
        actions={<button onClick={() => fetchData()} className="btn-secondary"><RefreshCw className="h-4 w-4" /> Refrescar</button>} />

      <div className="card mb-6 flex flex-wrap items-center gap-3 p-4">
        <div className="relative min-w-[260px] flex-1">
          <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
          <input type="text" placeholder="Buscar número de boleto, pasajero, email o pasaporte..." aria-label="Buscar boletos" value={searchTerm} onChange={(e) => setSearchTerm(e.target.value)} className="field pl-9" />
        </div>
        <div className="flex flex-wrap gap-1.5" role="group" aria-label="Filtrar por estado">
          {statusFilters.map(([value, label]) => <button key={value} type="button" onClick={() => setStatusFilter(value)} aria-pressed={statusFilter === value}
            className={`rounded-full border px-3 py-1.5 text-xs font-semibold transition ${statusFilter === value ? "border-navy-900 bg-navy-900 text-white" : "border-slate-200 bg-white text-slate-600 hover:border-navy-300"}`}>
            {label}
          </button>)}
        </div>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-5">
        <section className="card flex max-h-[720px] flex-col lg:col-span-3">
          <div className="card-header">
            <h2 className="section-title">Todos los Boletos ({filtered.length})</h2>
          </div>
          <div className="flex-1 space-y-2.5 overflow-y-auto p-4">
            {filtered.length === 0 ? (
              <EmptyState icon={Ticket} title="Sin boletos">No se encontraron boletos.</EmptyState>
            ) : filtered.map((b: any) => {
              const details = getFlightDetails(b.id_vuelo);
              const active = selectedBoleto?.id_boleto === b.id_boleto;
              return <button key={b.id_boleto} type="button" onClick={() => setSelectedBoleto(b)} aria-pressed={active}
                className={`w-full rounded-xl border p-4 text-left transition hover:border-navy-300 hover:bg-navy-50/50 ${active ? "border-navy-800 bg-navy-50 ring-1 ring-navy-800" : "border-slate-200 bg-white"}`}>
                <div className="flex items-start justify-between gap-3">
                  <div className="flex min-w-0 items-center gap-3">
                    <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-navy-900 text-sm font-bold text-gold-300">
                      {(b.nombre_pasajero || "?").trim().charAt(0).toUpperCase()}
                    </span>
                    <div className="min-w-0">
                      <p className="truncate font-semibold text-navy-900">{b.nombre_pasajero}</p>
                      <p className="truncate text-xs text-slate-500">{b.email_pasajero}</p>
                    </div>
                  </div>
                  <TicketStatusBadge state={b.estado} />
                </div>
                <div className="mt-3 flex flex-wrap items-center gap-x-6 gap-y-2 border-t border-dashed border-slate-200 pt-3 text-sm">
                  <div><p className="text-[10px] font-semibold uppercase text-slate-400">Boleto ID</p><p className="font-mono text-navy-900">#{b.id_boleto}</p></div>
                  {details && <RouteCodes from={details.org?.codigo} to={details.dst?.codigo} />}
                  <div className="ml-auto text-right"><p className="text-[10px] font-semibold uppercase text-slate-400">Costo</p><p className="font-bold text-navy-900">${b.costo}</p></div>
                </div>
              </button>;
            })}
          </div>
        </section>

        <aside className="card h-max lg:sticky lg:top-24 lg:col-span-2">
          {selectedBoleto ? (
            <div className="fade-up">
              <div className="card-header">
                <div><p className="eyebrow">Detalle del Boleto</p><p className="font-mono text-lg font-bold text-navy-900">#{selectedBoleto.id_boleto}</p></div>
                <TicketStatusBadge state={selectedBoleto.estado} />
              </div>
              <div className="space-y-5 p-5">
                {selectedPass && <section aria-label="Vista del pase de abordar" className="overflow-hidden rounded-2xl border border-slate-200 shadow-card">
                  <div className="bg-navy-900 p-5 text-white">
                    <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-gold-300">Aerolíneas Pabón · Pase de abordar</p>
                    <div className="mt-4 flex items-center justify-between text-3xl font-bold"><span>{selectedPass.origen}</span><Plane className="h-6 w-6 rotate-45 text-gold-300" /><span>{selectedPass.destino}</span></div>
                  </div>
                  <dl className="grid grid-cols-2 gap-4 p-5 text-sm">
                    <div><dt className="text-xs text-slate-500">Pasajero</dt><dd className="font-semibold text-navy-900">{selectedPass.pasajero}</dd></div>
                    <div><dt className="text-xs text-slate-500">Vuelo</dt><dd className="font-semibold text-navy-900">AP-{selectedPass.vuelo}</dd></div>
                    <div><dt className="text-xs text-slate-500">Asiento</dt><dd className="font-semibold text-navy-900">{selectedPass.asiento}</dd></div>
                    <div><dt className="text-xs text-slate-500">Puerta</dt><dd className="font-semibold text-navy-900">{selectedPass.puerta}</dd></div>
                  </dl>
                  <div className="space-y-1 px-5 pb-4 text-xs text-slate-600">
                    <p>Sale de {selectedPass.origen} (hora local): {selectedPass.salida_local?.replace("T", " ").slice(0, 16)} · {selectedPass.zona_salida}</p>
                    <p>Llega a {selectedPass.destino} (hora local): {selectedPass.llegada_local?.replace("T", " ").slice(0, 16)} · {selectedPass.zona_llegada}</p>
                  </div>
                  <figure className="border-t border-dashed border-slate-300 bg-slate-50 p-5 text-center">
                    <img src={`/api/boletos/${selectedBoleto.id_boleto}/wallet/qr.png`} width={150} height={150} alt="QR para abrir la guía de descarga del pase" className="mx-auto rounded-xl bg-white p-2 shadow-card" />
                    <figcaption className="mt-2 text-xs text-slate-600">Escanea con la cámara del celular para abrir el pase</figcaption>
                    <p className="mt-2 text-xs text-slate-500">El QR abre una página para descargar el pase y compartirlo desde Archivos a Passbook. Usa la cámara del celular, no el lector de códigos de la app. Ambos dispositivos deben estar en la misma red.</p>
                    <QRNetworkInfo />
                  </figure>
                </section>}

                {selectedBoleto.estado === "SALED" && <div className="grid gap-2 sm:grid-cols-2">
                  <button onClick={exportarBoletoPDF} disabled={updating} className="btn-primary">
                    {updating ? <Loader2 className="h-4 w-4 animate-spin" /> : <FileText className="h-4 w-4" />} Descargar boleto visual (PDF)
                  </button>
                  <a href={`/pase/${selectedBoleto.id_boleto}/billetera`} className="btn-gold"><Wallet className="h-4 w-4" /> Abrir pase para Passbook</a>
                </div>}

                <div>
                  <p className="eyebrow mb-1">Nombre</p>
                  <p className="text-lg font-bold text-navy-900">{selectedBoleto.nombre_pasajero}</p>
                  <p className="eyebrow mb-1 mt-4">Datos de Contacto e Identificación</p>
                  <p className="flex items-center gap-2 text-sm text-slate-600"><Mail className="h-4 w-4 text-slate-400" /> Email: {selectedBoleto.email_pasajero}</p>
                  <p className="mt-1 flex items-center gap-2 text-sm text-slate-600"><IdCard className="h-4 w-4 text-slate-400" /> Pasaporte: {selectedBoleto.pasaporte}</p>
                </div>

                {selectedDetails && <div className="rounded-xl border border-slate-200 bg-slate-50 p-4">
                  <p className="eyebrow mb-3">Información del Vuelo</p>
                  <div className="flex items-center justify-between">
                    <div><p className="text-2xl font-bold text-navy-900">{selectedDetails.org?.codigo}</p><p className="max-w-[90px] truncate text-xs text-slate-500">{selectedDetails.org?.pais}</p></div>
                    <div className="flex flex-1 flex-col items-center px-3">
                      <p className="text-[10px] text-slate-500">{selectedBoleto.tiempo_de_viaje} hrs</p>
                      <div className="my-1 flex w-full items-center gap-1" aria-hidden="true"><span className="h-px flex-1 border-t border-dashed border-slate-300" /><Plane className="h-3.5 w-3.5 rotate-45 text-gold-500" /><span className="h-px flex-1 border-t border-dashed border-slate-300" /></div>
                    </div>
                    <div className="text-right"><p className="text-2xl font-bold text-navy-900">{selectedDetails.dst?.codigo}</p><p className="max-w-[90px] truncate text-xs text-slate-500">{selectedDetails.dst?.pais}</p></div>
                  </div>
                  <div className="mt-3 flex justify-between border-t border-slate-200 pt-3">
                    <div><p className="text-[10px] text-slate-500">Asiento</p><p className="font-bold text-navy-900">#{selectedBoleto.id_asiento}</p></div>
                    <div className="text-right"><p className="text-[10px] text-slate-500">Precio Pagado</p><p className="font-bold text-navy-900">${selectedBoleto.costo}</p></div>
                  </div>
                </div>}

                <div className="border-t border-slate-100 pt-4">
                  <p className="eyebrow mb-3">Acciones de Estado</p>
                  <div className="grid gap-2">
                    {selectedBoleto.estado !== "SALED" && <button onClick={() => changeTicketState(selectedBoleto.id_boleto, "SALED")} disabled={updating} className="btn-success">
                      {updating ? <Loader2 className="h-4 w-4 animate-spin" /> : <CreditCard className="h-4 w-4" />} Confirmar Compra
                    </button>}
                    {selectedBoleto.estado !== "RESERVED" && <button onClick={() => changeTicketState(selectedBoleto.id_boleto, "RESERVED")} disabled={updating} className="btn-secondary">
                      {updating ? <Loader2 className="h-4 w-4 animate-spin" /> : <ChevronDown className="h-4 w-4" />} Mover a Reserva
                    </button>}
                    {selectedBoleto.estado !== "ANNULLED" && <button onClick={() => changeTicketState(selectedBoleto.id_boleto, "ANNULLED")} disabled={updating} className="btn-danger">
                      {updating ? <Loader2 className="h-4 w-4 animate-spin" /> : <XCircle className="h-4 w-4" />} Anular Boleto
                    </button>}
                  </div>
                </div>
              </div>
            </div>
          ) : (
            <div className="p-5"><EmptyState icon={Ticket} title="Ningún Boleto Seleccionado">Haz clic en un boleto de la lista de la izquierda para ver los detalles y actualizar su estado.</EmptyState></div>
          )}
        </aside>
      </div>
    </div>
  );
}
