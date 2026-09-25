"use client";

import { useEffect, useState } from "react";

type QRAddress = { url: string; ready: boolean };

export default function QRNetworkInfo() {
  const [address, setAddress] = useState<QRAddress | null>(null);

  useEffect(() => {
    let active = true;
    async function refresh() {
      try {
        const response = await fetch("/api/network/qr-address", { cache: "no-store" });
        if (!response.ok) return;
        const value: QRAddress = await response.json();
        if (active) setAddress(value);
      } catch { /* El QR sigue disponible mientras la API responde. */ }
    }
    refresh();
    const timer = window.setInterval(refresh, 10000);
    return () => { active = false; window.clearInterval(timer); };
  }, []);

  if (!address) return null;
  return <p className={`mt-2 text-xs ${address.ready ? "text-emerald-200" : "text-amber-200"}`}>
    {address.ready ? <>Dirección actual del QR: <a className="underline" href={address.url} target="_blank" rel="noreferrer">{address.url}</a>. Abre esta dirección en el celular para comprobar que la red permite la conexión.</> : <>No se detectó una dirección accesible desde el celular. Inicia el proyecto con <code>scripts/start-project.ps1</code> y genera el QR de nuevo.</>}
  </p>;
}
