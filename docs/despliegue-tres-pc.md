# Demostración distribuida con tres PCs

Esta modalidad ejecuta **una base de datos y una copia de la web/API en cada PC**. Las tres API se conectan por LAN a PostgreSQL América, PostgreSQL Europa/Asia y MongoDB. Si una PC se apaga, abre la web de una de las otras dos. La ruta de lectura regional y la cola persistente de sincronización se ven en `/sincronizacion`.

| Equipo | Rol del script | Base de datos local | Dirección para abrir la web |
| --- | --- | --- | --- |
| PC principal | `mongo` | MongoDB, puerto 27017 | `http://IP_MONGO:3001` |
| PC América | `america` | PostgreSQL América, puerto 5432 | `http://IP_AM:3001` |
| PC Europa/Asia | `europa` | PostgreSQL Europa/Asia, puerto 5435 | `http://IP_EU:3001` |

Cada PC necesita Windows PowerShell, Docker Desktop con contenedores Linux, Git, una copia **de la misma versión del proyecto**, y estar conectada a la misma LAN. Mantén Docker Desktop y las PCs activos durante la prueba. Reserva direcciones IP en el router si es posible; si cambian, edita `.env.distributed` en las tres PCs y reinicia los contenedores.

## 1. Comprobar que el Wi‑Fi permite conexiones entre equipos

En **cada PC**, ejecuta `ipconfig` en PowerShell y apunta la IPv4 del adaptador Wi‑Fi. Comprueba que las tres PCs pueden comunicarse entre sí, no solo salir a Internet. Algunas redes universitarias aíslan los clientes. Si `Test-NetConnection` hacia un puerto abierto de otra PC falla aun con la regla de firewall correcta, usa una red local/hotspot que permita tráfico entre clientes o pide al administrador que habilite la comunicación. Estar en el mismo SSID no garantiza conectividad LAN.

Antes del despliegue, cierra el modo de una PC en los equipos donde se haya ejecutado, para liberar los puertos:

```powershell
docker compose down
```

Esto conserva los volúmenes y datos del modo local. El despliegue distribuido usa volúmenes nuevos en cada PC, por lo que empieza con sus propias bases de datos.

## 2. Preparar el mismo archivo de configuración en las tres PCs

En la raíz del proyecto de **cada equipo**:

```powershell
Copy-Item .env.distributed.example .env.distributed
notepad .env.distributed
```

Reemplaza `PC_MONGO_IP`, `PC_AM_IP` y `PC_EU_IP` por las IPv4 reales. Escribe los **mismos** secretos en los tres archivos: `POSTGRES_PASSWORD` y `MONGO_PASSWORD` deben tener 16 caracteres alfanuméricos como mínimo; `BOARDING_PASS_SECRET`, 32 caracteres alfanuméricos como mínimo. Usa valores diferentes para los tres secretos. No compartas este archivo por chat ni lo subas al repositorio (está excluido por Git). Si ya existen volúmenes de una prueba distribuida, cambiar las contraseñas en el archivo no cambia automáticamente las contraseñas guardadas en las bases; mantén las originales o realiza la rotación dentro de las bases.

En la red universitaria, los puertos de las bases deben aceptar conexiones **solo de las otras dos PCs**. Configura el firewall de Windows con las IP de los pares. Por ejemplo, en la PC América, abre PowerShell como administrador y adapta las direcciones:

```powershell
$pares = @('IP_MONGO', 'IP_EU')
New-NetFirewallRule -DisplayName 'AirRes PG America LAN' -Direction Inbound -Action Allow -Protocol TCP -LocalPort 5432 -RemoteAddress $pares -Profile Any
New-NetFirewallRule -DisplayName 'AirRes web LAN' -Direction Inbound -Action Allow -Protocol TCP -LocalPort 3001 -RemoteAddress $pares -Profile Any
```

En la PC Europa/Asia usa el puerto `5435` y como pares las IP de Mongo y América. En la PC principal usa `27017` y como pares las IP de América y Europa/Asia. La regla web `3001` permite abrir la web desde las otras dos PCs; agrega la IP del celular a `-RemoteAddress` si necesitas escanear el QR desde allí. **No abras las bases de datos a toda la red universitaria.** La API escucha dentro de Docker y se accede por el proxy de la web; no hace falta publicar el puerto 8080.

## 3. Arrancar primero las tres bases de datos

Ejecuta cada línea en la PC indicada, desde la raíz del proyecto:

```powershell
# PC America
.\scripts\start-distributed.ps1 -Role america -Stage database

# PC Europa/Asia
.\scripts\start-distributed.ps1 -Role europa -Stage database

# PC principal, MongoDB
.\scripts\start-distributed.ps1 -Role mongo -Stage database
```

Puedes ejecutar las tres líneas en paralelo, cada una en su PC. Docker puede tardar en descargar imágenes e inicializar la base. Desde la PC principal, confirma los puertos:

```powershell
Test-NetConnection IP_AM -Port 5432
Test-NetConnection IP_EU -Port 5435
Test-NetConnection IP_MONGO -Port 27017
```

Los tres resultados deben tener `TcpTestSucceeded : True`. Si alguno falla, revisa IP, Docker, contenedor, reglas de firewall y aislamiento del Wi‑Fi antes de seguir.

## 4. Arrancar las tres aplicaciones

Primero en la **PC principal**:

```powershell
.\scripts\start-distributed.ps1 -Role mongo -Stage app
```

Este arranque carga el dataset y las matrices una sola vez, escribe las dos PostgreSQL y crea el snapshot en Mongo. Comprueba que `http://IP_MONGO:3001/api/health` responda y espera a que `/sincronizacion` muestre los tres nodos `UP` y la cola pendiente baje a cero. Un dataset grande puede tardar varios minutos. También puedes seguir los logs:

```powershell
docker compose --env-file .env.distributed -f docker-compose.distributed.yml -p airres_distributed --profile mongo logs -f backend
```

Después arranca las copias de la aplicación en los otros equipos:

```powershell
# PC America
.\scripts\start-distributed.ps1 -Role america -Stage app

# PC Europa/Asia
.\scripts\start-distributed.ps1 -Role europa -Stage app
```

Abre desde cada equipo `http://SU_IP:3001/sincronizacion`. El panel de cada copia consulta los mismos tres servidores, aunque su timeline de eventos se mantiene en memoria **por API**, de modo que al cambiar de web puede comenzar otra secuencia de eventos. Comprueba un vuelo con origen americano y otro con origen europeo/asiático: las lecturas habituales deben indicar su PostgreSQL regional. Mongo es el snapshot global y cubre lecturas cuando falta la fuente regional; no sustituye una PostgreSQL para reservar asientos.

## 5. Prueba real de caída y recuperación

Haz una prueba por vez, esperando a que se vacíe la cola antes de la siguiente:

1. Con los tres nodos `UP`, crea un vuelo o boleto de prueba y anota su ID en `/sincronizacion` o en la pantalla correspondiente. Espera a que la cola pendiente llegue a cero. Confirma el registro desde la web de otra PC.
2. **Apaga físicamente la PC América** (o apaga Docker Desktop en ella). Mantén el navegador en la PC Mongo o Europa/Asia. En hasta unos segundos, el panel debe marcar `PG_AM DOWN`. Las lecturas americanas pasan al snapshot Mongo si está disponible. La PC Europa/Asia conserva su PostgreSQL y puede atender escrituras de vuelos que ya estén replicados allí. Crea una operación de prueba sobre un vuelo existente y observa la cola pendiente: permanece hasta que América regrese.
3. Enciende América, arranca Docker y ejecuta `-Role america -Stage all` si los contenedores no vuelven solos. Espera `PG_AM UP`, evento de recuperación/reconciliación y cola pendiente en cero. Comprueba el vuelo/boleto de prueba desde ambas webs.
4. Repite con **PC Europa/Asia** apagada. Sigue usando la web de Mongo o América; Europa/Asia debe figurar `DOWN`. Las lecturas de esa región usan Mongo cuando responde y las escrituras sobre registros ya replicados pueden hacerse en América. Restaura y espera la reconciliación.
5. Repite con la **PC principal de Mongo** apagada. Abre `http://IP_AM:3001/sincronizacion` o `http://IP_EU:3001/sincronizacion`: ambas webs y sus API siguen disponibles. Los vuelos se leen desde sus PostgreSQL; las operaciones nuevas quedan pendientes de proyectarse a Mongo. Restaura la PC principal y espera `Mongo UP` y la cola pendiente en cero.

Al probar una caída, cambia manualmente a la URL de una PC superviviente. El proyecto no tiene una IP virtual o balanceador que cambie la URL automáticamente. Un QR generado con la IP de una PC apagada tampoco redirige solo a otra; genera/abre el pase desde la web de una PC activa. Si se apagan **ambas** PostgreSQL, Mongo permite consultar el snapshot, pero las reservas y cambios necesitan una PostgreSQL disponible.

La divergencia del panel compara conteos globales; que dos conteos sean iguales no demuestra que cada fila sea idéntica. Para la demostración, verifica también el **mismo ID y su estado** en cada PC después de la reconciliación.

## 6. Seguir desarrollando con una sola PC

El archivo `docker-compose.yml` no cambia. En una sola computadora, usa el flujo habitual:

```powershell
.\scripts\start-project.ps1
```

Abre `http://localhost:3001`. Antes de cambiar de modalidad en la misma PC, ejecuta `docker compose down` para el modo local o `.\scripts\stop-distributed.ps1 -Role ROL` para el modo distribuido (reemplaza `ROL` por `mongo`, `america` o `europa`). `down` sin `-v` conserva los datos. Las bases de ambos modos se guardan por separado.

## Límites operativos de esta versión

- La importación/reemplazo de matrices y dataset desde Ajustes debe hacerse con **una sola API activa**, preferentemente la principal. En las otras dos PCs ejecuta `.\scripts\stop-distributed.ps1 -Role america -AppOnly` y `.\scripts\stop-distributed.ps1 -Role europa -AppOnly` respectivamente. Al terminar, copia `backend/data/matrices.json`, `backend/data/input-info.json` y el CSV activo de `dataset/` desde la principal a las otras dos copias del proyecto; luego reinicia sus API con `start-distributed.ps1 -Role ROL -Stage app`. El control de importación en curso aún es local a cada proceso; no coordina tres importaciones simultáneas ni copia archivos entre PCs.
- La replicación es eventual. Durante la caída, una escritura de respaldo depende de que el vuelo y sus datos auxiliares ya estén presentes en la PostgreSQL superviviente. Verifica `cola pendiente = 0` y el registro concreto antes de apagar otra PC.
- Las tres API acceden a las mismas bases por LAN, así que el estado del panel representa conexiones reales. La red Wi‑Fi puede introducir latencia o bloquear puertos; un fallo de red se verá como un nodo caído aunque la PC siga encendida.
