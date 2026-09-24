export function formatFlightLocalTime(epochSeconds: number, timeZone: string, language: "es" | "en" = "es") {
  const zone = timeZone || "UTC";
  try {
    return new Intl.DateTimeFormat(language === "es" ? "es-BO" : "en-US", {
      timeZone: zone,
      year: "numeric", month: "short", day: "2-digit", hour: "2-digit", minute: "2-digit",
      hourCycle: "h23", timeZoneName: "short",
    }).format(new Date(epochSeconds * 1000));
  } catch {
    return new Intl.DateTimeFormat("es-BO", { timeZone: "UTC", year: "numeric", month: "short", day: "2-digit", hour: "2-digit", minute: "2-digit", hourCycle: "h23", timeZoneName: "short" }).format(new Date(epochSeconds * 1000));
  }
}
