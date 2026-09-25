"use client";

import { useEffect, useState } from "react";
import { CheckCircle2, Loader2, ShieldCheck, XCircle } from "lucide-react";

type Validation = {
  valid: boolean;
  id_boleto: number;
  id_vuelo: number;
  id_asiento: number;
  pasajero: string;
};

export default function VerificarBoleto({ params }: { params: { id: string } }) {
  const [result, setResult] = useState<Validation | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const token = new URLSearchParams(window.location.search).get("token");
    if (!token || !/^\d+$/.test(params.id)) {
      setError("Este código QR no contiene un boleto válido.");
      setLoading(false);
      return;
    }
    fetch(`/api/boletos/${params.id}/validar?token=${encodeURIComponent(token)}`, { cache: "no-store" })
      .then(async (response) => {
        if (!response.ok) throw new Error("El boleto no es válido o ya no está vigente.");
        return response.json() as Promise<Validation>;
      })
      .then(setResult)
      .catch((cause) => setError(cause instanceof Error ? cause.message : "No se pudo verificar el boleto."))
      .finally(() => setLoading(false));
  }, [params.id]);

  return <div className="mx-auto flex min-h-[65vh] max-w-lg items-center justify-center">
    <section className="w-full rounded-2xl border border-white/10 bg-[#202530] p-8 text-center shadow-2xl">
      {loading ? <Loader2 className="mx-auto h-12 w-12 animate-spin text-blue-400" /> : result?.valid ? <CheckCircle2 className="mx-auto h-14 w-14 text-green-400" /> : <XCircle className="mx-auto h-14 w-14 text-red-400" />}
      <h1 className="mt-5 text-2xl font-bold text-white">{loading ? "Verificando boleto…" : result?.valid ? "Boleto válido" : "No se pudo validar"}</h1>
      {result?.valid ? <div className="mt-6 space-y-2 text-left text-gray-200">
        <p><strong>Pasajero:</strong> {result.pasajero}</p>
        <p><strong>Boleto:</strong> #{result.id_boleto}</p>
        <p><strong>Vuelo:</strong> #{result.id_vuelo}</p>
        <p><strong>Asiento:</strong> #{result.id_asiento}</p>
      </div> : <p className="mt-4 text-sm text-gray-300">{error}</p>}
      <p className="mt-6 flex items-center justify-center gap-2 text-xs text-gray-400"><ShieldCheck size={15} /> Verificación en vivo · Aerolíneas Pabón</p>
    </section>
  </div>;
}
