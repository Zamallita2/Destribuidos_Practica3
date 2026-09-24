"use client";

import { useEffect, useState } from "react";
import { useLanguage } from "@/context/LanguageContext";

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

  return (
    <main className="mx-auto max-w-5xl space-y-6 p-8">
      <div>
        <h1 className="text-3xl font-bold">{es ? "Agente Viajero (TSP)" : "Traveling Salesperson (TSP)"}</h1>
        <p className="mt-2 text-gray-400">
          {es
            ? "Ruta óptima de la red dirigida. Los tramos representan rutas de la matriz, no reservas con horario."
            : "Optimal path on the directed route network. Legs are matrix routes, not scheduled bookings."}
        </p>
      </div>
      <div className="glass-panel space-y-5 p-6">
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-5">
          {cities.map((city) => (
            <label key={city.codigo} className="flex cursor-pointer gap-2 rounded-lg border border-white/10 p-3">
              <input type="checkbox" checked={selected.includes(city.codigo)} onChange={() => toggle(city.codigo)} />
              <span>{city.codigo}</span>
            </label>
          ))}
        </div>
        <div className="flex flex-wrap gap-4">
          <label className="space-y-1">
            <span className="block text-sm text-gray-400">{es ? "Optimizar" : "Optimize"}</span>
            <select value={criterion} onChange={(event) => setCriterion(event.target.value as "COST" | "TIME")} className="glass-card p-2">
              <option value="COST">{es ? "Costo" : "Cost"}</option>
              <option value="TIME">{es ? "Tiempo" : "Time"}</option>
            </select>
          </label>
          <label className="space-y-1">
            <span className="block text-sm text-gray-400">{es ? "Clase" : "Class"}</span>
            <select value={seatClass} onChange={(event) => setSeatClass(event.target.value as "REGULAR" | "VIP")} className="glass-card p-2">
              <option value="REGULAR">{es ? "Turística" : "Economy"}</option>
              <option value="VIP">{es ? "Primera" : "First"}</option>
            </select>
          </label>
          <label className="flex items-center gap-2 self-end pb-2">
            <input type="checkbox" checked={returnToOrigin} onChange={(event) => setReturnToOrigin(event.target.checked)} />
            {es ? "Regresar al origen" : "Return to start"}
          </label>
        </div>
        <button disabled={selected.length < 2 || loading} onClick={calculate} className="btn-primary disabled:opacity-50">
          {loading ? (es ? "Calculando…" : "Calculating…") : (es ? "Calcular ruta" : "Calculate route")}
        </button>
      </div>
      {message && <p role="alert" className="text-rose-400">{message}</p>}
      {result && (
        <div className="glass-panel space-y-4 p-6">
          <h2 className="text-xl font-semibold">{result.ruta.join(" → ")}</h2>
          <div className="flex gap-8">
            <span>{es ? "Costo" : "Cost"}: ${result.costo.toFixed(2)}</span>
            <span>{es ? "Tiempo" : "Time"}: {result.tiempo} h</span>
          </div>
          <ol className="space-y-2">
            {result.vuelos.map((leg, index) => (
              <li key={index} className="rounded-lg border border-white/10 p-3">
                {leg.salida} → {leg.llegada} · ${leg.cost.toFixed(2)} · {leg.time} h
              </li>
            ))}
          </ol>
        </div>
      )}
    </main>
  );
}
