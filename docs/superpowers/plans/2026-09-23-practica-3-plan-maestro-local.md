# Plan maestro de implementación — Práctica 3 Aerolíneas Pabón

**Estado:** propuesta para revisión; no autoriza implementación hasta que el usuario confirme el diseño.

**Objetivo:** cumplir la consigna de sincronización distribuida y reservas, conservando el CSV completo como evidencia y publicando únicamente vuelos físicamente posibles y reservables.

**Fuentes revisadas:** consigna PDF de cinco páginas, `dataset/02 - Practica 3 Dataset Flights.csv`, `backend/data/matrices.json`, servicios Go, inicializadores SQL/Mongo y páginas Next.js actuales.

## 1. Decisiones de producto y restricciones

1. La matriz es un grafo **dirigido**: `A → B` y `B → A` son decisiones independientes. Una tarifa `null` significa que esa clase no se vende en ese sentido. Una ruta es importable si tiene tarifa no nula en **al menos una** clase y una duración positiva. La API ofrecerá solo las clases con tarifa válida.
2. El CSV crudo será inmutable y auditable. Las filas se clasificarán como `RUTA_INVALIDA`, `RUTA_VALIDA_NO_PROGRAMADA` o `VUELO_OPERATIVO`. No se presentarán las filas no programadas como vuelos disponibles.
3. Habrá exactamente 50 aeronaves físicas, con los modelos y capacidades de la consigna. Un avión no tendrá dos vuelos solapados y su siguiente salida deberá coincidir con su ubicación anterior o con el destino de un reposicionamiento factible.
4. Los vuelos de reposicionamiento serán eventos operativos generados, sin venta de pasajes, siempre sobre una dirección permitida y con tiempo suficiente. No se crearán para forzar la aceptación de todo el CSV.
5. Los timestamps persistidos serán UTC. El CSV carece de zona horaria explícita: para reproducir la importación se interpretará como UTC y se documentará esta suposición; los vuelos nuevos exigirán fecha/hora con zona o UTC explícito. El frontend mostrará hora local y zona.
6. El CSV fechado entre 25/03/2026 y 05/04/2026 quedará como histórico. La demostración de reservas futuras utilizará una programación sintética separada, trazable a rutas válidas y generada con los mismos límites de flota.
7. Cada vuelo tendrá un único **nodo propietario de escrituras** para reservas y estados. PostgreSQL América será propietario de vuelos con origen americano; PostgreSQL Europa/Asia, de los demás. MongoDB será el lago/proyección de lectura. La elección de propietario depende del origen del vuelo, no del país del usuario ni de un header manipulable.
8. Una partición de red no habilitará ventas del mismo asiento en varios nodos. El nodo propietario podrá seguir vendiendo localmente; los demás tendrán lectura potencialmente atrasada o responderán que la escritura no está disponible. La consistencia fuerte de reservas y la consistencia eventual de proyecciones se documentarán por separado.
9. Los relojes Lamport/vectoriales demostrarán orden causal y detección de concurrencia; la unicidad de reservas se impondrá con transacciones y restricciones de base de datos. Para eventos concurrentes no transaccionales habrá desempate determinista y registro de conflicto. No se afirmará orden total de entrega global sin un protocolo que lo garantice.
10. El TSP de la consigna se resolverá sobre la red dirigida de rutas y tarifas; la búsqueda de vuelos reservables será un problema temporal distinto. Ambos resultados se etiquetarán claramente.

## 2. Línea base y límites observados

- CSV: 60.000 filas; las 60.000 usan alguno de los 15 aeropuertos de la matriz; 24.789 tienen ambas tarifas y 29.377 tienen por lo menos una. Estas cifras se recalcularán en una prueba de importación reproducible.
- Los 24.789 registros con ambas tarifas suman cerca de 200.770 horas de vuelo en 12 días. La flota solo dispone de 14.400 horas teóricas en ese intervalo. El objetivo del plan es maximizar el subconjunto operativo, no afirmar que los 29.377 sean programables.
- `backend/data/seed_flights.go` usa llegada `salida + 1 h`, requiere ambas tarifas y desactiva la continuidad.
- `backend/main.go` importa el CSV completo en ambos PostgreSQL y no replica inicialmente esos vuelos a Mongo.
- `backend/services/sync_service.go` descarta eventos cuando se llena el canal y retrasa todo el consumidor durante una devolución.
- `backend/handlers/reserva_handler.go` protege la transacción PostgreSQL local, pero la rama Mongo consulta e inserta sin operación atómica. La ocupación simulada por avión y la disponibilidad por vuelo se mezclan.
- `backend/services/tsp_service.go` calcula un camino hamiltoniano abierto usando matrices; para criterio `TIME` puede aceptar una arista cuyo precio para la clase elegida sea `null`.
- El dashboard consulta solamente 100 vuelos en la API y calcula métricas a partir de esa muestra; la exportación PDF existe, pero hay contenido fijo en el diseño del boleto que debe contrastarse con el boleto real.

## 3. Entregas separadas y orden de implementación

Cada entrega termina con una aplicación ejecutable y pruebas propias. Aprobar este plan maestro permitirá redactar tareas técnicas pequeñas antes de implementar cada entrega.

### Entrega A — Catálogo de rutas e importación auditable

**Áreas:** `backend/data/seed_flights.go`, `backend/data/matrices.json`, modelos Go, esquema SQL y colección de importación en Mongo.

1. Extraer un servicio de catálogo que valide aeropuertos, duración y tarifas por sentido y clase; conservar `null` como ausencia, sin convertirlo en cero.
2. Reemplazar el filtro `eco == nil || first == nil` por `eco == nil && first == nil`; guardar `economy_available` y `first_available` por ruta/vuelo.
3. Parsear fecha y hora con errores por fila; calcular llegada con `travel_time`, y guardar línea CSV, `aircraft_id_original`, estado original y versión/hash de matriz.
4. Hacer la importación repetible mediante una clave estable del archivo y número de línea. Registrar los conteos y motivos de exclusión sin borrar la fuente.
5. Separar registros de importación de la tabla de vuelos operativos. Una ruta válida aún no implica un avión asignado.

**Aceptación:** prueba con 60.000 filas que reproduce el desglose 60.000/29.377 y distingue rutas con una sola clase; prueba dirigida donde `A → B` existe y `B → A` no; importación repetida sin duplicados; llegadas según matriz, incluso al día siguiente.

### Entrega B — Programación física de flota

**Áreas:** nuevo paquete `backend/scheduling`, modelo de vuelo, migraciones PostgreSQL, importador y comandos de generación de datos de demostración.

1. Definir estado operativo por aeronave: modelo, aeropuerto disponible y UTC de disponibilidad. El primer vuelo fija una posición inicial documentada.
2. Construir candidatos de conexión entre vuelos sin solapamiento, con tiempo de escala configurable y origen coincidente. Excluir cancelados del encadenamiento; conservarlos en el histórico.
3. Asignar vuelos a un máximo de 50 cadenas. Empezar con asignación determinista; comparar su cantidad aceptada con una optimización de flujo/cobertura de caminos y elegir la implementación que aporte una mejora medible con coste razonable.
4. Permitir reposicionamiento únicamente si existe una secuencia dirigida permitida que cabe en el hueco, y guardar la secuencia generada con coste/tiempo operacional. No vender esos tramos.
5. Generar un calendario futuro de demostración a partir de rutas válidas, con semilla fija y procedencia separada. Mantener inalteradas las fechas del CSV original.

**Aceptación:** validador independiente de todas las cadenas: continuidad espacial, ausencia de solapes, 50 IDs máximos, duraciones y escalas respetadas. Reporte de vuelos programados, rechazados por capacidad/tiempo y reposicionamientos. No se fija anticipadamente una cifra de vuelos aceptados.

### Entrega C — Propiedad de datos y sincronización recuperable

**Áreas:** `backend/db/router.go`, `backend/main.go`, `backend/services/{lamport,vector,sync}_service.go`, modelos/migraciones, `docker-compose.yml` e inicialización Mongo.

1. Definir IDs globales estables para vuelos, boletos, asientos y eventos; quitar la generación `CountDocuments()+1`.
2. Guardar cambios de dominio y eventos de salida en la misma transacción PostgreSQL del propietario mediante una tabla `outbox`. El publicador reintentará sin perder eventos tras reinicio; los receptores deduplicarán por `event_id`.
3. Hacer una carga inicial y reconciliación periódica hacia Mongo para vuelos, asientos, boletos, catálogos y matrices. Evitar que el arranque inserte todo el CSV en ambos propietarios.
4. Persistir `node_id`, Lamport y vector por evento; incrementar Lamport en emisión/recepción y vector por nodo real. Detectar versiones causales, obsoletas y concurrentes; registrar decisiones. Definir desempate estable `(lamport, node_id, event_id)` solo para entidades que admiten resolución automática.
5. Ejecutar instancias configuradas con identificadores de nodo distintos y documentar qué escrituras admite cada una. Añadir estado de sincronización: pendiente, aplicado, reintentando y último evento confirmado por destino.

**Aceptación:** pruebas de evento duplicado, fuera de orden y concurrente; corte de Mongo o de un PostgreSQL y recuperación sin pérdida; comparación automática de conteos y versiones tras reconexión. Mostrar en la demo que una reserva local confirmada sobrevive a la caída del proceso de sincronización.

### Entrega D — Reservas, compras, devoluciones y estados

**Áreas:** `backend/handlers/reserva_handler.go`, modelos/migraciones SQL, servicio de reservas, rutas API y pantallas de boletos.

1. Modelar ocupación por par `(vuelo, asiento)`; comprobar que el asiento pertenece al avión asignado y que su clase tiene precio para esa ruta. Calcular el precio en servidor, sin confiar en `costo` enviado por cliente.
2. Crear índice único parcial para una reserva/venta activa por `(vuelo, asiento)` y realizar el cambio en transacción del propietario. Definir estados `AVAILABLE`, `RESERVED`, `SALED`, `REFUNDED`, `ANNULLED` y transiciones válidas.
3. Cancelar con transición explícita a `REFUNDED`, guardar `available_at = cancellation_at + intervalo configurable`, y liberar con trabajador persistente en ese instante. Ningún `Sleep` bloqueará otros eventos.
4. Validar estados de vuelo y no permitir compra tras el inicio del abordaje o sobre vuelos pasados/cancelados. Usar claves de idempotencia para reintentos de compra/cancelación.
5. Sembrar el 73% vendido y 3% reservado de forma determinista para la programación de demostración, representando la ocupación por vuelo y clase; el resto estará disponible. No inventar datos personales de pasajeros.

**Aceptación:** dos solicitudes simultáneas al mismo asiento producen una sola venta; intento desde otro nodo no genera duplicado; cancelación sigue sin disponibilidad hasta vencer el intervalo; comprador no puede cambiar precio o clase; porcentajes configurables demostrables.

### Entrega E — Dijkstra y TSP correctos

**Áreas:** `backend/services/{sugerencias,tsp}_service.go`, handlers, API y `frontend/app/sugerencias/page.tsx`.

1. Construir una única representación de grafo dirigido a partir de la tarifa seleccionada: `null` elimina la arista, también al minimizar tiempo. Reutilizarla para Dijkstra/Yen y TSP.
2. Validar ciudades no repetidas, origen, clase, criterio y existencia de camino. Exponer explícitamente `camino_abierto` y `ciclo_con_retorno`; el retorno solo es posible si la arista final existe.
3. Sustituir la enumeración factorial para 15 ciudades por programación dinámica de subconjuntos o una alternativa exacta acotada, devolviendo coste, tiempo y aristas usadas.
4. Mostrar el resultado como **ruta óptima de red**. Para itinerarios reservables, añadir una consulta temporal separada que exija vuelos operativos, conexiones cronológicas y plazas disponibles; no presentar la ruta de matriz como compra garantizada.

**Aceptación:** ejemplos asimétricos y sin retorno, clase única, criterio tiempo con precio `null`, conjunto sin solución y 15 ciudades dentro de un tiempo de respuesta documentado. Comparación de resultados con fuerza bruta en instancias pequeñas.

### Entrega F — Interfaz, paneles y pasajes

**Áreas:** `frontend/app/{page,vuelos,boletos,gestion-boletos,sugerencias}/page.tsx`, traducciones, nuevos endpoints agregados.

1. Listar solo vuelos operativos reservables por defecto, con fecha, zona, clase disponible, tarifa y cupo por vuelo. Paginación/filtrado del lado servidor para no limitar métricas a 100 registros.
2. Crear panel global y vista de vuelo con vendidos, reservados, disponibles e ingresos separados por clase, calculados desde el propietario o una proyección con indicador de retraso.
3. Enviar clase/asiento/fecha válidos y mostrar conflictos de reserva y estado de sincronización sin ofrecer asientos ocupados.
4. Revisar el PDF para que nombre, ruta, fecha, clase, asiento, puerta y código de boleto provengan del registro real; generar ejemplos verificables y legibles. La compatibilidad Wallet requiere una decisión de formato/proveedor y, si corresponde, credenciales de firma; no se simulará un pase Wallet válido.
5. Completar textos en español e inglés y visualización de horas locales para usuarios globales.

**Aceptación:** recorrido completo de consulta → reserva → compra → PDF → cancelación; métricas coinciden con base de datos; un vuelo de una sola clase nunca ofrece la clase ausente; el PDF coincide con la reserva real; interfaz utilizable en dos idiomas.

### Entrega G — Tolerancia a fallos y evidencia académica

**Áreas:** pruebas de integración, scripts de demo, `docker-compose.yml`, documentación técnica y capturas/resultados.

1. Documentar topología, propietario por entidad, flujos de eventos, orden causal, desempates, consistencia de cada operación, límites ante particiones y procedimiento de recuperación.
2. Preparar escenario reproducible: nodos levantados, dos compradores del mismo asiento, corte de comunicación, operación local permitida, reconexión, convergencia, devolución diferida y rutas asimétricas.
3. Añadir verificadores de invariantes: ninguna doble venta, ninguna cadena de avión con salto/solape, coincidencia eventual de proyecciones, ninguna oferta de clase `null` y totales del dashboard.
4. Ejecutar `go test ./...`, pruebas de integración con tres bases, compilación Next.js y recorrido manual de la demo. Registrar comandos, datos de entrada, resultados y capturas en el entregable.

**Aceptación:** demostración repetible desde un entorno limpio; reinicio durante eventos pendientes sin pérdida; documentación y evidencias alineadas con cada punto del PDF.

## 4. Orden de prioridad

1. A y B: establecer qué vuelos existen y cuáles pueden operarse.
2. C y D: asegurar sincronización y venta sin sobreventa.
3. E: algoritmos de rutas correctos sobre la red dirigida.
4. F: interfaz, métricas y boleto comprobable.
5. G: pruebas de fallos, documentación y evidencia final.

## 5. Puntos para revisar antes de ejecutar

1. **Supuesto sobre la hora del CSV:** interpretar el valor como UTC porque no trae zona horaria; cualquier otra interpretación exige una fuente que indique la zona de cada registro.
2. **Propiedad de vuelos:** asignar por aeropuerto de origen a los dos PostgreSQL y usar Mongo como proyección, manteniendo tres bases conectadas.
3. **Programación futura:** generar datos de prueba marcados como sintéticos; no desplazar las fechas del CSV histórico sin etiquetarlo.
4. **Wallet:** PDF funcional queda dentro del alcance obligatorio. Un Wallet firmado será una entrega adicional si se dispone de proveedor y credenciales; el PDF de la consigna contiene una frase incompleta sobre cantidad/formato de boletos.

## 6. Riesgos y pruebas de regresión prioritarias

1. Un vuelo `A → B` con retorno `B → A` ausente no debe crear regreso en Dijkstra, TSP ni reposicionamiento.
2. Una ruta con tarifa solo Economy debe importarse y vender solo Economy; análogo para First.
3. Dos reservas concurrentes del mismo asiento desde interfaces distintas deben terminar en una sola venta, aun con reintentos y evento duplicado.
4. Un retraso de 15 minutos de una devolución no debe bloquear la propagación de otra reserva ni liberar el asiento anticipadamente.
5. El planificador debe rechazar un segundo vuelo cuyo origen, hora o duración contradigan la posición de cualquiera de los 50 aviones.
