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

  const valid = !loading && result?.valid;
  return <div className="mx-auto flex min-h-[65vh] max-w-md items-center justify-center">
    <section className="w-full overflow-hidden rounded-3xl bg-white text-center shadow-lift">
      <div className={`px-8 py-8 ${loading ? "bg-navy-900" : valid ? "bg-emerald-600" : "bg-red-600"} text-white`}>
        {loading ? <Loader2 className="mx-auto h-12 w-12 animate-spin" /> : valid ? <CheckCircle2 className="mx-auto h-14 w-14" /> : <XCircle className="mx-auto h-14 w-14" />}
        <h1 className="mt-4 text-2xl font-bold text-white">{loading ? "Verificando boleto…" : valid ? "Boleto válido" : "No se pudo validar"}</h1>
      </div>
      <div className="p-8">
        {valid && result ? <dl className="grid grid-cols-2 gap-4 text-left">
          <div className="col-span-2"><dt className="text-xs text-slate-500">Pasajero</dt><dd className="text-lg font-bold text-navy-900">{result.pasajero}</dd></div>
          <div><dt className="text-xs text-slate-500">Boleto</dt><dd className="font-semibold text-navy-900">#{result.id_boleto}</dd></div>
          <div><dt className="text-xs text-slate-500">Vuelo</dt><dd className="font-semibold text-navy-900">#{result.id_vuelo}</dd></div>
          <div><dt className="text-xs text-slate-500">Asiento</dt><dd className="font-semibold text-navy-900">#{result.id_asiento}</dd></div>
        </dl> : !loading && <p className="text-sm text-slate-600">{error}</p>}
        <p className="mt-6 flex items-center justify-center gap-2 border-t border-slate-100 pt-4 text-xs text-slate-500"><ShieldCheck size={15} /> Verificación en vivo · Aerolíneas Pabón</p>
      </div>
    </section>
  </div>;
}
