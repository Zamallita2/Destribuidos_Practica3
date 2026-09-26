"use client";

import { useEffect, useState, type FormEvent } from "react";
import { Search, Info, User, Check, X, CreditCard, Plane, MapPin, Loader2, ArrowRight, Ticket } from "lucide-react";
import dynamic from "next/dynamic";
import { formatFlightLocalTime } from "@/lib/flightTime";
import { PURCHASE_CAPITALS } from "@/data/capitals";
import QRNetworkInfo from "@/components/QRNetworkInfo";
import { translateUiText } from "@/lib/englishUi";
import { useLanguage } from "@/context/LanguageContext";

const PlaneModelViewer = dynamic(() => import("@/components/PlaneModelViewer"), { 
  ssr: false,
  loading: () => (
    <div className="w-full h-[300px] bg-white/5 animate-pulse rounded-3xl flex items-center justify-center text-gray-600 font-bold uppercase text-[10px] tracking-widest border border-white/5">
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
  const [lastWrite, setLastWrite] = useState<{ ticketID: number; node: string } | null>(null);
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
    const id = flightIDSearch.trim();
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
      case 'AVAILABLE': return 'bg-gray-700/50 hover:bg-blue-500 border border-gray-600 hover:shadow-[0_0_15px_rgba(59,130,246,0.6)] cursor-pointer text-gray-300';
      case 'RESERVED': return 'bg-yellow-500/80 text-white cursor-pointer shadow-[0_0_10px_rgba(234,179,8,0.4)]';
      case 'SALED': return 'bg-emerald-600 text-white cursor-pointer border border-emerald-400 shadow-[0_0_15px_rgba(16,185,129,0.5)]';
      default: return 'bg-red-900/50 text-gray-500 cursor-not-allowed';
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
        setLastWrite({ ticketID: ticket.id_boleto, node: res.headers.get("X-Write-Node") || "" });
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
    return <div className="h-screen flex items-center justify-center"><Loader2 className="animate-spin text-blue-500 w-12 h-12" /></div>;
  }

  return (
    <div className="animate-in fade-in slide-in-from-bottom-5 duration-500 pb-20 max-w-7xl mx-auto">
      {lastWrite && <p className="mb-5 rounded-xl border border-blue-500/30 bg-blue-500/10 p-4 text-sm text-blue-100">Boleto #{lastWrite.ticketID} registrado primero en <strong>{lastWrite.node === "pg_am" ? "PostgreSQL América" : lastWrite.node === "pg_eu" ? "PostgreSQL Europa/Asia" : "el servidor disponible"}</strong>. La sincronización con los demás nodos puede verse en el panel de Sincronización.</p>}
      {purchasedPass && <section role="dialog" aria-label="Pase de abordar" className="glass-panel mb-8 flex flex-wrap items-center justify-between gap-5 border border-emerald-500/40 p-6">
        <div>
          <h3 className="text-xl font-bold text-emerald-300">Compra confirmada · Pase de abordar #{purchasedPass.id_boleto}</h3>
          <p>{purchasedPass.pasajero} · Vuelo AP-{purchasedPass.vuelo} · Asiento {purchasedPass.asiento}</p>
          <p className="text-sm text-gray-400">Sale de {purchasedPass.origen} (hora local): {formatFlightLocalTime(Math.floor(new Date(purchasedPass.salida_local).getTime() / 1000), purchasedPass.zona_salida, language)}</p>
          <p className="text-sm text-gray-400">Llega a {purchasedPass.destino} (hora local): {formatFlightLocalTime(Math.floor(new Date(purchasedPass.llegada_local).getTime() / 1000), purchasedPass.zona_llegada, language)}</p>
          <p className="text-xs text-gray-400">El boleto visual se descarga como PDF. Puedes volver a obtenerlo desde Gestión de Boletos.</p>
          <div className="mt-3 flex flex-wrap gap-2">
            <button onClick={() => downloadVisualTicket(purchasedPass.id_boleto)} disabled={pdfDownloading} className="rounded bg-blue-600 px-4 py-2 text-sm font-semibold text-white disabled:opacity-50">{pdfDownloading ? "Descargando PDF..." : "Descargar boleto visual (PDF)"}</button>
            <a href={`/pase/${purchasedPass.id_boleto}/billetera`} className="rounded bg-white px-4 py-2 text-sm font-semibold text-black">Abrir pase para Passbook</a>
            <a href={`/gestion-boletos?boleto=${purchasedPass.id_boleto}`} className="rounded border border-white/20 px-4 py-2 text-sm font-semibold text-white">Ver en Gestión de Boletos</a>
          </div>
          <p className="mt-2 text-xs text-gray-400">En iPhone, descarga el ZIP y comparte el .pkpass desde Archivos a Passbook. Apple Wallet oficial requiere firma de emisor.</p>
          {walletCapabilities.apple && <a href={`/api/boletos/${purchasedPass.id_boleto}/wallet/apple.pkpass`} className="mr-4 mt-3 inline-block rounded bg-white px-4 py-2 text-sm font-semibold text-black">Añadir a Apple Wallet</a>}
          {googleWalletURL && <a href={googleWalletURL} target="_blank" rel="noopener noreferrer" className="mt-3 inline-block rounded bg-white px-4 py-2 text-sm font-semibold text-black">Añadir a Google Wallet</a>}
          <button onClick={() => setPurchasedPass(null)} className="mt-3 text-sm text-blue-300 underline">Cerrar</button>
        </div>
        <figure className="text-center text-xs text-gray-300"><img src={`/api/boletos/${purchasedPass.id_boleto}/wallet/qr.png`} width={160} height={160} alt="QR para abrir la guía de descarga del pase" className="mx-auto mb-2 rounded bg-white p-2" /><figcaption>Escanea con la cámara del celular para abrir el pase</figcaption></figure>
        <p className="w-full text-xs text-gray-400">El QR abre una página con la descarga y los pasos para importar el pase en Passbook. Usa la cámara del celular, no el lector de códigos de la app. Ambos dispositivos deben estar en la misma red.</p>
        <QRNetworkInfo />
      </section>}
      <div className="mb-10 text-center">
        <h2 className="text-4xl font-bold bg-clip-text text-transparent bg-gradient-to-r from-blue-400 to-purple-400 flex items-center justify-center gap-3">
          <Ticket className="text-blue-500 w-10 h-10" /> Venta de Boletos
        </h2>
        <p className="text-gray-400 mt-3 text-lg">Busca tu destino, selecciona tu asiento y vuela con Pabon-go.</p>
        <a href="/gestion-boletos" className="mt-4 inline-block rounded-lg border border-blue-400/40 px-4 py-2 text-sm font-semibold text-blue-300 hover:bg-blue-500/10">Ya compré un boleto · Ir a Gestión de Boletos</a>
      </div>

      {/* SEARCH BAR */}
      <form onSubmit={handleFlightIDSearch} className="glass-panel mb-5 flex flex-wrap items-end gap-3 border-blue-500/20 p-5">
        <label htmlFor="flight-id-search" className="min-w-[220px] flex-1 text-sm font-semibold text-blue-200">
          Buscar vuelo por ID
          <input id="flight-id-search" type="number" min="1" inputMode="numeric" value={flightIDSearch} onChange={(event) => setFlightIDSearch(event.target.value)} placeholder="ID que aparece en Vuelos" className="mt-2 w-full rounded-xl border border-white/10 bg-white/5 px-4 py-3 text-white outline-none focus:border-blue-500" />
        </label>
        <button type="submit" disabled={flightSearchLoading} className="flex items-center gap-2 rounded-xl bg-blue-600 px-6 py-3 font-bold text-white hover:bg-blue-500 disabled:opacity-50">
          {flightSearchLoading ? <Loader2 className="h-5 w-5 animate-spin" /> : <Search className="h-5 w-5" />} Buscar por ID
        </button>
        {flightSearchError && <p role="alert" className="w-full text-sm text-amber-300">{flightSearchError}</p>}
      </form>
      <p className="mb-5 text-center text-sm text-gray-400">También puedes buscar por origen y destino:</p>
      <div className="glass-panel p-6 mb-10 flex flex-wrap items-end gap-6 justify-center shadow-2xl border-white/5">
        <div className="flex-1 min-w-[250px]">
          <label className="text-xs font-bold text-blue-400 uppercase tracking-wider mb-2 block">Origen</label>
          <div className="relative space-y-2">
            <div className="relative">
              <Search className="absolute left-3 top-3 w-4 h-4 text-gray-400" />
              <input
                type="text"
                placeholder="Filtrar ciudad/país..."
                value={originSearch}
                onChange={(e) => setOriginSearch(e.target.value)}
                className="w-full bg-white/5 border border-white/10 rounded-lg py-1.5 pl-9 pr-3 text-xs text-white placeholder:text-gray-500 outline-none focus:border-blue-500"
              />
            </div>
            <div className="relative">
              <MapPin className="absolute left-3 top-3 w-5 h-5 text-gray-500" />
              <select 
                value={selectedOrigin} 
                onChange={(e) => setSelectedOrigin(e.target.value)}
                className="w-full bg-white/5 border border-white/10 rounded-xl py-3 pl-11 pr-4 text-white outline-none focus:border-blue-500 transition appearance-none"
              >
                <option value="" className="text-black">Seleccione Ciudad ({filteredOriginCiudades.length})...</option>
                {filteredOriginCiudades.map((c: any) => <option key={c.id} value={c.id} className="text-black">{c.pais} - {c.codigo}</option>)}
              </select>
            </div>
          </div>
        </div>
        <ArrowRight className="mb-3 text-gray-600 hidden md:block" />
        <div className="flex-1 min-w-[250px]">
          <label className="text-xs font-bold text-purple-400 uppercase tracking-wider mb-2 block">Destino</label>
          <div className="relative space-y-2">
            <div className="relative">
              <Search className="absolute left-3 top-3 w-4 h-4 text-gray-400" />
              <input
                type="text"
                placeholder="Filtrar ciudad/país..."
                value={destinationSearch}
                onChange={(e) => setDestinationSearch(e.target.value)}
                className="w-full bg-white/5 border border-white/10 rounded-lg py-1.5 pl-9 pr-3 text-xs text-white placeholder:text-gray-500 outline-none focus:border-purple-500"
              />
            </div>
            <div className="relative">
              <MapPin className="absolute left-3 top-3 w-5 h-5 text-gray-500" />
              <select 
                value={selectedDestination} 
                onChange={(e) => setSelectedDestination(e.target.value)}
                className="w-full bg-white/5 border border-white/10 rounded-xl py-3 pl-11 pr-4 text-white outline-none focus:border-purple-500 transition appearance-none"
              >
                <option value="" className="text-black">Seleccione Ciudad ({filteredDestinationCiudades.length})...</option>
                {filteredDestinationCiudades.map((c: any) => <option key={c.id} value={c.id} className="text-black">{c.pais} - {c.codigo}</option>)}
              </select>
            </div>
          </div>
        </div>
        <button 
          onClick={handleSearch}
          className="bg-blue-600 hover:bg-blue-500 text-white px-8 py-3 rounded-xl font-bold flex items-center gap-2 transition shadow-lg shadow-blue-600/20 active:scale-95"
        >
          <Search className="w-5 h-5" /> Buscar Vuelos
        </button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-10">
        {/* FLIGHT LIST */}
        <div className="lg:col-span-1 space-y-4">
          <h3 className="text-xl font-bold mb-4 flex items-center gap-2 text-white/80"><Plane className="w-5 h-5 text-blue-400" /> Vuelos Disponibles</h3>
          {filteredVuelos.length === 0 ? (
            <div className="glass-panel p-10 text-center text-gray-500 italic border-dashed border-2 border-white/5">
              No hay vuelos que coincidan con tu búsqueda.
            </div>
          ) : (
            filteredVuelos.map((v: any) => (
              <div 
                key={v.id} 
                onClick={() => loadSeats(v)}
                className={`glass-panel p-5 cursor-pointer transition-all hover:border-blue-500/50 group ${selectedVuelo?.id === v.id ? 'border-blue-500 bg-blue-500/10' : 'hover:scale-[1.02]'}`}
              >
                <p className="mb-3 text-xs font-semibold text-blue-300">Vuelo AP-{v.id}</p>
                <div className="flex justify-between items-start mb-4">
                  <div className="bg-white/10 p-2 rounded-lg group-hover:bg-blue-500/20 transition">
                    <Plane className="w-6 h-6 text-blue-400" />
                  </div>
                  <div className="text-right">
                    <p className="text-[10px] text-gray-500 uppercase font-bold">Desde</p>
                    <p className="text-2xl font-bold text-white">$ {getPrice(v, 'REGULAR') || getPrice(v, 'VIP')}</p>
                  </div>
                </div>
                <div className="flex items-center justify-between text-sm">
                   <div>
                      <p className="text-gray-400 font-medium">Sale · hora local {(ciudades.find((c:any) => c.id === v.id_origen) as any)?.codigo}</p>
                      <p className="text-white font-bold">{formatFlightLocalTime(v.salida_programada, (ciudades.find((c:any) => c.id === v.id_origen) as any)?.time_zone, language)}</p>
                   </div>
                   <div className="h-px bg-white/20 flex-1 mx-4 relative">
                      <div className="absolute -top-1 right-0 w-2 h-2 rounded-full bg-blue-500" />
                   </div>
                   <div className="text-right">
                      <p className="text-gray-400 font-medium">Llega · hora local {(ciudades.find((c:any) => c.id === v.id_destino) as any)?.codigo}</p>
                      <p className="text-white font-bold">{formatFlightLocalTime(v.llegada_programada, (ciudades.find((c:any) => c.id === v.id_destino) as any)?.time_zone, language)}</p>
                   </div>
                </div>
              </div>
            ))
          )}
        </div>

        {/* SEATING AREA */}
        <div className="lg:col-span-2">
          {selectedVuelo ? (
            <div className="grid grid-cols-1 xl:grid-cols-5 gap-8">
              <div className="xl:col-span-3 glass-panel p-8">
                <div className="flex justify-between items-center mb-10">
                  <h3 className="font-bold text-xl">Mapa de Asientos</h3>
                  <div className="bg-white/5 px-4 py-2 rounded-full border border-white/10 flex items-center gap-2 text-xs font-medium text-gray-400">
                    Avión ID: {selectedVuelo.id_avion}
                  </div>
                </div>

                <div className="mb-10">
                   <PlaneModelViewer aircraftName={aviones.find((aircraft) => aircraft.id === selectedVuelo.id_avion)?.nombre} />
                </div>

                {/* DYNAMIC SCROLLABLE SEATING MAP */}
                <div className="max-h-[600px] overflow-y-auto pr-4 scrollbar-thin scrollbar-thumb-white/10">
                  <div className="max-w-xs mx-auto bg-white/5 rounded-[3rem] p-8 border border-white/10 relative shadow-2xl min-h-[500px]">
                    <div className="w-16 h-24 absolute -top-12 left-1/2 -translate-x-1/2 bg-white/10 rounded-t-full rounded-b-xl border border-white/20" />
                    
                    {/* VIP SECTION */}
                    <div className="text-center text-[10px] text-purple-400 font-bold tracking-[0.2em] uppercase mb-6 mt-8 border-b border-purple-900/40 pb-2">Primera Clase</div>
                    <div className="grid grid-cols-4 gap-x-3 gap-y-4 mb-10">
                      {asientos.filter((s:any) => s.clase === 'VIP').map((seat:any) => (
                        <button 
                          key={`${selectedVuelo?.id}-${seat.id}-${seat.codigo}`}
                          onClick={() => {
                            setSelectedSeat(seat);
                            if (seat.estado !== 'AVAILABLE') {
                              setPassenger({
                                nombre: seat.nombre_pasajero || "",
                                email: seat.email_pasajero || "",
                                pasaporte: seat.pasaporte || ""
                              });
                            } else {
                              setPassenger({ nombre: "", email: "", pasaporte: "" });
                            }
                          }}
                          className={`w-10 h-12 rounded-t-xl rounded-b-md shadow-lg flex items-center justify-center transition-all ${selectedSeat?.id === seat.id ? 'ring-2 ring-blue-500 ring-offset-4 ring-offset-zinc-900 scale-110 z-10' : ''} ${colorPorEstado(seat.estado)}`}
                        >
                          <span className="text-[10px] font-bold">{seat.codigo}</span>
                        </button>
                      ))}
                    </div>

                    {/* REGULAR SECTION */}
                    <div className="text-center text-[10px] text-blue-400 font-bold tracking-[0.2em] uppercase mb-6 border-b border-blue-900/40 pb-2">Clase Económica</div>
                    <div className="grid grid-cols-7 gap-x-2 gap-y-3">
                      {asientos.filter((s:any) => s.clase === 'REGULAR').map((seat:any, idx) => {
                        const seatElement = (
                          <button 
                            key={`${selectedVuelo?.id}-${seat.id}-${seat.codigo}`}
                            onClick={() => {
                              setSelectedSeat(seat);
                              if (seat.estado !== 'AVAILABLE') {
                                setPassenger({
                                  nombre: seat.nombre_pasajero || "",
                                  email: seat.email_pasajero || "",
                                  pasaporte: seat.pasaporte || ""
                                });
                              } else {
                                setPassenger({ nombre: "", email: "", pasaporte: "" });
                              }
                            }}
                            className={`w-8 h-10 rounded-t-lg rounded-b shadow flex items-center justify-center transition-all ${selectedSeat?.id === seat.id ? 'ring-2 ring-blue-500 ring-offset-4 ring-offset-zinc-900 scale-110 z-10' : ''} ${colorPorEstado(seat.estado)}`}
                          >
                            <span className="text-[9px] font-bold">{seat.codigo}</span>
                          </button>
                        );

                        // If we are at the 4th position (index 3, 10, 17...) we insert an empty div for the aisle
                        if (idx % 6 === 3) {
                          return (
                            <>
                              <div key={`aisle-${idx}`} className="col-span-1" />
                              {seatElement}
                            </>
                          );
                        }
                        return seatElement;
                      })}
                    </div>
                  </div>
                </div>

                <div className="mt-8 flex flex-wrap gap-6 items-center justify-center text-[10px] font-bold uppercase tracking-wider text-gray-500 border-t border-white/5 pt-8">
                  <div className="flex items-center gap-2"><div className="w-3 h-3 rounded bg-gray-700/50 border border-gray-600"/> Disponible</div>
                  <div className="flex items-center gap-2"><div className="w-3 h-3 rounded bg-yellow-500/80"/> Reservado</div>
                  <div className="flex items-center gap-2"><div className="w-3 h-3 rounded bg-emerald-600"/> Vendido</div>
                </div>
              </div>

              {/* BOOKING PANEL */}
              <div className="xl:col-span-2 glass-panel p-6 h-max sticky top-32">
                {selectedSeat ? (
                  <div className="animate-in fade-in zoom-in duration-300">
                    <div className="flex items-center justify-between mb-8">
                      <div>
                        <h3 className="text-3xl font-bold text-white tracking-tight">Asiento {selectedSeat.codigo}</h3>
                        <span className={`text-[10px] px-3 py-1 rounded-full font-black uppercase tracking-widest ${selectedSeat.clase === 'VIP' ? 'bg-purple-900/60 text-purple-300' : 'bg-blue-900/60 text-blue-300'}`}>
                          {selectedSeat.clase === 'VIP' ? 'Primera Clase' : 'Económico'}
                        </span>
                      </div>
                      <div className="text-right">
                         <p className="text-[10px] text-gray-500 font-bold uppercase">Precio Final</p>
                         <p className="text-2xl font-black text-white">$ {getPrice(selectedVuelo, selectedSeat.clase)}</p>
                      </div>
                    </div>

                    <div className="space-y-4">
                       <div className="p-4 bg-white/5 rounded-2xl border border-white/10 space-y-3">
                          <div className="flex items-center gap-3">
                             <div className="w-10 h-10 rounded-full bg-blue-500/20 flex items-center justify-center text-blue-400">
                                <Plane className="w-5 h-5" />
                             </div>
                             <div>
                                <p className="text-xs text-gray-500 font-bold uppercase">Ruta</p>
                                <p className="text-sm font-bold text-white">
                                  {(ciudades.find((c:any) => c.id === selectedVuelo.id_origen) as any)?.codigo} 
                                  <ArrowRight className="inline w-3 h-3 mx-2 text-gray-600" />
                                  {(ciudades.find((c:any) => c.id === selectedVuelo.id_destino) as any)?.codigo}
                                </p>
                             </div>
                          </div>
                          <div className="h-px bg-white/5" />
                       </div>
                       
                       <div className="space-y-4 pt-2">
                          <p className="text-[10px] text-gray-500 font-bold uppercase tracking-widest">Información del Pasajero</p>
                          <label className="block text-xs text-gray-300">
                            Capital desde donde compras (elige el servidor inicial)
                            <select value={buyerTimeZone} onChange={(event) => setBuyerTimeZone(event.target.value)} className="mt-1 w-full rounded-xl border border-white/10 bg-slate-900 px-4 py-2.5 text-white">
                              {PURCHASE_CAPITALS.map(([capital, zone]) => <option key={capital} value={zone}>{capital}</option>)}
                            </select>
                          </label>
                          <p className="text-xs text-gray-400">América registra primero en PostgreSQL América; Europa y Asia en PostgreSQL Europa/Asia. El país de arriba solo elige de dónde se consultan las listas. La salida usa la hora del aeropuerto de origen y la llegada la del destino.</p>
                          <div className="space-y-3">
                             <input 
                                type="text" 
                                placeholder="Nombre Completo"
                                value={passenger.nombre}
                                disabled={selectedSeat.estado !== 'AVAILABLE'}
                                onChange={(e) => setPassenger({...passenger, nombre: e.target.value})}
                                className="w-full bg-white/5 border border-white/10 rounded-xl py-2.5 px-4 text-sm text-white placeholder:text-gray-600 outline-none focus:border-blue-500/50 transition disabled:opacity-50 disabled:cursor-not-allowed"
                             />
                             <input 
                                type="email" 
                                placeholder="Email"
                                value={passenger.email}
                                disabled={selectedSeat.estado !== 'AVAILABLE'}
                                onChange={(e) => setPassenger({...passenger, email: e.target.value})}
                                className="w-full bg-white/5 border border-white/10 rounded-xl py-2.5 px-4 text-sm text-white placeholder:text-gray-600 outline-none focus:border-blue-500/50 transition disabled:opacity-50 disabled:cursor-not-allowed"
                             />
                             <input 
                                type="text" 
                                placeholder="Número de Pasaporte"
                                value={passenger.pasaporte}
                                disabled={selectedSeat.estado !== 'AVAILABLE'}
                                onChange={(e) => setPassenger({...passenger, pasaporte: e.target.value})}
                                className="w-full bg-white/5 border border-white/10 rounded-xl py-2.5 px-4 text-sm text-white placeholder:text-gray-600 outline-none focus:border-blue-500/50 transition disabled:opacity-50 disabled:cursor-not-allowed"
                             />
                          </div>
                          <div className="bg-blue-500/10 p-3 rounded-xl border border-blue-500/20 flex items-center justify-between">
                             <span className="text-[10px] text-blue-400 font-bold uppercase">Viaje Estimado</span>
                             <span className="text-xs font-bold text-white">{Math.round((selectedVuelo.llegada_programada - selectedVuelo.salida_programada) / 3600)} Horas</span>
                          </div>
                       </div>

                       {selectedSeat.estado === "AVAILABLE" ? (
                         <div className="space-y-3 pt-4">
                            <button 
                              onClick={() => procesarBoleto('RESERVED')} 
                              disabled={bookingLoading}
                              className="w-full flex items-center justify-center gap-3 py-4 rounded-2xl bg-white/5 hover:bg-white/10 text-yellow-400 font-bold transition border border-yellow-500/30 active:scale-[0.98]"
                            >
                               {bookingLoading ? <Loader2 className="animate-spin w-5 h-5" /> : <><Check className="w-5 h-5" /> Pre-Reservar (72h)</>}
                            </button>
                            <button 
                              onClick={() => procesarBoleto('SALED')} 
                              disabled={bookingLoading}
                              className="w-full flex items-center justify-center gap-3 py-4 rounded-2xl bg-blue-600 hover:bg-blue-500 text-white font-black transition shadow-xl shadow-blue-600/30 active:scale-[0.98]"
                            >
                               {bookingLoading ? <Loader2 className="animate-spin w-5 h-5" /> : <><CreditCard className="w-5 h-5" /> Comprar Ticket Ahora</>}
                            </button>
                         </div>
                       ) : (
                         <div className="p-6 bg-red-500/10 rounded-2xl border border-red-500/30 text-center">
                            <X className="w-12 h-12 text-red-500 mx-auto mb-4" />
                            <p className="font-bold text-white">Asiento Ocupado</p>
                            <p className="text-sm text-gray-400 mt-2">Este asiento ya ha sido reservado o vendido. Por favor selecciona otro.</p>
                         </div>
                       )}
                    </div>
                  </div>
                ) : (
                  <div className="h-[400px] flex flex-col items-center justify-center text-center px-6">
                    <div className="w-20 h-20 rounded-3xl bg-white/5 border border-white/10 flex items-center justify-center mb-6 animate-pulse">
                      <Info className="text-gray-600 w-10 h-10" />
                    </div>
                    <h4 className="text-xl font-bold text-white mb-2">Selección de Asiento</h4>
                    <p className="text-sm text-gray-500 leading-relaxed">Haz clic en un asiento del mapa para ver los detalles del precio y completar tu reserva.</p>
                  </div>
                )}
              </div>
            </div>
          ) : (
            <div className="h-full min-h-[500px] glass-panel flex flex-col items-center justify-center text-center p-10 opacity-50 border-dashed border-2">
              <Plane className="w-20 h-20 text-gray-600 mb-6" />
              <h3 className="text-2xl font-bold text-white mb-2">Sin Vuelo Seleccionado</h3>
              <p className="max-w-md">Realiza una búsqueda y selecciona un vuelo de la lista de la izquierda para ver el mapa de asientos disponible.</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
