"use client";

import { useEffect, useState, type FormEvent } from "react";
import { Search, Check, X, CreditCard, Plane, MapPin, Loader2, ArrowRight, Ticket, Armchair, Clock, Download, Wallet, Hash, Box } from "lucide-react";
import dynamic from "next/dynamic";
import { formatFlightLocalTime } from "@/lib/flightTime";
import { PURCHASE_CAPITALS } from "@/data/capitals";
import QRNetworkInfo from "@/components/QRNetworkInfo";
import { translateUiText } from "@/lib/englishUi";
import { useLanguage } from "@/context/LanguageContext";
import { EmptyState, PageHeader, RouteCodes } from "@/components/ui";

const PlaneModelViewer = dynamic(() => import("@/components/PlaneModelViewer"), { 
  ssr: false,
  loading: () => (
    <div className="flex h-[300px] w-full animate-pulse items-center justify-center rounded-2xl bg-slate-100 text-xs font-semibold uppercase tracking-widest text-slate-400">
      Cargando Motor 3D...
    </div>
  )
});

export default function Boletos() {
  const { language } = useLanguage();
  const localized = (message: string) => language === "en" ? translateUiText(message) : message;
  const [ciudades, setCiudades] = useState([]);
  const [aviones, setAviones] = useState<{ id: number; nombre: string }[]>([]);
  const [vuelos, setVuelos] = useState([]);
  const [asientos, setAsientos] = useState([]);
  const [precios, setPrecios] = useState<any>(null);
  
  const [selectedOrigin, setSelectedOrigin] = useState("");
  const [selectedDestination, setSelectedDestination] = useState("");
  const [filteredVuelos, setFilteredVuelos] = useState<any[]>([]);
  const [flightIDSearch, setFlightIDSearch] = useState("");
  const [flightSearchError, setFlightSearchError] = useState<string | null>(null);
  const [flightSearchLoading, setFlightSearchLoading] = useState(false);
  
  const [selectedVuelo, setSelectedVuelo] = useState<any>(null);
  const [selectedSeat, setSelectedSeat] = useState<any>(null);
  const [passenger, setPassenger] = useState({
    nombre: "",
    email: "",
    pasaporte: ""
  });
  const [loading, setLoading] = useState(true);
  const [bookingLoading, setBookingLoading] = useState(false);
  const [buyerTimeZone, setBuyerTimeZone] = useState("America/Bogota");
  const [purchasedPass, setPurchasedPass] = useState<{ id_boleto: number; pasajero: string; vuelo: number; asiento: string; qr_url: string; salida_local: string; llegada_local: string; zona_salida: string; zona_llegada: string; origen: string; destino: string } | null>(null);
  const [lastWrite, setLastWrite] = useState<{ ticketID: number; node: string; server: string } | null>(null);
  const [pdfDownloading, setPdfDownloading] = useState(false);
  const [walletCapabilities, setWalletCapabilities] = useState({ apple: false, google: false });
  const [googleWalletURL, setGoogleWalletURL] = useState("");

  useEffect(() => {
    fetch("/api/wallet/capabilities").then((response) => response.json()).then(setWalletCapabilities).catch(() => {});
  }, []);

  useEffect(() => {
    if (!purchasedPass || !walletCapabilities.google) return;
    fetch(`/api/boletos/${purchasedPass.id_boleto}/wallet/google`)
      .then((response) => response.json()).then((result) => setGoogleWalletURL(result.url || "")).catch(() => {});
  }, [purchasedPass, walletCapabilities.google]);

  useEffect(() => {
    // Reset passenger when seat changes
    if (selectedSeat) {
      setPassenger({ nombre: "", email: "", pasaporte: "" });
    }
  }, [selectedSeat]);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const countryData = JSON.parse(localStorage.getItem("airres-country") || "{}");
        const countryHeaders = {
          "X-User-Country": countryData.name || "Estados Unidos",
          "X-Region": countryData.region || "America"
        };
        
        const [cRes, vRes, pRes, aRes] = await Promise.all([
          fetch("/api/ciudades", { headers: countryHeaders }),
          fetch("/api/vuelos", { headers: countryHeaders }),
          fetch("/api/precios", { headers: countryHeaders }),
          fetch("/api/aviones", { headers: countryHeaders })
        ]);

        if (cRes.ok) setCiudades(await cRes.json());
        if (vRes.ok) setVuelos(await vRes.json());
        if (pRes.ok) setPrecios(await pRes.json());
        if (aRes.ok) setAviones(await aRes.json());
      } catch (e) {
        console.error(e);
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, []);

  const [searchMode, setSearchMode] = useState<"route" | "id">("route");
  const [originSearch, setOriginSearch] = useState("");
  const [destinationSearch, setDestinationSearch] = useState("");

  const handleSearch = () => {
    if (!selectedOrigin || !selectedDestination) return;
    setFlightIDSearch("");
    setFlightSearchError(null);
    const filtered = vuelos.filter((v: any) => 
      v.id_origen === parseInt(selectedOrigin) && 
      v.id_destino === parseInt(selectedDestination) &&
      (v.id_estado_vuelo === 1 || v.estado_vuelo === "SCHEDULED" || v.id_estado === 1)
    );
    setFilteredVuelos(filtered);
    setSelectedVuelo(null);
    setAsientos([]);
  };

  const filteredOriginCiudades = ciudades.filter((c: any) =>
    `${c.pais} ${c.codigo} ${c.ciudad || ''}`.toLowerCase().includes(originSearch.toLowerCase())
  );

  const filteredDestinationCiudades = ciudades.filter((c: any) =>
    `${c.pais} ${c.codigo} ${c.ciudad || ''}`.toLowerCase().includes(destinationSearch.toLowerCase())
  );

  const loadSeats = async (vuelo: any) => {
    setSelectedVuelo(vuelo);
    setSelectedSeat(null);
    setAsientos([]);
    setLoading(true);
    try {
      const countryData = JSON.parse(localStorage.getItem("airres-country") || "{}");
      const countryHeaders = {
        "X-User-Country": countryData.name || "Estados Unidos",
        "X-Region": countryData.region || "America"
      };
      const res = await fetch(`/api/vuelos/${vuelo.id}/asientos`, { headers: countryHeaders });
      if (!res.ok) throw new Error("No se pudieron cargar los asientos de este vuelo.");
      setAsientos(await res.json());
    } catch(e) {
      console.error(e);
      setFlightSearchError("No se pudieron cargar los asientos de este vuelo. Inténtalo de nuevo.");
    } finally {
      setLoading(false);
    }
  };

  const handleFlightIDSearch = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    await searchFlightByID(flightIDSearch.trim());
  };

  const searchFlightByID = async (id: string) => {
    if (!/^[1-9]\d*$/.test(id) || Number(id) > 4294967295) {
      setFlightSearchError("Ingresa un ID de vuelo válido.");
      return;
    }
    setFlightSearchLoading(true);
    setFlightSearchError(null);
    setFilteredVuelos([]);
    setSelectedVuelo(null);
    setSelectedSeat(null);
    setAsientos([]);
    try {
      const countryData = JSON.parse(localStorage.getItem("airres-country") || "{}");
      const headers = {
        "X-User-Country": countryData.name || "Estados Unidos",
        "X-Region": countryData.region || "America"
      };
      const res = await fetch(`/api/vuelos?id=${encodeURIComponent(id)}&scope=all`, { headers });
      if (!res.ok) throw new Error("No se pudo consultar el vuelo. Inténtalo de nuevo.");
      const matches = await res.json();
      const flight = matches.find((item: any) => String(item.id) === id);
      if (!flight) {
        setFlightSearchError(`No se encontró el vuelo AP-${id}.`);
      } else if (![1, 8].includes(flight.id_estado_vuelo) || flight.salida_programada <= Math.floor(Date.now() / 1000)) {
        setFlightSearchError(`El vuelo AP-${id} existe, pero ya no admite compras.`);
      } else {
        setFilteredVuelos([flight]);
        await loadSeats(flight);
      }
    } catch (error) {
      setFlightSearchError(error instanceof Error ? error.message : "No se pudo consultar el vuelo.");
    } finally {
      setFlightSearchLoading(false);
    }
  };

  // Deep link from the flight catalog: /boletos?vuelo=ID
  useEffect(() => {
    const requested = new URLSearchParams(window.location.search).get("vuelo");
    if (!requested) return;
    setSearchMode("id");
    setFlightIDSearch(requested);
    searchFlightByID(requested);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const getPrice = (vuelo: any, clase: string) => {
    if (!precios || !ciudades.length) return 0;
    const org = (ciudades.find((c: any) => c.id === vuelo.id_origen) as any)?.codigo;
    const dst = (ciudades.find((c: any) => c.id === vuelo.id_destino) as any)?.codigo;
    
    if (clase === 'VIP') {
      return precios.matriz_precios_vip?.[org]?.[dst] ?? 0;
    }
    return precios.matriz_precios_regular?.[org]?.[dst] ?? 0;
  };

  const colorPorEstado = (estado: string) => {
    switch(estado) {
      case 'AVAILABLE': return 'border border-navy-200 bg-white text-navy-700 hover:border-navy-600 hover:bg-navy-50';
      case 'RESERVED': return 'border border-amber-300 bg-amber-200 text-amber-900';
      case 'SALED': return 'border border-slate-300 bg-slate-300 text-slate-600';
      default: return 'cursor-not-allowed border border-slate-200 bg-slate-100 text-slate-300 line-through';
    }
  };

  const downloadVisualTicket = async (ticketID: number) => {
    setPdfDownloading(true);
    try {
      const { downloadBoardingPassPdf } = await import("@/lib/boardingPassPdf");
      await downloadBoardingPassPdf({ id_boleto: ticketID, estado: "SALED" });
    } catch (error) {
      console.error(error);
      alert(localized(`El boleto #${ticketID} quedó registrado, pero no se descargó el PDF. Puedes intentarlo de nuevo desde Gestión de Boletos.`));
    } finally {
      setPdfDownloading(false);
    }
  };

  const procesarBoleto = async (nuevoEstado: string) => {
    if (selectedVuelo && ![1, 8].includes(selectedVuelo.id_estado_vuelo)) {
      alert(localized("⚠️ Este vuelo ya no admite reservas ni compras."));
      return;
    }

    if (!passenger.nombre || !passenger.email || !passenger.pasaporte) {
      alert(localized("⚠️ Por favor completa todos los datos del pasajero antes de continuar."));
      return;
    }

    setBookingLoading(true);
    try {
      const cost = getPrice(selectedVuelo, selectedSeat.clase);
      const travelTime = Math.round((selectedVuelo.llegada_programada - selectedVuelo.salida_programada) / 3600);
      
      const res = await fetch(`/api/reservas`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-User-Country": JSON.parse(localStorage.getItem("airres-country") || "{}").name || "Estados Unidos",
          "X-Region": JSON.parse(localStorage.getItem("airres-country") || "{}").region || "America"
        },
        body: JSON.stringify({
          id_vuelo: selectedVuelo.id,
          id_asiento: selectedSeat.id,
          nombre_pasajero: passenger.nombre,
          email_pasajero: passenger.email,
          pasaporte: passenger.pasaporte,
          tiempo_de_viaje: travelTime,
          estado: nuevoEstado,
          costo: cost,
          time_zone_compra: buyerTimeZone
        })
      });

      if (!res.ok) {
        const errData = await res.json().catch(() => ({}));
        alert(localized(`⚠️ ${errData.error || "Error al procesar la reserva/compra"}`));
      } else {
        const ticket = await res.json();
        setLastWrite({ ticketID: ticket.id_boleto, node: res.headers.get("X-Write-Node") || "", server: res.headers.get("X-Served-By") || "" });
        if (ticket.replication_pending) {
          alert(localized("El boleto quedó guardado, pero la confirmación de réplica está pendiente. Consulta su estado en Gestión de Boletos."));
        }
        if (nuevoEstado === "SALED") {
          try {
            const passResponse = await fetch(`/api/boletos/${ticket.id_boleto}/pase`);
            if (!passResponse.ok) throw new Error("pase_no_disponible");
            const pass = await passResponse.json();
            setPurchasedPass(pass);
            await downloadVisualTicket(ticket.id_boleto);
          } catch (passError) {
            console.error(passError);
            alert(localized(`La compra del boleto #${ticket.id_boleto} quedó registrada, pero el pase no se pudo mostrar. Puedes consultarlo en Gestión de Boletos.`));
          }
        }
        // Refresh seats
        await loadSeats(selectedVuelo);
        setSelectedSeat(null);
      }
    } catch(e) {
      console.error(e);
      alert(localized("⚠️ No se pudo completar la solicitud. Revisa la conexión e inténtalo de nuevo."));
    } finally {
      setBookingLoading(false);
    }
  };

  if (loading && !vuelos.length) {
    return <div className="flex h-[60vh] items-center justify-center"><Loader2 className="h-10 w-10 animate-spin text-navy-500" /></div>;
  }

  const cityByID = (id: number) => ciudades.find((c: any) => c.id === id) as any;
  const step = purchasedPass ? 4 : selectedSeat ? 3 : selectedVuelo ? 2 : 1;
  const steps = ["Buscar vuelo", "Elegir asiento", "Pasajero y pago", "Confirmación"];
  const chooseSeat = (seat: any) => {
    setSelectedSeat(seat);
    if (seat.estado !== 'AVAILABLE') {
      setPassenger({ nombre: seat.nombre_pasajero || "", email: seat.email_pasajero || "", pasaporte: seat.pasaporte || "" });
    } else {
      setPassenger({ nombre: "", email: "", pasaporte: "" });
    }
  };
  const seatButton = (seat: any, size: "vip" | "regular") => <button
    key={`${selectedVuelo?.id}-${seat.id}-${seat.codigo}`}
    onClick={() => chooseSeat(seat)}
    aria-label={`Asiento ${seat.codigo}`}
    aria-pressed={selectedSeat?.id === seat.id}
    className={`flex items-center justify-center rounded-t-lg rounded-b font-bold shadow-sm transition ${size === "vip" ? "h-11 w-11 text-[11px]" : "h-9 w-8 text-[9px]"} ${selectedSeat?.id === seat.id ? "z-10 scale-110 border-navy-900 bg-navy-900 text-white ring-2 ring-gold-400 ring-offset-2" : colorPorEstado(seat.estado)}`}
  >{seat.codigo}</button>;
  const vipSeats = asientos.filter((s: any) => s.clase === 'VIP');
  const regularSeats = asientos.filter((s: any) => s.clase === 'REGULAR');
  const availableCount = asientos.filter((s: any) => s.estado === 'AVAILABLE').length;

  return (
    <div className="fade-up pb-10">
      <PageHeader icon={Ticket} eyebrow="Reservas" title="Venta de Boletos"
        subtitle="Busca tu destino, selecciona tu asiento y vuela con Pabon-go."
        actions={<a href="/gestion-boletos" className="btn-secondary">Ya compré un boleto · Ir a Gestión de Boletos</a>} />

      <ol className="mb-6 grid grid-cols-2 gap-2 sm:grid-cols-4" aria-label="Pasos de la compra">
        {steps.map((label, index) => {
          const number = index + 1;
          const done = number < step;
          const current = number === step;
          return <li key={label} aria-current={current ? "step" : undefined}
            className={`flex items-center gap-3 rounded-xl border px-3 py-2.5 text-sm font-semibold ${current ? "border-navy-900 bg-navy-900 text-white" : done ? "border-emerald-200 bg-emerald-50 text-emerald-800" : "border-slate-200 bg-white text-slate-400"}`}>
            <span className={`flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-xs ${current ? "bg-gold-400 text-navy-950" : done ? "bg-emerald-600 text-white" : "bg-slate-100 text-slate-500"}`}>
              {done ? <Check className="h-4 w-4" /> : number}
            </span>
            {label}
          </li>;
        })}
      </ol>

      {lastWrite && <p className="alert alert-info mb-5"><span>Boleto #{lastWrite.ticketID} registrado primero en <strong>{lastWrite.node === "pg_am" ? "PostgreSQL América" : lastWrite.node === "pg_eu" ? "PostgreSQL Europa/Asia" : "el servidor disponible"}</strong>{lastWrite.server && <> por el <strong>{lastWrite.server === "america" ? "Servidor América" : lastWrite.server === "europa" ? "Servidor Europa" : lastWrite.server === "asia" ? "Servidor Asia" : lastWrite.server}</strong></>}. La sincronización con los demás nodos puede verse en el panel de Sincronización.</span></p>}

      {purchasedPass && <section role="dialog" aria-label="Pase de abordar" className="mb-8 overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-lift">
        <div className="flex flex-wrap items-center justify-between gap-3 bg-emerald-600 px-6 py-3 text-white">
          <p className="flex items-center gap-2 font-semibold"><Check className="h-5 w-5" /> Compra confirmada · Pase de abordar #{purchasedPass.id_boleto}</p>
          <button onClick={() => setPurchasedPass(null)} className="text-sm font-semibold text-white/90 underline hover:text-white">Cerrar</button>
        </div>
        <div className="grid md:grid-cols-[minmax(0,1fr)_auto]">
          <div className="p-6">
            <p className="eyebrow">Aerolíneas Pabón · Pase de abordar</p>
            <div className="mt-3 flex items-center gap-4"><RouteCodes from={purchasedPass.origen} to={purchasedPass.destino} size="lg" /></div>
            <dl className="mt-5 grid grid-cols-2 gap-4 text-sm sm:grid-cols-4">
              <div><dt className="text-xs text-slate-500">Pasajero</dt><dd className="font-semibold text-navy-900">{purchasedPass.pasajero}</dd></div>
              <div><dt className="text-xs text-slate-500">Vuelo</dt><dd className="font-semibold text-navy-900">AP-{purchasedPass.vuelo}</dd></div>
              <div><dt className="text-xs text-slate-500">Asiento</dt><dd className="font-semibold text-navy-900">{purchasedPass.asiento}</dd></div>
              <div><dt className="text-xs text-slate-500">Boleto</dt><dd className="font-semibold text-navy-900">#{purchasedPass.id_boleto}</dd></div>
            </dl>
            <div className="mt-4 space-y-1 text-sm text-slate-600">
              <p>Sale de {purchasedPass.origen} (hora local): {formatFlightLocalTime(Math.floor(new Date(purchasedPass.salida_local).getTime() / 1000), purchasedPass.zona_salida, language)}</p>
              <p>Llega a {purchasedPass.destino} (hora local): {formatFlightLocalTime(Math.floor(new Date(purchasedPass.llegada_local).getTime() / 1000), purchasedPass.zona_llegada, language)}</p>
            </div>
            <div className="mt-5 flex flex-wrap gap-2">
              <button onClick={() => downloadVisualTicket(purchasedPass.id_boleto)} disabled={pdfDownloading} className="btn-primary"><Download className="h-4 w-4" />{pdfDownloading ? "Descargando PDF..." : "Descargar boleto visual (PDF)"}</button>
              <a href={`/pase/${purchasedPass.id_boleto}/billetera`} className="btn-gold"><Wallet className="h-4 w-4" />Abrir pase para Passbook</a>
              <a href={`/gestion-boletos?boleto=${purchasedPass.id_boleto}`} className="btn-secondary">Ver en Gestión de Boletos</a>
              {walletCapabilities.apple && <a href={`/api/boletos/${purchasedPass.id_boleto}/wallet/apple.pkpass`} className="btn-secondary">Añadir a Apple Wallet</a>}
              {googleWalletURL && <a href={googleWalletURL} target="_blank" rel="noopener noreferrer" className="btn-secondary">Añadir a Google Wallet</a>}
            </div>
            <p className="mt-3 text-xs text-slate-500">El boleto visual se descarga como PDF. Puedes volver a obtenerlo desde Gestión de Boletos.</p>
            <p className="mt-1 text-xs text-slate-500">En iPhone, descarga el ZIP y comparte el .pkpass desde Archivos a Passbook. Apple Wallet oficial requiere firma de emisor.</p>
          </div>
          <figure className="flex flex-col items-center justify-center gap-2 border-t border-dashed border-slate-300 bg-slate-50 p-6 text-center text-xs text-slate-600 md:border-l md:border-t-0">
            <img src={`/api/boletos/${purchasedPass.id_boleto}/wallet/qr.png`} width={160} height={160} alt="QR para abrir la guía de descarga del pase" className="rounded-xl bg-white p-2 shadow-card" />
            <figcaption className="max-w-[180px]">Escanea con la cámara del celular para abrir el pase</figcaption>
          </figure>
        </div>
        <div className="border-t border-slate-100 px-6 py-3 text-xs text-slate-500">
          <p>El QR abre una página con la descarga y los pasos para importar el pase en Passbook. Usa la cámara del celular, no el lector de códigos de la app. Ambos dispositivos deben estar en la misma red.</p>
          <QRNetworkInfo />
        </div>
      </section>}

      {/* SEARCH WIDGET */}
      <section className="card mb-6 overflow-hidden">
        <div className="flex border-b border-slate-200 bg-slate-50" role="tablist" aria-label="Tipo de búsqueda">
          {([["route", "Por ruta", MapPin], ["id", "Por número de vuelo", Hash]] as const).map(([mode, label, Icon]) => <button key={mode} type="button" role="tab" aria-selected={searchMode === mode} onClick={() => setSearchMode(mode)}
            className={`flex items-center gap-2 border-b-2 px-5 py-3 text-sm font-semibold transition ${searchMode === mode ? "border-gold-400 bg-white text-navy-900" : "border-transparent text-slate-500 hover:text-navy-800"}`}>
            <Icon className="h-4 w-4" aria-hidden="true" />{label}
          </button>)}
        </div>
        {searchMode === "route" ? (
          <div className="grid gap-4 p-5 sm:p-6 lg:grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)_auto] lg:items-end">
            <div>
              <label htmlFor="origin-select" className="field-label">Origen</label>
              <input type="text" placeholder="Filtrar ciudad/país..." value={originSearch} onChange={(e) => setOriginSearch(e.target.value)} aria-label="Filtrar origen" className="field mb-2 py-1.5 text-xs" />
              <div className="relative">
                <MapPin className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
                <select id="origin-select" value={selectedOrigin} onChange={(e) => setSelectedOrigin(e.target.value)} className="field pl-9">
                  <option value="">Seleccione Ciudad ({filteredOriginCiudades.length})...</option>
                  {filteredOriginCiudades.map((c: any) => <option key={c.id} value={c.id}>{c.pais} - {c.codigo}</option>)}
                </select>
              </div>
            </div>
            <button type="button" aria-label="Intercambiar origen y destino" onClick={() => { setSelectedOrigin(selectedDestination); setSelectedDestination(selectedOrigin); }}
              className="mx-auto flex h-10 w-10 items-center justify-center rounded-full border border-slate-300 bg-white text-navy-700 shadow-sm transition hover:bg-navy-50 lg:mb-0.5">
              <ArrowRight className="h-4 w-4" />
            </button>
            <div>
              <label htmlFor="destination-select" className="field-label">Destino</label>
              <input type="text" placeholder="Filtrar ciudad/país..." value={destinationSearch} onChange={(e) => setDestinationSearch(e.target.value)} aria-label="Filtrar destino" className="field mb-2 py-1.5 text-xs" />
              <div className="relative">
                <MapPin className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
                <select id="destination-select" value={selectedDestination} onChange={(e) => setSelectedDestination(e.target.value)} className="field pl-9">
                  <option value="">Seleccione Ciudad ({filteredDestinationCiudades.length})...</option>
                  {filteredDestinationCiudades.map((c: any) => <option key={c.id} value={c.id}>{c.pais} - {c.codigo}</option>)}
                </select>
              </div>
            </div>
            <button onClick={handleSearch} disabled={!selectedOrigin || !selectedDestination} className="btn-gold btn-lg"><Search className="h-5 w-5" /> Buscar Vuelos</button>
          </div>
        ) : (
          <form onSubmit={handleFlightIDSearch} className="flex flex-wrap items-end gap-3 p-5 sm:p-6">
            <label htmlFor="flight-id-search" className="min-w-[220px] flex-1">
              <span className="field-label">Buscar vuelo por ID</span>
              <input id="flight-id-search" type="number" min="1" inputMode="numeric" value={flightIDSearch} onChange={(event) => setFlightIDSearch(event.target.value)} placeholder="ID que aparece en Vuelos" className="field" />
            </label>
            <button type="submit" disabled={flightSearchLoading} className="btn-gold btn-lg">
              {flightSearchLoading ? <Loader2 className="h-5 w-5 animate-spin" /> : <Search className="h-5 w-5" />} Buscar por ID
            </button>
          </form>
        )}
        {flightSearchError && <p role="alert" className="alert alert-warning mx-5 mb-5 sm:mx-6">{flightSearchError}</p>}
      </section>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        {/* FLIGHT LIST */}
        <div className="space-y-3 lg:col-span-1">
          <h2 className="section-title flex items-center gap-2"><Plane className="h-5 w-5 text-navy-500" /> Vuelos Disponibles</h2>
          {filteredVuelos.length === 0 ? (
            <EmptyState icon={Search} title="Sin vuelos">No hay vuelos que coincidan con tu búsqueda.</EmptyState>
          ) : filteredVuelos.map((v: any) => {
            const origin = cityByID(v.id_origen);
            const destination = cityByID(v.id_destino);
            const active = selectedVuelo?.id === v.id;
            return <button key={v.id} type="button" onClick={() => loadSeats(v)} aria-pressed={active}
              className={`w-full rounded-2xl border bg-white p-4 text-left shadow-card transition hover:-translate-y-0.5 hover:shadow-lift ${active ? "border-navy-800 ring-2 ring-navy-800" : "border-slate-200"}`}>
              <div className="flex items-start justify-between gap-3">
                <div>
                  <p className="text-xs font-semibold text-slate-500">Vuelo AP-{v.id}</p>
                  <RouteCodes from={origin?.codigo} to={destination?.codigo} />
                </div>
                <div className="text-right">
                  <p className="text-[10px] font-semibold uppercase text-slate-400">Desde</p>
                  <p className="text-xl font-bold text-navy-900">$ {getPrice(v, 'REGULAR') || getPrice(v, 'VIP')}</p>
                </div>
              </div>
              <div className="mt-3 grid grid-cols-2 gap-2 border-t border-dashed border-slate-200 pt-3 text-xs">
                <div><p className="text-slate-400">Sale · hora local {origin?.codigo}</p><p className="font-semibold text-navy-900">{formatFlightLocalTime(v.salida_programada, origin?.time_zone, language)}</p></div>
                <div className="text-right"><p className="text-slate-400">Llega · hora local {destination?.codigo}</p><p className="font-semibold text-navy-900">{formatFlightLocalTime(v.llegada_programada, destination?.time_zone, language)}</p></div>
              </div>
            </button>;
          })}
        </div>

        {/* SEATING AREA */}
        <div className="lg:col-span-2">
          {selectedVuelo ? (
            <div className="grid grid-cols-1 gap-6 xl:grid-cols-5">
              <section className="card xl:col-span-3">
                <div className="card-header">
                  <div>
                    <h2 className="section-title">Mapa de Asientos</h2>
                    <p className="section-subtitle">{availableCount} asientos disponibles</p>
                  </div>
                  <span className="badge badge-slate">Avión ID: {selectedVuelo.id_avion}</span>
                </div>
                <div className="card-body">
                  <details className="group mb-5 rounded-xl border border-slate-200">
                    <summary className="flex cursor-pointer list-none items-center gap-2 px-4 py-3 text-sm font-semibold text-navy-800">
                      <Box className="h-4 w-4" aria-hidden="true" /> Ver el avión en 3D
                      <span className="ml-auto text-xs text-slate-400">{aviones.find((aircraft) => aircraft.id === selectedVuelo.id_avion)?.nombre}</span>
                    </summary>
                    <div className="border-t border-slate-200 p-3">
                      <PlaneModelViewer aircraftName={aviones.find((aircraft) => aircraft.id === selectedVuelo.id_avion)?.nombre} />
                    </div>
                  </details>

                  <div className="mb-5 flex flex-wrap items-center justify-center gap-4 text-xs font-medium text-slate-600">
                    <span className="flex items-center gap-2"><span className="h-4 w-4 rounded border border-navy-200 bg-white" /> Disponible</span>
                    <span className="flex items-center gap-2"><span className="h-4 w-4 rounded border border-amber-300 bg-amber-200" /> Reservado</span>
                    <span className="flex items-center gap-2"><span className="h-4 w-4 rounded bg-slate-300" /> Vendido</span>
                    <span className="flex items-center gap-2"><span className="h-4 w-4 rounded bg-navy-900 ring-2 ring-gold-400" /> Tu selección</span>
                  </div>

                  <div className="max-h-[640px] overflow-y-auto pr-1">
                    <div className="relative mx-auto max-w-sm rounded-t-[7rem] rounded-b-[2.5rem] border-2 border-slate-200 bg-slate-50 px-6 pb-8 pt-20">
                      <div aria-hidden="true" className="absolute left-1/2 top-6 h-8 w-28 -translate-x-1/2 rounded-t-full border-2 border-b-0 border-slate-300" />
                      <p className="absolute left-1/2 top-9 -translate-x-1/2 text-[10px] font-semibold uppercase tracking-[0.2em] text-slate-400">Cabina</p>

                      <div className="mb-4 flex items-center gap-2"><span className="h-px flex-1 bg-gold-300" /><span className="badge badge-gold">Primera Clase</span><span className="h-px flex-1 bg-gold-300" /></div>
                      <div className="mb-8 grid grid-cols-4 justify-items-center gap-x-3 gap-y-3">
                        {vipSeats.map((seat: any) => seatButton(seat, "vip"))}
                      </div>

                      <div className="mb-4 flex items-center gap-2"><span className="h-px flex-1 bg-navy-200" /><span className="badge badge-blue">Clase Económica</span><span className="h-px flex-1 bg-navy-200" /></div>
                      <div className="grid grid-cols-7 justify-items-center gap-x-1.5 gap-y-2.5">
                        {regularSeats.map((seat: any, idx: number) => idx % 6 === 3
                          ? [<div key={`aisle-${idx}`} aria-hidden="true" />, seatButton(seat, "regular")]
                          : seatButton(seat, "regular"))}
                      </div>
                    </div>
                  </div>
                </div>
              </section>

              {/* BOOKING PANEL */}
              <aside className="card h-max xl:sticky xl:top-24 xl:col-span-2">
                {selectedSeat ? (
                  <div className="fade-up">
                    <div className="flex items-start justify-between gap-3 rounded-t-2xl bg-navy-900 p-5 text-white">
                      <div>
                        <p className="text-xs font-semibold uppercase tracking-[0.16em] text-gold-300">{selectedSeat.clase === 'VIP' ? 'Primera Clase' : 'Económico'}</p>
                        <h3 className="mt-1 text-2xl font-bold text-white">Asiento {selectedSeat.codigo}</h3>
                      </div>
                      <div className="text-right">
                        <p className="text-[10px] font-semibold uppercase text-navy-200">Precio Final</p>
                        <p className="text-2xl font-bold text-gold-300">$ {getPrice(selectedVuelo, selectedSeat.clase)}</p>
                      </div>
                    </div>
                    <div className="space-y-4 p-5">
                      <div className="flex items-center justify-between rounded-xl bg-slate-50 px-4 py-3">
                        <div><p className="eyebrow">Ruta</p><RouteCodes from={cityByID(selectedVuelo.id_origen)?.codigo} to={cityByID(selectedVuelo.id_destino)?.codigo} /></div>
                        <div className="text-right"><p className="eyebrow">Viaje Estimado</p><p className="flex items-center justify-end gap-1 font-semibold text-navy-900"><Clock className="h-4 w-4 text-slate-400" />{Math.round((selectedVuelo.llegada_programada - selectedVuelo.salida_programada) / 3600)} Horas</p></div>
                      </div>

                      <p className="eyebrow">Información del Pasajero</p>
                      <label className="block">
                        <span className="mb-1.5 block text-xs text-slate-600">Capital desde donde compras (elige el servidor inicial)</span>
                        <select value={buyerTimeZone} onChange={(event) => setBuyerTimeZone(event.target.value)} className="field">
                          {PURCHASE_CAPITALS.map(([capital, zone]) => <option key={capital} value={zone}>{capital}</option>)}
                        </select>
                      </label>
                      <p className="text-xs text-slate-500">América registra primero en PostgreSQL América; Europa y Asia en PostgreSQL Europa/Asia. El país de arriba solo elige de dónde se consultan las listas. La salida usa la hora del aeropuerto de origen y la llegada la del destino.</p>
                      <div className="space-y-3">
                        <input type="text" placeholder="Nombre Completo" aria-label="Nombre Completo" value={passenger.nombre} disabled={selectedSeat.estado !== 'AVAILABLE'} onChange={(e) => setPassenger({ ...passenger, nombre: e.target.value })} className="field" />
                        <input type="email" placeholder="Email" aria-label="Email" value={passenger.email} disabled={selectedSeat.estado !== 'AVAILABLE'} onChange={(e) => setPassenger({ ...passenger, email: e.target.value })} className="field" />
                        <input type="text" placeholder="Número de Pasaporte" aria-label="Número de Pasaporte" value={passenger.pasaporte} disabled={selectedSeat.estado !== 'AVAILABLE'} onChange={(e) => setPassenger({ ...passenger, pasaporte: e.target.value })} className="field" />
                      </div>

                      {selectedSeat.estado === "AVAILABLE" ? (
                        <div className="grid gap-2 pt-2">
                          <button onClick={() => procesarBoleto('SALED')} disabled={bookingLoading} className="btn-gold btn-lg w-full">
                            {bookingLoading ? <Loader2 className="h-5 w-5 animate-spin" /> : <><CreditCard className="h-5 w-5" /> Comprar Ticket Ahora</>}
                          </button>
                          <button onClick={() => procesarBoleto('RESERVED')} disabled={bookingLoading} className="btn-secondary w-full">
                            {bookingLoading ? <Loader2 className="h-5 w-5 animate-spin" /> : <><Check className="h-5 w-5" /> Pre-Reservar (72h)</>}
                          </button>
                        </div>
                      ) : (
                        <div className="alert alert-error flex-col items-center text-center">
                          <X className="mx-auto h-8 w-8" />
                          <p className="font-bold">Asiento Ocupado</p>
                          <p>Este asiento ya ha sido reservado o vendido. Por favor selecciona otro.</p>
                        </div>
                      )}
                    </div>
                  </div>
                ) : (
                  <div className="p-5"><EmptyState icon={Armchair} title="Selección de Asiento">Haz clic en un asiento del mapa para ver los detalles del precio y completar tu reserva.</EmptyState></div>
                )}
              </aside>
            </div>
          ) : (
            <EmptyState icon={Plane} title="Sin Vuelo Seleccionado">Realiza una búsqueda y selecciona un vuelo de la lista de la izquierda para ver el mapa de asientos disponible.</EmptyState>
          )}
        </div>
      </div>
    </div>
  );
}
