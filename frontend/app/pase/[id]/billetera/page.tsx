import Link from "next/link";

export default function DescargarPase({ params }: { params: { id: string } }) {
  const ticketID = Number(params.id);
  if (!Number.isSafeInteger(ticketID) || ticketID <= 0) {
    return <p className="text-red-300">Número de boleto inválido.</p>;
  }

  return (
    <div className="mx-auto max-w-xl space-y-6 rounded-2xl border border-blue-400/30 bg-slate-900 p-6 text-white shadow-xl">
      <div>
        <p className="text-sm font-semibold uppercase tracking-widest text-blue-300">Aerolíneas Pabón</p>
        <h2 className="mt-2 text-3xl font-bold">Pase de abordar #{ticketID}</h2>
        <p className="mt-2 text-sm text-slate-300">Descarga el archivo y guárdalo en Passbook desde tu iPhone.</p>
      </div>

      <a
        href={`/api/boletos/${ticketID}/wallet/demo.zip`}
        className="block rounded-xl bg-blue-600 px-5 py-4 text-center font-bold hover:bg-blue-500"
      >
        Descargar pase para Passbook (ZIP)
      </a>

      <ol className="list-decimal space-y-2 pl-5 text-sm text-slate-200">
        <li>En el iPhone, toca el botón y guarda el ZIP en Archivos.</li>
        <li>Abre Archivos → Descargas y toca el ZIP para descomprimirlo.</li>
        <li>Mantén pulsado el archivo <strong>AP-{ticketID}-demo.pkpass</strong>, elige Compartir y selecciona Passbook.</li>
      </ol>

      <p className="text-xs text-slate-400">
        El ZIP contiene el mismo .pkpass de demostración. Apple Wallet oficial no acepta pases sin firma; usa la aplicación Passbook con la que ya comprobaste la importación.
      </p>
      <div className="flex flex-wrap gap-4 text-sm">
        <Link href={`/gestion-boletos?boleto=${ticketID}`} className="text-blue-300 underline">Ver boleto en Gestión</Link>
      </div>
    </div>
  );
}
