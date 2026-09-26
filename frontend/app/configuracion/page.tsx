import LogoUploader from "@/components/LogoUploader";
import { Settings } from "lucide-react";
import { PageHeader } from "@/components/ui";

export default function Configuracion() {
  return (
    <div className="fade-up">
      <PageHeader icon={Settings} eyebrow="Operación" title="Configuración Global" subtitle="Ajustes del sistema y personalización de marca para todos los nodos BD." />

      <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
        <LogoUploader />

        <section className="card card-body space-y-5">
          <h2 className="section-title">Preferencias del Sistema</h2>
          <div>
            <label htmlFor="airline-name" className="field-label">Nombre de la Aerolínea</label>
            <input id="airline-name" type="text" defaultValue="AirRes Global" className="field" />
          </div>
          <div>
            <label htmlFor="sync-mode" className="field-label">Modo de Sincronización DB</label>
            <select id="sync-mode" className="field">
              <option value="realtime">En Tiempo Real (Recomendado)</option>
              <option value="batch">Asíncrono (Diferido 5m)</option>
            </select>
          </div>
          <button className="btn-primary w-full">Guardar Preferencias</button>
        </section>
      </div>
    </div>
  );
}
