# Verificación de la implementación

Pruebas realizadas con `docker-compose.integration.yml` en el proyecto aislado `aeropabon_codex_test`, sin modificar los volúmenes del Compose principal.

| Comprobación | Resultado observado |
| --- | --- |
| Importación del CSV actual | 29.377 filas admitidas por origen, destino, duración y al menos una tarifa; 30.623 filas en `reports/vuelos_rechazados.csv` |
| Vuelos de demostración | 50 vuelos futuros adicionales; 29.427 vuelos en cada PostgreSQL y en MongoDB |
| Ocupación de un A380 de 449 plazas comercializables | 328 `SALED`, 13 `RESERVED`, 108 `AVAILABLE`; coincide con redondeo de 73 % y 3 % |
| Compra Unicode | «张伟» persistió con ID 1 en ambos PostgreSQL y MongoDB; costo 1400 tomado de la matriz |
| Hora local | ATL: `2026-09-24T17:00:00-04:00`; TYO: `2026-09-25T22:00:00+09:00`; compra desde `America/Bogota` |
| QR | Imagen PNG servida con HTTP 200; URL firmada validada con `valid:true`; Apple y Google oficiales informan HTTP 503 cuando faltan credenciales |
| Pase `.pkpass` de demostración | HTTP 200 con `application/vnd.apple.pkpass` y descarga `AP-2-demo.pkpass`; ZIP con `pass.json`, `manifest.json` e iconos, sin `signature`; contiene pasajero árabe y `PKTransitTypeAir` |
| Descarga móvil simulada | Petición al proxy de Next.js con host `192.168.1.55:3002`: el QR del `.pkpass` apunta a ese host en la red local en lugar de `localhost` |
| Frontend Docker | Imagen construida y servicio activo en puerto 3002; `/boletos`, `/api/health` y descarga `.pkpass` respondieron HTTP 200 a través del proxy |
| Boleto anulado | El endpoint de descarga rechaza el boleto anulado con HTTP 410 |
| Caída de PostgreSQL América | Compra con nombre árabe continuó con HTTP 200 en la copia Europa/Asia y apareció en América al reiniciar |
| Caída de PostgreSQL Europa/Asia | Compra de vuelo europeo continuó con HTTP 200 en América y apareció en Europa/Asia al reiniciar |
| Caída de MongoDB | Compra continuó con HTTP 200 en PostgreSQL y apareció en MongoDB al reiniciar |
| Concurrencia | Dos compras simultáneas del mismo asiento respondieron HTTP 200 y HTTP 409 |
| Reembolso | `SALED` pasó a `REFUNDED` y luego a `ANNULLED` con demora de prueba de 0 minutos |
| Reinicio del backend | Permanecieron 29.427 vuelos y 5 boletos en cada PostgreSQL, sin duplicados ni pérdida |
| TSP dirigido | ATL, TYO y PEK produjo PEK → TYO → ATL, costo 1900 y 19 horas para Turista |
| Pruebas automatizadas | `go test ./...`, `npx tsc --noEmit` y `npm run build` completaron sin errores |

La prueba de integración limita la generación automática a cinco manifiestos al inicio; las consultas y compras generan el manifiesto del vuelo solicitado. El Compose principal no tiene ese límite. El pase `.pkpass` de demostración aún debe ensayarse en una billetera de terceros en un dispositivo real; no se han ensayado credenciales de emisor. La prueba de fallo cubre **una caída individual**, no una partición de red con escritores simultáneos.
