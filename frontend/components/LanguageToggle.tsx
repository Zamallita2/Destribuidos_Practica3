"use client";

import { useLanguage } from "@/context/LanguageContext";

export default function LanguageToggle() {
  const { language, setLanguage } = useLanguage();
  return <div role="group" aria-label="Idioma" className="inline-flex rounded-full border border-slate-200 bg-slate-100 p-0.5 text-xs font-bold">
    {(["es", "en"] as const).map((code) => <button key={code} type="button" onClick={() => setLanguage(code)} aria-pressed={language === code}
      className={`rounded-full px-3 py-1.5 transition ${language === code ? "bg-navy-900 text-white shadow-sm" : "text-slate-500 hover:text-navy-800"}`}>
      {code.toUpperCase()}
    </button>)}
  </div>;
}
