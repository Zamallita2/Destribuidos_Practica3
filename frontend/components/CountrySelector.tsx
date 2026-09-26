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

  return (
    <div className="relative z-50">
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="flex items-center gap-2 px-4 py-2 glass-card hover:bg-white/10 transition-colors rounded-full text-sm font-medium border border-gray-700/50"
      >
        <Globe className="w-4 h-4 text-blue-400" />
        <span>{countries.find((country) => country.iso2 === selected.code) ? countryName(countries.find((country) => country.iso2 === selected.code)!) : selected.name}</span>
      </button>

      <AnimatePresence>
        {isOpen && (
          <motion.div
            initial={{ opacity: 0, y: 10, scale: 0.95 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: 10, scale: 0.95 }}
            transition={{ duration: 0.2 }}
            className="absolute right-0 mt-3 w-72 glass-panel p-2 shadow-2xl overflow-hidden"
          >
            <div className="p-2 border-b border-white/10 mb-2">
              <div className="relative">
                <Search className="absolute left-3 top-2.5 w-4 h-4 text-gray-500" />
                <input 
                  type="text" 
                  autoFocus
                  placeholder="Buscar país..."
                  className="w-full bg-white/5 pl-9 pr-3 py-2 rounded-lg text-sm text-white outline-none border border-transparent focus:border-blue-500/50 transition-all font-heading"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                />
              </div>
            </div>

            <div className="max-h-[300px] overflow-y-auto custom-scrollbar">
              {loading ? (
                <div className="p-10 flex justify-center"><Loader2 className="animate-spin text-blue-500" /></div>
              ) : filteredCountries.length === 0 ? (
                <div className="p-10 text-center text-gray-500 text-xs">No se encontraron resultados</div>
              ) : (
                filteredCountries.map((c) => (
                  <button
                    key={c.iso2}
                    onClick={() => handleSelect(c)}
                    className={`w-full flex items-center justify-between px-3 py-2.5 rounded-lg text-sm transition-colors group ${
                      selected.code === c.iso2 
                      ? "bg-blue-500/20 text-blue-400 font-bold" 
                      : "hover:bg-white/5 text-gray-300"
                    }`}
                  >
                    <div className="flex flex-col items-start translate-x-0 group-hover:translate-x-1 transition-transform">
                      <span className="font-heading">{countryName(c)}</span>
                      <span className="text-[10px] text-gray-500 font-bold uppercase tracking-tighter">{language === "en" ? "Server" : "Servidor"}: {c.server}</span>
                    </div>
                    {selected.code === c.iso2 && <Check className="w-4 h-4 text-blue-400" />}
                  </button>
                ))
              )}
            </div>
            
            <div className="p-2 mt-2 bg-blue-500/5 rounded-lg border border-blue-500/10">
              <p className="text-[9px] text-blue-400 text-center uppercase font-bold tracking-widest">Global Pabon-go Routing v2.0</p>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}
