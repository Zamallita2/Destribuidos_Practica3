# Entregables — Práctica 3

**Aerolíneas Rafael Pabón · Sincronización de procesos distribuidos**

| # | Entregable | Dónde está |
|---|---|---|
| 1 | Código fuente del sistema | Este repositorio: `backend/` (Go), `frontend/` (Next.js), `gateway/` (nginx) y `docker-compose.yml` |
| 2 | Documentación técnica: arquitectura distribuida | [arquitectura-distribuida.md](arquitectura-distribuida.md), secciones 1, 2 y 8 |
| 2 | Documentación técnica: estrategia de sincronización | [arquitectura-distribuida.md](arquitectura-distribuida.md), secciones 3 y 4 |
| 3 | Dashboard funcional | Panel general `/` y panel por vuelo `/dashboard/vuelos/{id}` (capturas en la evidencia) |
| 4 | Archivos PDF de pasajes | [pasajes/](pasajes/): 3 PDF y 1 pase de Wallet (`.pkpass`) |
| 5 | Evidencia de funcionamiento | [evidencia-funcionamiento.md](evidencia-funcionamiento.md), [registro de pruebas](evidencia/registro-de-pruebas.txt) y [pruebas automáticas](evidencia/pruebas-automaticas.txt) |

Documentos complementarios:

- [implementacion-final.md](implementacion-final.md): detalles de importación, ocupación y pases.
- [gestion-datos-entrada.md](gestion-datos-entrada.md): cómo reemplazar el dataset y las matrices.
- [demo-sincronizacion.md](demo-sincronizacion.md): guía para demostrar caídas de servidores y bases.
- [qr-red-local.md](qr-red-local.md): QR del pase desde un celular.
- [../distribuido/GUIA.md](../distribuido/GUIA.md): despliegue real en 3 PCs conectadas con Tailscale.

## Ejecución

```bash
docker compose up -d --build
```

- Interfaz: <http://localhost:3001>
- Estado de las bases: <http://localhost:8080/api/health>
- Relojes de los tres servidores: <http://localhost:8080/api/cluster>
