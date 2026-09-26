import Link from "next/link";
import { Download, Wallet } from "lucide-react";

export default function DescargarPase({ params }: { params: { id: string } }) {
  const ticketID = Number(params.id);
  if (!Number.isSafeInteger(ticketID) || ticketID <= 0) {
    return <p role="alert" className="alert alert-error">Número de boleto inválido.</p>;
  }

  return (
    <div className="mx-auto max-w-xl overflow-hidden rounded-3xl bg-white shadow-lift">
      <div className="bg-navy-900 p-6 text-white sm:p-8">
        <p className="text-xs font-semibold uppercase tracking-[0.2em] text-gold-300">Aerolíneas Pabón</p>
        <h1 className="mt-2 flex items-center gap-3 text-3xl font-bold text-white"><Wallet className="h-7 w-7 text-gold-300" aria-hidden="true" />Pase de abordar #{ticketID}</h1>
        <p className="mt-2 text-sm text-navy-200">Descarga el archivo y guárdalo en Passbook desde tu iPhone.</p>
      </div>
      <div className="space-y-6 p-6 sm:p-8">
        <a href={`/api/boletos/${ticketID}/wallet/demo.zip`} className="btn-gold btn-lg w-full">
          <Download className="h-5 w-5" aria-hidden="true" /> Descargar pase para Passbook (ZIP)
        </a>
        <ol className="space-y-3 text-sm text-slate-700">
          {[
            <>En el iPhone, toca el botón y guarda el ZIP en Archivos.</>,
            <>Abre Archivos → Descargas y toca el ZIP para descomprimirlo.</>,
            <>Mantén pulsado el archivo <strong>AP-{ticketID}-demo.pkpass</strong>, elige Compartir y selecciona Passbook.</>,
          ].map((text, index) => <li key={index} className="flex gap-3">
            <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-navy-900 text-xs font-bold text-gold-300">{index + 1}</span>
            <span className="pt-0.5">{text}</span>
          </li>)}
        </ol>
        <p className="alert alert-info text-xs">
          El ZIP contiene el mismo .pkpass de demostración. Apple Wallet oficial no acepta pases sin firma; usa la aplicación Passbook con la que ya comprobaste la importación.
        </p>
        <Link href={`/gestion-boletos?boleto=${ticketID}`} className="btn-secondary w-full">Ver boleto en Gestión</Link>
      </div>
    </div>
  );
}
