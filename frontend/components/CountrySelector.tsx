"use client";

import { useEffect, useState } from "react";
import { Globe, Check, Search, Loader2 } from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";
import { useLanguage } from "@/context/LanguageContext";

interface Country {
  nameES: string;
  nameEN: string;
  iso2: string;
  server: string;
}

export default function CountrySelector() {
  const { language } = useLanguage();
  const countryName = (country: Country) => language === "en"
    ? new Intl.DisplayNames(["en"], { type: "region" }).of(country.iso2) || country.nameEN
    : country.nameES;
  const [isOpen, setIsOpen] = useState(false);
  const [countries, setCountries] = useState<Country[]>([]);
  const [searchQuery, setSearchQuery] = useState("");
  const [selected, setSelected] = useState<any>({ name: "Estados Unidos", code: "US", region: "America" });
  const [loading, setLoading] = useState(true);

  // Load and Persist
  useEffect(() => {
    const loadData = async () => {
      try {
        const res = await fetch("/countries.json");
        const data = await res.json();
        setCountries(data);
        
        const saved = localStorage.getItem("airres-country");
        if (saved) {
          const parsed = JSON.parse(saved);
          setSelected(parsed);
        } else {
          // Initialize US as default
          const us = data.find((c: Country) => c.iso2 === "US");
          if (us) {
            const initial = { name: us.nameES, code: us.iso2, region: us.server };
            setSelected(initial);
            localStorage.setItem("airres-country", JSON.stringify(initial));
          }
        }
      } catch (e) {
        console.error("Error loading countries:", e);
      } finally {
        setLoading(false);
      }
    };
    loadData();
  }, []);

  const handleSelect = (c: Country) => {
    const fresh = { name: c.nameES, code: c.iso2, region: c.server };
    setSelected(fresh);
    localStorage.setItem("airres-country", JSON.stringify(fresh));
    setIsOpen(false);
    window.location.reload(); 
  };

  const filteredCountries = countries.filter(c => 
    c.nameES.toLowerCase().includes(searchQuery.toLowerCase()) || 
    c.nameEN.toLowerCase().includes(searchQuery.toLowerCase()) ||
    c.iso2.toLowerCase().includes(searchQuery.toLowerCase())
  ).slice(0, 50); // Limit to top 50 for performance

  const current = countries.find((country) => country.iso2 === selected.code);
  return (
    <div className="relative z-50">
      <button
        onClick={() => setIsOpen(!isOpen)}
        aria-expanded={isOpen}
        aria-label="País de compra"
        className="flex items-center gap-2 rounded-full border border-slate-200 bg-white px-3 py-1.5 text-sm font-medium text-navy-900 shadow-sm transition hover:border-navy-300"
      >
        <Globe className="h-4 w-4 text-navy-500" />
        <span className="max-w-[120px] truncate sm:max-w-none">{current ? countryName(current) : selected.name}</span>
        <span className="hidden rounded-full bg-navy-50 px-2 py-0.5 text-[10px] font-bold uppercase text-navy-600 sm:inline">{selected.region}</span>
      </button>

      <AnimatePresence>
        {isOpen && (
          <motion.div
            initial={{ opacity: 0, y: 8, scale: 0.98 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: 8, scale: 0.98 }}
            transition={{ duration: 0.15 }}
            className="absolute right-0 mt-2 w-72 overflow-hidden rounded-2xl border border-slate-200 bg-white p-2 shadow-lift"
          >
            <div className="relative mb-2 p-1">
              <Search className="absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
              <input
                type="text"
                autoFocus
                placeholder="Buscar país..."
                className="field pl-9"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
              />
            </div>

            <div className="max-h-[300px] overflow-y-auto">
              {loading ? (
                <div className="flex justify-center p-10"><Loader2 className="animate-spin text-navy-500" /></div>
              ) : filteredCountries.length === 0 ? (
                <div className="p-10 text-center text-xs text-slate-500">No se encontraron resultados</div>
              ) : (
                filteredCountries.map((c) => (
                  <button
                    key={c.iso2}
                    onClick={() => handleSelect(c)}
                    className={`flex w-full items-center justify-between rounded-lg px-3 py-2.5 text-left text-sm transition-colors ${
                      selected.code === c.iso2 ? "bg-navy-50 font-semibold text-navy-900" : "text-slate-700 hover:bg-slate-50"
                    }`}
                  >
                    <span className="flex flex-col">
                      <span>{countryName(c)}</span>
                      <span className="text-[10px] font-semibold uppercase tracking-wide text-slate-400">{language === "en" ? "Server" : "Servidor"}: {c.server}</span>
                    </span>
                    {selected.code === c.iso2 && <Check className="h-4 w-4 text-navy-600" />}
                  </button>
                ))
              )}
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}
