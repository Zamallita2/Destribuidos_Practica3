# Guía: el sistema repartido en 3 PCs con Tailscale

Cada integrante corre una parte del sistema en su computadora. Tailscale une las tres PCs en una red privada, aunque estén en redes distintas (casa, universidad o datos del celular).

| PC | Corre | Puertos que usa |
|---|---|---|
| **PC 1 · América** | PostgreSQL América + Servidor América (principal) | 5432, 8091 |
| **PC 2 · Europa** | PostgreSQL Europa/Asia + Servidor Europa | 5435, 8092 |
| **PC 3 · Asia** | MongoDB + Servidor Asia + gateway + página web | 27017, 8093, 8080, 3001 |

## Resumen: quién hace qué

| Orden | Quién | Qué hace |
|---|---|---|
| 1 | Quien tiene el código más reciente | Sube la rama `Jhojan_Code` a GitHub (commit y push). Las otras PCs la necesitan para descargar este código. |
| 2 | Los 3 | Instalan Docker Desktop y Tailscale, y entran a Tailscale con **la misma cuenta** (es lo más simple). |
| 3 | Cada uno | Clona el repositorio en la rama `Jhojan_Code` y anota su IP con `tailscale ip -4`. |
| 4 | Los 3 | Crean `distribuido/.env` con las 3 IPs. Tiene que ser **idéntico** en las tres PCs; compártanlo por un canal privado. |
| 5 | Cada uno | Levanta **solo su parte** (paso 4). El orden no importa. |
| 6 | Cualquiera | Abre `http://<IP-de-la-PC-Asia>:3001` para usar el sistema. |

> **Lo más probable que falle: el Firewall de Windows.** Si el `ping` entre las PCs funciona pero `/api/health` muestra una base en `false`, permitan a Docker Desktop en el firewall (redes privadas y públicas) de la PC que tiene esa base. Se comprueba con `Test-NetConnection <IP> -Port 5432` (5435 para Europa, 27017 para Asia). Más detalles en *Problemas comunes*.

> **Para la presentación:** desconectar Tailscale en una PC es una **caída de red real**, no una simulada con Docker, y el sistema sigue funcionando con las otras dos. Es la mejor forma de demostrar que el sistema es distribuido.

## Paso 1 · Instalar lo necesario (en las 3 PCs)

1. **Docker Desktop:** <https://www.docker.com/products/docker-desktop/>. Ábranlo y esperen a que diga que está en ejecución.
2. **Git**, y el proyecto en la **misma rama** en las tres PCs:
   ```powershell
   git clone https://github.com/Zamallita2/Destribuidos_Practica3.git
   cd Destribuidos_Practica3
   git checkout Jhojan_Code
   git pull
   ```
3. **Tailscale:** <https://tailscale.com/download>. Instálenlo e inicien sesión.

## Paso 2 · Unir las 3 PCs en la misma red de Tailscale

Las tres PCs tienen que quedar en **la misma red de Tailscale** (en Tailscale se llama *tailnet*). Hay dos formas:

- **Más simple:** una sola persona crea la cuenta, y las otras dos PCs inician sesión en Tailscale con esa misma cuenta.
- **Cada uno con su cuenta:** el dueño de la red invita a los demás desde <https://login.tailscale.com/admin/users>.

Después, en cada PC, anoten su IP de Tailscale:

```powershell
tailscale ip -4
```

Da algo como `100.101.102.103`. También aparece en el ícono de Tailscale, en la barra de tareas.

**Comprueben la conexión.** Desde cada PC hagan `ping` a las IPs de las otras dos:

```powershell
ping 100.101.102.103
```

Si el `ping` no responde, revisen que Tailscale esté conectado en ambas PCs.

## Paso 3 · Crear el archivo de configuración (en las 3 PCs)

En la carpeta `distribuido/`, copien `.env.example` como `.env`:

```powershell
cd distribuido
copy .env.example .env      # en Mac/Linux: cp .env.example .env
```

Abran `.env` y pongan las 3 IPs reales y contraseñas propias. **El archivo tiene que ser idéntico en las 3 PCs:**

```ini
AM_HOST=100.x.x.x   # IP de Tailscale de la PC América
EU_HOST=100.x.x.x   # IP de Tailscale de la PC Europa
AS_HOST=100.x.x.x   # IP de Tailscale de la PC Asia
POSTGRES_PASSWORD=una-clave-que-elijan
MONGO_PASSWORD=otra-clave-que-elijan
BOARDING_PASS_SECRET=un-secreto-largo-cualquiera
```

El `.env` no se sube a GitHub, porque está en `.gitignore`. Compártanlo por un canal privado.

## Paso 4 · Levantar cada PC

Cada integrante, desde la carpeta `distribuido/`, ejecuta **solo el comando de su PC**:

```powershell
# PC 1 · América
docker compose -f docker-compose.pc-america.yml up -d --build

# PC 2 · Europa
docker compose -f docker-compose.pc-europa.yml up -d --build

# PC 3 · Asia
docker compose -f docker-compose.pc-asia.yml up -d --build
```

La primera vez tarda varios minutos, porque se descargan y compilan las imágenes.

**El orden no importa.** El servidor América espera hasta 10 minutos a que la base de Europa esté encendida, y recién entonces importa los vuelos. Pueden seguirlo con:

```powershell
docker compose -f docker-compose.pc-america.yml logs -f backend_am
```

Mientras espera muestra `Waiting for databases`. Cuando termina muestra `Successfully imported 345 vuelos` y `Successfully imported 1269 vuelos`.

## Paso 5 · Verificar

Desde **cualquiera** de las tres PCs, conectada a Tailscale:

- **Página web:** `http://<AS_HOST>:3001`
- **Estado de las 3 bases:** `http://<AS_HOST>:8080/api/health`. Debe decir `"status":"ok"`.
- **Relojes de los 3 servidores:** `http://<AS_HOST>:8080/api/cluster`. Los tres deben aparecer con `"up":true` y su propia zona horaria.

En la página, **Sincronización** muestra los tres servidores con su hora local, el reloj de Lamport y el reloj vectorial.

## Paso 6 · Demostración de fallas reales

| Qué hacer | Qué se observa |
|---|---|
| En la PC Europa: `docker compose -f docker-compose.pc-europa.yml stop` | El servidor Europa y su base aparecen **DOWN**. Las compras desde Europa las atiende otro servidor y se guardan en la otra base. |
| Volver a encenderla: `docker compose -f docker-compose.pc-europa.yml start` | En uno o dos minutos las tres bases vuelven a tener los mismos boletos y la cola queda en 0. |
| **Desconectar Tailscale** en una PC (clic en el ícono → *Disconnect*) | Es una **caída de red real**: esa PC deja de ser visible para las otras dos. El sistema sigue funcionando con las otras dos PCs. |
| Comprar el mismo asiento desde dos navegadores con países de regiones distintas | Una compra funciona y la otra dice *asiento ocupado*: el bloqueo distribuido coordina las dos bases. |

> **Para la entrevista.** Si se desconecta la PC América o la PC Europa, el sistema sigue vendiendo con la base que queda. Pero si las dos bases quedan separadas y **ambas** siguen recibiendo compras (una partición de red), no se puede garantizar al mismo tiempo la consistencia y la disponibilidad: es el teorema CAP. El sistema prioriza la disponibilidad y concilia los datos al reconectarse. Está explicado en `docs/arquitectura-distribuida.md`, sección 9.

## Problemas comunes

| Problema | Solución |
|---|---|
| Una PC no alcanza a otra (el `ping` falla) | Revisen que Tailscale diga *Connected* en las dos PCs y que estén en la misma red de Tailscale. |
| El `ping` responde pero `/api/health` dice que una base está en `false` | En Windows, permitan a Docker Desktop en el **Firewall de Windows** (redes privadas y públicas). Pruébenlo con `Test-NetConnection 100.x.x.x -Port 5432` (use 5435 para Europa y 27017 para Asia). |
| América sigue en `Waiting for databases` | La base de Europa no es alcanzable: revisen `EU_HOST` en el `.env` y el firewall de la PC Europa. |
| Cambiaron las contraseñas del `.env` después del primer arranque | Las bases guardan la contraseña original. Borren los datos y vuelvan a crearlos con `docker compose -f docker-compose.pc-XXX.yml down -v` y después `up -d`, en las 3 PCs. |
| El QR del pase no abre en el celular | El celular también necesita Tailscale instalado y conectado, porque el QR apunta a `http://<AS_HOST>:3001`. |
| Quieren empezar de cero | En cada PC: `docker compose -f docker-compose.pc-XXX.yml down -v`, y después levántenlas de nuevo. |

## Apagar todo

```powershell
docker compose -f docker-compose.pc-XXX.yml down      # conserva los datos
docker compose -f docker-compose.pc-XXX.yml down -v   # borra también los datos
```

Para trabajar en una sola PC, como antes, sigue funcionando `docker compose up -d --build` desde la carpeta principal del proyecto.
