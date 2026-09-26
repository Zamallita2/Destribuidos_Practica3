"use client";

import { useState } from "react";
import { UploadCloud, Check } from "lucide-react";

export default function LogoUploader() {
  const [dragActive, setDragActive] = useState(false);
  const [uploaded, setUploaded] = useState<string | null>(null);

  const handleDrag = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (e.type === "dragenter" || e.type === "dragover") {
      setDragActive(true);
    } else if (e.type === "dragleave") {
      setDragActive(false);
    }
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setDragActive(false);
    if (e.dataTransfer.files && e.dataTransfer.files[0]) {
      handleFiles(e.dataTransfer.files[0]);
    }
  };

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    e.preventDefault();
    if (e.target.files && e.target.files[0]) {
      handleFiles(e.target.files[0]);
    }
  };

  const handleFiles = (file: File) => {
    // In a real application, send this file to the Go backend via FormData
    const url = URL.createObjectURL(file);
    setUploaded(url);
    // Simulation:
    setTimeout(() => {
      // alert("Logo guardado exitosamente");
    }, 500);
  };

  return (
    <section className="card card-body w-full">
      <h2 className="section-title">Personaliza tu Aerolínea</h2>
      <p className="section-subtitle mb-5">Sube aquí el logotipo corporativo para visualizarlo en los recibos y el portal principal.</p>

      <form
        className={`relative flex h-64 w-full flex-col items-center justify-center rounded-2xl border-2 border-dashed transition ${
          dragActive ? "border-navy-500 bg-navy-50" : "border-slate-300 bg-slate-50 hover:border-navy-300"
        }`}
        onDragEnter={handleDrag}
        onDragLeave={handleDrag}
        onDragOver={handleDrag}
        onDrop={handleDrop}
      >
        <input type="file" id="file-upload" className="hidden" accept="image/*" onChange={handleChange} />

        {uploaded ? (
          <div className="fade-up flex flex-col items-center">
            <div className="mb-4 h-24 w-24 overflow-hidden rounded-2xl border border-slate-200 bg-white p-2 shadow-card">
              <img src={uploaded} alt="Logo de Aerolínea" className="h-full w-full object-contain" />
            </div>
            <div className="flex items-center gap-2 font-medium text-emerald-700"><Check className="h-5 w-5" /> Logo Asignado</div>
          </div>
        ) : (
          <label htmlFor="file-upload" className="flex h-full w-full cursor-pointer flex-col items-center justify-center">
            <span className="mb-4 flex h-14 w-14 items-center justify-center rounded-2xl bg-white text-navy-600 shadow-card"><UploadCloud className="h-7 w-7" /></span>
            <p className="font-semibold text-navy-900">Arrastra tu logo aquí</p>
            <p className="mt-1 text-sm text-slate-500">Formatos: PNG, JPG, SVG</p>
            <span className="btn-secondary btn-sm mt-5">Explorar archivos</span>
          </label>
        )}
      </form>
    </section>
  );
}
