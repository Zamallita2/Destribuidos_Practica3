# Demostración de caída real de nodos

Abre [Sincronización](http://localhost:3001/sincronizacion) con el proyecto iniciado. El panel consulta las conexiones reales cada cuatro segundos. No hay botón de simulación: los cambios aparecen cuando un servidor deja de responder.

Desde la carpeta del proyecto, detén **solo una** base a la vez:

```powershell
docker compose stop postgres_am
docker compose start postgres_am

docker compose stop postgres_eu
docker compose start postgres_eu

docker compose stop mongodb
docker compose start mongodb
```

Mientras un nodo está detenido, revisa su tarjeta DOWN, la hora de la última lectura y los conteos de los otros nodos. En otra pestaña, abre **Vuelos** o **Boletos** para comprobar que los datos siguen disponibles desde una réplica. Al iniciar el nodo, espera la tarjeta UP y el evento de reconciliación. Para mostrar una cola pendiente, realiza una operación válida mientras un destino está caído y observa cómo se entrega al recuperarse.

Los conteos se marcan **Sin lectura** cuando el nodo está caído; eso no significa cero registros. La diferencia de cantidades es una señal de divergencia, pero cero no demuestra igualdad de contenido ni de versión. El total de outbox pendiente puede ser parcial si la PostgreSQL que guarda eventos está caída. El historial de eventos se conserva en memoria mientras la API está encendida; los datos y la cola de sincronización permanecen en las bases.

MongoDB es una copia para lectura y sincronización; una reserva requiere PostgreSQL. El panel muestra un nodo disponible para consultar datos, pero cada pantalla puede usar su propia ruta de lectura.

## Orden de lectura por región

| País seleccionado | Primera lectura | Si falla el principal | Si también falla MongoDB |
| --- | --- | --- | --- |
| América | PostgreSQL América | MongoDB (copia global) | PostgreSQL Europa/Asia (réplica) |
| Europa o Asia | PostgreSQL Europa/Asia | MongoDB (copia global) | PostgreSQL América (réplica) |

El país seleccionado decide la fuente para las listas; no filtra los vuelos por país. Los vuelos nuevos se crean en la PostgreSQL de su **aeropuerto de origen**. Un boleto nuevo se registra primero según la **capital de compra** seleccionada: América en PG América y Europa/Asia en PG Europa/Asia; si falta esa réplica, se usa la otra PostgreSQL cuando contiene el vuelo. MongoDB no acepta reservas.
