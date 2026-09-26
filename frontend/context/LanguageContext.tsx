"use client";

import React, { createContext, useContext, useState, useEffect, useRef } from "react";
import es from "../locales/es.json";
import en from "../locales/en.json";
import { translateUiText } from "@/lib/englishUi";

type Language = "es" | "en";

interface LanguageContextProps {
  language: Language;
  setLanguage: (lang: Language) => void;
  t: (key: string) => string;
}

const translations = {
  es,
  en,
};

const LanguageContext = createContext<LanguageContextProps | undefined>(undefined);

export const LanguageProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [language, setLanguageState] = useState<Language>("es");
  const originals = useRef(new WeakMap<Text, string>());
  const originalAttributes = useRef(new WeakMap<Element, Map<string, string>>());

  useEffect(() => {
    const saved = localStorage.getItem("airres-lang") as Language;
    if (saved && (saved === "es" || saved === "en")) {
      setLanguageState(saved);
    }
  }, []);

  useEffect(() => {
    const visit = (root: Node) => {
      const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT | NodeFilter.SHOW_ELEMENT);
      const apply = (node: Node) => {
        if (node.nodeType === Node.TEXT_NODE) {
          const textNode = node as Text;
          const parent = textNode.parentElement;
          if (!parent || ["SCRIPT", "STYLE", "TEXTAREA"].includes(parent.tagName)) return;
          const original = originals.current.get(textNode) ?? textNode.nodeValue ?? "";
          if (!originals.current.has(textNode)) {
            if (translateUiText(original) === original) return;
            originals.current.set(textNode, original);
          }
          const next = language === "en" ? translateUiText(original) : original;
          if (textNode.nodeValue !== next) textNode.nodeValue = next;
        } else if (node instanceof Element) {
          let saved = originalAttributes.current.get(node);
          if (!saved) { saved = new Map(); originalAttributes.current.set(node, saved); }
          for (const attribute of ["placeholder", "title", "aria-label", "alt"]) {
            const value = node.getAttribute(attribute);
            if (value === null) continue;
            if (!saved.has(attribute)) saved.set(attribute, value);
            const original = saved.get(attribute)!;
            const next = language === "en" ? translateUiText(original) : original;
            if (value !== next) node.setAttribute(attribute, next);
          }
        }
      };
      apply(root);
      while (walker.nextNode()) apply(walker.currentNode);
    };
    visit(document.body);
    const observer = new MutationObserver((records) => {
      for (const record of records) {
        if (record.type === "characterData") {
          const node = record.target as Text;
          const current = node.nodeValue ?? "";
          const previous = originals.current.get(node);
          const displayed = previous === undefined ? undefined : language === "en" ? translateUiText(previous) : previous;
          if (current !== displayed) {
            if (translateUiText(current) === current) originals.current.delete(node);
            else originals.current.set(node, current);
          }
          visit(node);
        } else {
          record.addedNodes.forEach(visit);
        }
      }
    });
    observer.observe(document.body, { childList: true, characterData: true, subtree: true });
    return () => observer.disconnect();
  }, [language]);

  const setLanguage = (lang: Language) => {
    setLanguageState(lang);
    localStorage.setItem("airres-lang", lang);
  };

  const t = (key: string): string => {
    const keys = key.split(".");
    let value: any = translations[language];
    for (const k of keys) {
      if (value === undefined) break;
      value = value[k];
    }
    return value || key;
  };

  return (
    <LanguageContext.Provider value={{ language, setLanguage, t }}>
      {children}
    </LanguageContext.Provider>
  );
};

export const useLanguage = () => {
  const context = useContext(LanguageContext);
  if (!context) {
    throw new Error("useLanguage must be used within a LanguageProvider");
  }
  return context;
};
