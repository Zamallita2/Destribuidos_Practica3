# Aerolíneas Pabón: implementación y operación

## Importación del CSV

`backend/data/matrices.json` define las rutas dirigidas. Se importa una fila si sus dos aeropuertos están en `airports`, la duración dirigida es positiva y existe una tarifa positiva en **Primera o Turista**. Una tarifa nula deja indisponible esa clase, pero no invalida la otra. El sentido inverso se evalúa por separado. No se filtra por continuidad física del avión.

Al arrancar, la aplicación escribe `reports/vuelos_rechazados.csv` con las columnas originales del CSV y `motivo_rechazo`. Solo aparecen rechazos por aeropuerto fuera del catálogo, origen igual al destino o falta de ruta en la matriz. Otros errores de formato se registran por separado. El archivo es UTF-8 y puede abrirse en Excel.

Los vuelos históricos pertenecen al PostgreSQL de la región de origen. Los identificadores de América son menores que 1.000.000.000 y los de Europa/Asia son mayores o iguales. El otro PostgreSQL conserva una copia de recuperación; MongoDB mantiene la proyección central.

## Ocupación y pasajeros

Cada vuelo tiene un manifiesto propio en `ocupaciones_vuelo`. Incluye un asiento y un pasajero simulado para cada plaza inicialmente vendida (`SALED`) o reservada (`RESERVED`). La lista de nombres incluye caracteres latinos, chinos, japoneses y árabes. Un vuelo sin tarifa para una clase muestra esos asientos como `BLOCKED`; solo las plazas comercializables forman la base del 73 % y el 3 %.

Los asientos son indivisibles: se usa `round(plazas_comercializables × 0,73)` y `round(plazas_comercializables × 0,03)`. Por ejemplo, 228 plazas producen 166 vendidas y 7 reservadas. El manifiesto se genera de forma determinista y se guarda en PostgreSQL, se replica al otro PostgreSQL y se proyecta a MongoDB. Una tarea de fondo llena los manifiestos faltantes; consultar asientos o reservar genera el manifiesto de ese vuelo de inmediato. `GET /api/dashboard` informa cuántos manifiestos se han generado.

Los boletos posteriores se guardan como registros separados. La reserva usa una transacción, bloqueo del vuelo y un índice único sobre `(id_vuelo, id_asiento)` para estados activos. Un asiento ocupado en el manifiesto tampoco puede comprarse de nuevo. Una cancelación pasa por `REFUNDED` y queda disponible al vencer `REFUND_DELAY_MINUTES` (15 por defecto).

## Sincronización y fallos

Cada compra o modificación crea un evento en `sync_outbox` dentro de la misma transacción. El publicador reintenta los eventos hasta que ambos PostgreSQL y MongoDB los reciban. Los relojes vectoriales conservan causalidad y Lamport más identificador de nodo resuelve eventos concurrentes. Al reiniciar se restaura el estado de los relojes desde el outbox. La reconciliación periódica copia registros faltantes tras la recuperación de un PostgreSQL.

Si cae uno de los PostgreSQL, las lecturas y escrituras usan la copia del otro. Si cae MongoDB, las consultas de Asia recurren al PostgreSQL disponible. Una compra se confirma con HTTP 200 después de que exista una segunda copia; si solo quedó guardada localmente, responde HTTP 202 y muestra `replication_pending`. El `GET /api/health` publica el estado de los tres nodos. El asignador transaccional `id_allocators` evita consumir identificadores de dominio por transacciones revertidas, algo que `nextval()` no garantiza.

Este diseño cubre la caída individual y posterior recuperación de un servicio. La garantía de evitar reservas dobles bajo una **partición de red con dos escritores activos e incomunicados** requeriría un mecanismo de consenso/fencing o un servicio de reservas con quórum; no debe presentarse como resuelto por relojes lógicos. El equipo debe probar exactamente ese límite si la rúbrica incluye particiones, además de detener procesos.

## Zonas horarias

Los tiempos se guardan como Unix UTC. `data/airports.json` indica la zona IANA de cada aeropuerto. El navegador muestra la salida en la zona del origen y la llegada en la zona del destino, independientemente de la capital de compra seleccionada. El boleto guarda `time_zone_compra` solo como contexto del comprador; no altera la hora del vuelo. El backend valida la zona IANA y el contenedor incluye `tzdata`.

## Pases y Wallet

Una compra confirmada genera un pase con QR en `GET /api/boletos/{id}/pase`. El QR lleva una URL firmada con HMAC y el escáner puede comprobar su vigencia mediante `GET /api/boletos/{id}/validar?token=...`. El boleto PDF usa el mismo QR verificable. `BOARDING_PASS_SECRET` debe ser estable y secreto. Para una demostración desde un teléfono, abrir el frontend con la IP de la computadora y configurar `PUBLIC_BASE_URL` con una URL que el teléfono pueda abrir. Cuando el valor sigue apuntando a `localhost`, el backend intenta usar el host recibido por el proxy del frontend. El frontend dirige sus llamadas `/api` al backend por el proxy de Next.js, de modo que el navegador móvil no intenta conectar con su propio `localhost`.

**Modo de demostración sin certificados:** el botón «Añadir a mi Billetera / Descargar Pase» está disponible después de una compra `SALED`, y también en Gestión de Boletos. Descarga `GET /api/boletos/{id}/wallet/demo.pkpass` como `application/vnd.apple.pkpass`. El archivo ZIP incluye `pass.json` con estilo `boardingPass`, nombre, asiento, ruta, puerta, horarios locales y código QR, además de iconos PNG y `manifest.json` con hashes. Omite intencionalmente `signature`. El usuario puede abrir el archivo desde Descargas o Archivos y elegir una billetera de terceros que admita pases sin firmar. El pase se muestra sin conexión una vez importado; la comprobación de vigencia del QR necesita conexión. La asociación automática de la descarga con una app depende del sistema y la app instalada. **Apple Wallet oficial rechazará el archivo sin firma**, y no se promete que todas las billeteras de terceros lo acepten. Se recomienda ensayar con la app y el teléfono concretos antes de la presentación.

Los endpoints oficiales siguen siendo opcionales y no son necesarios para la demostración. El botón de Apple oficial solo aparece cuando se configuran `APPLE_PASS_TYPE_ID`, `APPLE_TEAM_ID`, `APPLE_PASS_CERT_PATH`, `APPLE_PASS_KEY_PATH` y `APPLE_WWDR_CERT_PATH`; el endpoint construye y firma otro `.pkpass`. El botón de Google oficial requiere `GOOGLE_WALLET_ISSUER_ID` y `GOOGLE_WALLET_CREDENTIALS_PATH` (JSON de la cuenta de servicio autorizada); genera un enlace con JWT RSA firmado. Si alguna vez se usan, colocar las claves en `wallet-secrets/` (excluido de Git) y dar sus rutas **dentro del contenedor**. Estos flujos oficiales no se han ensayado con dispositivos reales.

## Sustituir datos en el futuro

1. Reemplazar el CSV en `dataset/` y `backend/data/matrices.json`, conservando las claves `airports`, `travel_time`, `economy_fares` y `first_class_fares`.
2. Si aparecen aeropuertos nuevos, añadir su país, región y zona IANA a `backend/data/airports.json`. No es posible deducir con seguridad la zona horaria o región solo del código en la matriz.
3. Reiniciar el backend con `docker compose up -d --build backend`. Las matrices se actualizan en ambas bases y Mongo; el importador agrega filas nuevas sin borrar vuelos que puedan tener boletos. Los manifiestos se regeneran al detectar un cambio de matriz y conservan las compras activas.
4. Revisar `reports/vuelos_rechazados.csv` y `GET /api/diagnostico/continuidad`. El diagnóstico informa saltos físicos, pero no descarta vuelos.
5. Si la intención es **reemplazar** por completo un dataset anterior, respaldar los boletos y realizar una migración explícita de vuelos históricos. La importación habitual es aditiva para no borrar reservas reales.

## Demostración aislada

`docker compose -f docker-compose.integration.yml -p aeropabon_codex_test up -d --build` usa volúmenes independientes de la instalación normal: frontend en `http://localhost:3002` y API en `http://localhost:8081`. Para probar desde un teléfono conectado a la misma red, abrir `http://IP_DE_LA_COMPUTADORA:3002`; el proxy del frontend lleva las llamadas `/api` al backend. En ese entorno `SEED_OCCUPANCY_MAX_FLIGHTS=5` limita el llenado de fondo para pruebas rápidas; las consultas generan el manifiesto del vuelo solicitado. La instalación normal no tiene ese límite.

Para comprobar recuperación: comprar un asiento y verificarlo en los tres nodos; detener MongoDB, comprar otro, reactivar MongoDB y comprobar la proyección; repetir deteniendo cada PostgreSQL y comprando por su región. Verificar además que dos compras concurrentes del mismo asiento produzcan un éxito y un conflicto, que un reembolso respete el plazo y que los identificadores no se reutilicen tras un rollback.
