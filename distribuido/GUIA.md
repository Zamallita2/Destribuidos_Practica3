# Guía: el sistema repartido en 3 PCs por Wi-Fi

Las tres PCs deben estar conectadas **a la misma red Wi-Fi**. Cada una guarda una de las tres bases de datos, que se sincronizan entre sí.

| PC | Quién | IP | Corre | Puertos |
|---|---|---|---|---|
| **PC 1 · América** | Zamallita | `172.20.10.15` | PostgreSQL América | 5432 |
| **PC 2 · Europa** | Carla | `172.20.10.3` | PostgreSQL Europa/Asia | 5435 |
| **PC 3 · App** | Jhojan | `172.20.10.5` | MongoDB + backend + página web | 27017, 8080, 3001 |

> Si alguien se reconecta al Wi-Fi, su IP puede cambiar. En ese caso, actualicen el archivo `distribuido/.env` en las 3 PCs.

## Paso 1 · Preparar (las 3 PCs)

1. Tener **Docker Desktop** abierto y en ejecución.
2. Tener el proyecto en la rama `Carla_Code`, con la carpeta `distribuido/` (esta guía y los archivos `docker-compose.pc*.yml`).
3. Crear el archivo de configuración a partir de la plantilla, que ya tiene las IPs:
   ```powershell
   cd distribuido
   copy .env.example .env        # en Mac: cp .env.example .env
   ```
4. **Windows:** permitir Docker Desktop en el **Firewall de Windows**, marcando **redes privadas y públicas**. Si no aparece el aviso: Panel de control → Firewall de Windows Defender → Permitir una aplicación → Docker Desktop.

## Paso 2 · Encender las bases (PC 1 y PC 2 primero)

**PC 1 · Zamallita:**
```powershell
docker compose -f docker-compose.pc1-america.yml up -d
```

**PC 2 · Carla:**
```powershell
docker compose -f docker-compose.pc2-europa.yml up -d
```

**Comprobarlo desde la PC 3 (Jhojan)** antes de seguir:
```bash
nc -zv 172.20.10.15 5432     # debe decir "succeeded"
nc -zv 172.20.10.3 5435      # debe decir "succeeded"
```
Desde Windows se comprueba con `Test-NetConnection 172.20.10.15 -Port 5432`.

## Paso 3 · Encender la aplicación (PC 3, al final)

**PC 3 · Jhojan**, cuando las dos bases respondan:
```bash
docker compose -f docker-compose.pc3-app.yml up -d --build
docker compose -f docker-compose.pc3-app.yml logs -f backend
```
Esperen a ver dos líneas `Successfully imported ... vuelos`: una para América y otra para Europa/Asia.

> **El orden importa:** el backend importa los vuelos al arrancar y solo lo hace en las bases que responden en ese momento. Si se encendió antes que las PCs 1 y 2, borren sus datos y vuelvan a arrancarlo: `docker compose -f docker-compose.pc3-app.yml down -v` y después `up -d`.

## Paso 4 · Usar el sistema

Desde **cualquiera** de las 3 PCs (o un celular conectado al mismo Wi-Fi):

- Página web: <http://172.20.10.5:3001>
- Estado de las 3 bases: <http://172.20.10.5:8080/api/health>. Debe decir `"status":"ok"`.

## Qué pide la consigna y cómo se muestra

| Requisito | Dónde verlo |
|---|---|
| 3 bases de datos sincronizadas, en 3 máquinas distintas | Pantalla **Sincronización**: las 3 en `UP` con los mismos conteos |
| Relojes de Lamport y vectoriales | Cada boleto guarda `lamport_clock` y `vector_clock`; la cola `sync_outbox` los usa para ordenar eventos |
| Prevención de doble reserva | Comprar el mismo asiento en dos navegadores: uno funciona y el otro dice *asiento ocupado* |
| Tolerancia a fallos | Ver la tabla siguiente |
| Consultar, reservar, comprar y cancelar boletos | Pantallas **Vuelos**, **Boletos** y **Gestión de boletos** |
| Rutas óptimas (Dijkstra) y agente viajero (TSP) | Pantallas **Sugerencias** y **Agente viajero** |
| 2 dashboards (general y por vuelo) | **Panel** y el panel de cada vuelo |
| Multiidioma | Selector de idioma |
| Pasajes en PDF y Wallet | Después de comprar, o desde **Gestión de boletos** |
| Devolución: liberar el asiento a los 15 minutos | Cancelar un boleto: queda `REFUNDED` y se libera al vencer el plazo |

## Demostración de fallas reales

| Qué hacer | Qué se observa |
|---|---|
| **Desconectar el Wi-Fi** de la PC 1 o la PC 2 | Esa base aparece **DOWN** en Sincronización. El sistema sigue consultando y vendiendo con las otras bases. |
| Volver a conectar el Wi-Fi | En uno o dos minutos la base vuelve a `UP` y recibe los datos que se perdió; los conteos vuelven a coincidir. |
| En la PC 1: `docker compose -f docker-compose.pc1-america.yml stop`, y luego `start` | Lo mismo, pero apagando solo la base. |

> **Para la entrevista.** Si una base queda aislada, el sistema sigue funcionando con la otra y sincroniza al reconectarse: prioriza la **disponibilidad** y ofrece **consistencia eventual** (teorema CAP).

## Problemas comunes

| Problema | Solución |
|---|---|
| `/api/health` muestra una base en `false` | Revisen que esa PC esté en el mismo Wi-Fi, que su IP no haya cambiado y que el firewall permita Docker. |
| El `ping` funciona pero `nc`/`Test-NetConnection` falla | Es el firewall de esa PC; permitan Docker Desktop en redes privadas y públicas. |
| Cambió la IP de alguien | Actualicen `distribuido/.env` en las 3 PCs y en la PC 3 ejecuten `docker compose -f docker-compose.pc3-app.yml up -d`. |
| Quieren empezar de cero | En cada PC: `docker compose -f docker-compose.pcX-....yml down -v`, y vuelvan a los pasos 2 y 3. |

## Apagar

```powershell
docker compose -f docker-compose.pcX-....yml down      # conserva los datos
docker compose -f docker-compose.pcX-....yml down -v   # borra también los datos
```
