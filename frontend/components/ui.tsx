import type { ReactNode } from "react";
import type { LucideIcon } from "lucide-react";

type Icon = LucideIcon;

// Flight states follow the IDs stored in estados_vuelo.
export const flightStates: Record<number, { label: string; badge: string }> = {
  1: { label: "Programado", badge: "badge-blue" },
  2: { label: "Embarcando", badge: "badge-green" },
  3: { label: "Despegó", badge: "badge-sky" },
  4: { label: "En vuelo", badge: "badge-sky" },
  5: { label: "Aterrizó", badge: "badge-violet" },
  6: { label: "Llegó", badge: "badge-slate" },
  7: { label: "Cancelado", badge: "badge-red" },
  8: { label: "Retrasado", badge: "badge-amber" },
};

export function FlightStatusBadge({ state }: { state: number }) {
  const status = flightStates[state];
  return <span className={`badge ${status?.badge || "badge-slate"}`}>
    <span className="h-1.5 w-1.5 rounded-full bg-current" aria-hidden="true" />
    {status?.label || "Sin estado"}
  </span>;
}

const ticketStates: Record<string, { label: string; badge: string }> = {
  SALED: { label: "Vendido", badge: "badge-green" },
  RESERVED: { label: "Reservado", badge: "badge-amber" },
  REFUNDED: { label: "Reembolso en curso", badge: "badge-violet" },
  ANNULLED: { label: "Anulado", badge: "badge-red" },
};

export function TicketStatusBadge({ state }: { state: string }) {
  const status = ticketStates[state];
  return <span className={`badge ${status?.badge || "badge-slate"}`}>{status?.label || state}</span>;
}

export function PageHeader({ icon: IconComponent, eyebrow, title, subtitle, actions }: {
  icon?: Icon; eyebrow?: string; title: string; subtitle?: ReactNode; actions?: ReactNode;
}) {
  return <div className="mb-6 flex flex-wrap items-end justify-between gap-4">
    <div className="flex min-w-0 items-start gap-4">
      {IconComponent && <div className="hidden h-12 w-12 shrink-0 items-center justify-center rounded-2xl bg-navy-900 text-gold-300 shadow-lift sm:flex">
        <IconComponent className="h-6 w-6" aria-hidden={true} />
      </div>}
      <div className="min-w-0">
        {eyebrow && <p className="eyebrow">{eyebrow}</p>}
        <h1 className="text-2xl font-bold text-navy-900 sm:text-3xl">{title}</h1>
        {subtitle && <p className="mt-1.5 max-w-3xl text-sm text-slate-500 sm:text-base">{subtitle}</p>}
      </div>
    </div>
    {actions && <div className="flex flex-wrap items-center gap-2">{actions}</div>}
  </div>;
}

export function KpiCard({ icon: IconComponent, label, value, note, tone = "navy" }: {
  icon?: Icon; label: string; value: ReactNode; note?: string; tone?: "navy" | "gold" | "green" | "amber";
}) {
  const tones = {
    navy: "bg-navy-50 text-navy-700",
    gold: "bg-gold-50 text-gold-600",
    green: "bg-emerald-50 text-emerald-700",
    amber: "bg-amber-50 text-amber-700",
  };
  return <div className="card card-body">
    <div className="flex items-start justify-between gap-3">
      <p className="text-sm font-medium text-slate-500">{label}</p>
      {IconComponent && <span className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-xl ${tones[tone]}`}>
        <IconComponent className="h-5 w-5" aria-hidden={true} />
      </span>}
    </div>
    <p className="mt-2 text-2xl font-bold tracking-tight text-navy-900 sm:text-3xl">{value}</p>
    {note && <p className="mt-1 text-xs text-slate-500">{note}</p>}
  </div>;
}

export function EmptyState({ icon: IconComponent, title, children }: { icon?: Icon; title: string; children?: ReactNode }) {
  return <div className="flex flex-col items-center justify-center rounded-2xl border-2 border-dashed border-slate-200 bg-slate-50/60 px-6 py-12 text-center">
    {IconComponent && <span className="mb-4 flex h-14 w-14 items-center justify-center rounded-2xl bg-white text-navy-400 shadow-card">
      <IconComponent className="h-7 w-7" aria-hidden={true} />
    </span>}
    <h3 className="text-base font-bold text-navy-900">{title}</h3>
    {children && <div className="mt-1.5 max-w-md text-sm text-slate-500">{children}</div>}
  </div>;
}

export function RouteCodes({ from, to, size = "md" }: { from?: string; to?: string; size?: "md" | "lg" }) {
  const text = size === "lg" ? "text-3xl" : "text-lg";
  return <span className={`inline-flex items-center gap-2 font-bold tracking-tight text-navy-900 ${text}`}>
    {from || "?"}
    <svg viewBox="0 0 24 24" className={size === "lg" ? "h-6 w-6 text-gold-500" : "h-4 w-4 text-gold-500"} fill="currentColor" aria-hidden="true">
      <path d="M21 16v-2l-8-5V3.5a1.5 1.5 0 0 0-3 0V9l-8 5v2l8-2.5V19l-2 1.5V22l3.5-1 3.5 1v-1.5L13 19v-5.5z" transform="rotate(90 12 12)" />
    </svg>
    {to || "?"}
  </span>;
}
