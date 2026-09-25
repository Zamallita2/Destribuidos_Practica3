# Gestión de datos de entrada

El módulo **Datos de entrada**, situado encima de Ajustes, permite reemplazar el dataset y cualquiera de las tres matrices sin editar archivos a mano. Primero se seleccionan archivos y se pulsa **Validar y preparar**. La vista previa cuenta filas y rutas admitidas. **Procesar y reemplazar datos** inicia el trabajo y muestra su avance.

## Archivos admitidos

- Dataset: `.csv` o `.xlsx`, primera hoja si es Excel. Debe contener `flight_date, flight_time, origin, destination, aircraft_id, status, gate` en cualquier orden. También se aceptan `fecha, hora, origen, destino, id_avion, estado, puerta`. La fecha usa `MM/DD/YY` y la hora `HH:MM` (UTC), como el CSV original. Los IDs de avión deben existir en el catálogo.
- Las tres matrices juntas: `.json` con `airports`, `travel_time`, `economy_fares` y `first_class_fares`, como `backend/data/matrices.json`.
- Una matriz: `.json` con el objeto de filas y columnas, o `.csv`/`.xlsx` con destinos en la primera fila y orígenes en la primera columna. Las celdas vacías, `-` y `null` significan que la tarifa no está disponible; para tiempos se interpretan como cero. Los valores positivos habilitan la ruta o clase. Las tres matrices deben cubrir el mismo conjunto de aeropuertos.

Solo hay que subir los archivos modificados. Los demás se conservan. La vista previa rechaza formatos, columnas y matrices incompletas. El catálogo de aeropuertos necesita país, región y zona horaria para cada código en `backend/data/airports.json`; si un dataset nuevo introduce un aeropuerto desconocido, hay que completar ese catálogo antes de procesarlo.

## Reemplazo

El botón de procesamiento **borra todos los vuelos, boletos y reservas anteriores** y vuelve a importar desde el dataset y matrices resultantes. La vista previa y la confirmación muestran esta consecuencia. Los archivos nuevos se guardan en `dataset/02 - Practica 3 Dataset Flights.csv` y `backend/data/matrices.json`, montados como volúmenes de Docker, por lo que sobreviven al reinicio del contenedor. La importación usa transacciones PostgreSQL para que una falla antes de confirmar conserve los datos anteriores. Luego se actualizan matrices, vuelos de demostración y MongoDB. El progreso termina en 100 % cuando la copia de vuelos en MongoDB coincide con PostgreSQL. Los manifiestos de ocupación se generan en segundo plano y también se crean al consultar o reservar un vuelo.

El CSV de filas rechazadas se puede descargar desde el módulo después del procesamiento. Los motivos de rechazo de rutas quedan en `reports/vuelos_rechazados.csv`.
