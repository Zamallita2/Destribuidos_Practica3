# Archivos de Ejemplo — Datos de Entrada

Aeropuertos usados: **ATL · PEK · DXB · TYO · LON · LAX · PAR · FRA · IST · SIN · MAD · AMS · DFW · CAN · SAO**

---

## ¿Qué sube en cada campo?

| Campo del formulario | Archivo de ejemplo | Formatos aceptados |
|---|---|---|
| **Tres matrices** (todo junto) | `ejemplo_matrices_completas.json` | Solo `.json` |
| **Matriz de tiempos** | `ejemplo_tiempos.csv` / `.xlsx` | `.json` `.csv` `.xlsx` |
| **Precios turista** | `ejemplo_tarifas_turista.csv` / `.xlsx` | `.json` `.csv` `.xlsx` |
| **Precios primera clase** | `ejemplo_tarifas_primera.json` / `.xlsx` | `.json` `.csv` `.xlsx` |

> **Nota:** Si subes el JSON combinado (`ejemplo_matrices_completas.json`) en el campo "Tres matrices",
> **no** necesitas subir los otros tres archivos por separado.

---

## Archivos incluidos

### `ejemplo_matrices_completas.json`
Un solo JSON con las tres matrices + lista de aeropuertos.
Úsalo cuando quieras cambiar todo a la vez.

```json
{
  "airports": ["ATL", "DFW", "LAX", "LON", "MAD", "PAR"],
  "travel_time":      { ... },   // horas de vuelo entre pares
  "economy_fares":    { ... },   // USD turista  — null = no disponible
  "first_class_fares":{ ... }    // USD primera  — null = no disponible
}
```

---

### `ejemplo_tiempos.csv` / `ejemplo_tiempos.xlsx`
Solo la matriz de tiempos de vuelo **en horas**.

Formato (CSV):
```
,ATL,DFW,LAX,LON,MAD,PAR
ATL,0,2,5,9,9,10
DFW,2,0,3,10,10,11
...
```
- Primera celda vacía (esquina superior izquierda).
- `0` = mismo aeropuerto.
- Todos los pares deben tener valor (no dejar vacíos aquí).

---

### `ejemplo_tarifas_turista.csv` / `ejemplo_tarifas_turista.xlsx`
Precios de clase turista **en USD**.

```
,ATL,DFW,LAX,LON,MAD,PAR
ATL,0,200,400,800,,
DFW,200,0,300,,850,
...
```
- **Celda vacía** = ruta no disponible en clase turista.
- `0` = mismo aeropuerto (siempre 0).

---

### `ejemplo_tarifas_primera.json` / `ejemplo_tarifas_primera.xlsx`
Precios de primera clase **en USD**.

JSON independiente (sin el campo `airports`):
```json
{
  "ATL": { "ATL": 0, "DFW": 270, "LAX": 540, "LON": null, ... },
  ...
}
```
- `null` = ruta no disponible en primera clase.
- Una ruta puede tener turista pero no primera (y viceversa).

---

## Reglas importantes

1. **Códigos de aeropuerto**: exactamente 3 letras mayúsculas.
   Solo se aceptan los códigos ya registrados en el sistema (`airports.json`).
   Los 15 actuales: `ATL PEK DXB TYO LON LAX PAR FRA IST SIN MAD AMS DFW CAN SAO`

2. **Ruta válida**: debe tener tiempo > 0 **y** al menos una tarifa (turista o primera) > 0.

3. **Matrices completas**: cada código en `airports` debe aparecer como fila y columna
   en las tres matrices. No puede haber celdas faltantes.

4. **Precios**: valores en USD, positivos o `null`/vacío.
   Los tiempos siempre son obligatorios (no pueden ser `null`).

5. **Tamaño máximo**: 10 MB por archivo de matrices, 50 MB para el dataset de vuelos.
