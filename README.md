# Aerolíneas Pabón — Práctica 3 (Sistemas Distribuidos)

Sistema de reservas de vuelos distribuido en dos PostgreSQL regionales (América y Europa/Asia) y un MongoDB central, sincronizados mediante outbox, relojes vectoriales y relojes de Lamport.

## Componentes

| Servicio      | Tecnología        | Puerto local |
|---------------|-------------------|--------------|
| `frontend`    | Next.js           | 3001         |
| `backend`     | Go (API REST)     | 8080         |
| `postgres_am` | PostgreSQL 16     | 5432         |
| `postgres_eu` | PostgreSQL 16     | 5435         |
| `mongodb`     | MongoDB           | 27017        |

## Arranque rápido

```bash
docker compose up -d --build
docker compose logs --tail=30 backend
```

- Interfaz: <http://localhost:3001>
- Estado de los nodos: <http://localhost:8080/api/health>

En Windows, `scripts/start-project.ps1` levanta el proyecto y mantiene actualizada la IP de la red local para los QR de los boletos.

La primera carga importa unos 30.000 vuelos y genera los manifiestos de ocupación en segundo plano; puede tardar varios minutos, pero la API responde durante ese proceso.

## Estructura

```
backend/        API en Go (handlers, rutas, acceso a datos, sincronización)
frontend/       Aplicación Next.js
dataset/        CSV de vuelos de entrada
init-scripts/   Scripts de inicialización de PostgreSQL y MongoDB
scripts/        Utilidades de arranque (PowerShell)
docs/           Documentación técnica
consigna/       Enunciado de la práctica
```

## Documentación

- [Implementación y operación](docs/implementacion-final.md)
- [Gestión de datos de entrada](docs/gestion-datos-entrada.md)
- [Demostración de caída de nodos](docs/demo-sincronizacion.md)
- [Verificación](docs/verificacion.md)
- [QR del boleto en otra red Wi-Fi](docs/qr-red-local.md)
