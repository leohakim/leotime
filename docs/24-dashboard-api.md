# Dashboard API

El panel `#dashboard` resume actividad reciente, ultimos 7 dias, heatmap, semana actual y reparto por proyecto.

## Ruta HTTP

Requiere cookie de sesion valida.

```text
GET /api/v1/dashboard/stats
GET /api/v1/dashboard/stats?activityMonth=2026-07
```

## Respuesta JSON

```json
{
  "recentEntries": [
    {
      "id": "ten_...",
      "clientId": "cli_...",
      "projectId": "prj_...",
      "projectName": "Portal Web",
      "projectColor": "#2563eb",
      "taskId": "tsk_...",
      "description": "Support",
      "startedAt": "2026-07-05T09:00:00Z",
      "durationSeconds": 3600,
      "billable": true
    }
  ],
  "lastSevenDays": [
    { "date": "2026-07-05", "label": "today", "totalSeconds": 7200 }
  ],
  "activityHeatmap": [
    { "date": "2026-07-05", "totalSeconds": 7200, "level": 2 }
  ],
  "weekDays": [
    { "date": "2026-06-30", "weekday": "Mon", "totalSeconds": 3600 }
  ],
  "weekSpentSeconds": 30600,
  "weekBillableSeconds": 12600,
  "weekBillableMinor": 21000,
  "weekCurrency": "EUR",
  "projectBreakdown": [
    {
      "projectId": "prj_...",
      "projectName": "Portal Web",
      "projectColor": "#2563eb",
      "totalSeconds": 18000
    }
  ]
}
```

## Reglas

| Campo | Regla |
| --- | --- |
| `recentEntries` | Hasta 5 entradas finalizadas mas recientes. |
| `lastSevenDays` | Hoy + 6 dias previos. `label`: `today`, `yesterday`, `2d`… |
| `activityMonth` | Mes visible en el heatmap (`YYYY-MM`). Parametro `activityMonth` en la query. |
| `activityHeatmap` | Dias del mes seleccionado mas relleno semanal. `inMonth=false` para padding. |
| `weekDays` | Semana actual empezando en lunes UTC. |
| `weekBillableMinor` | Suma de lineas facturables con tarifa proyecto > cliente. |
| `projectBreakdown` | Reparto de la semana actual por proyecto. |

## Frontend

`DashboardPanel` en `#dashboard` muestra:

- entradas recientes con boton play para reiniciar timer,
- barras de ultimos 7 dias,
- heatmap estilo GitHub,
- placeholder single-owner en lugar de team activity,
- grafico semanal, totales y donut por proyecto.

## Tests

- Store: `apps/api/internal/store/dashboard_test.go`
- HTTP: `TestDashboardStatsHTTP` en `router_test.go`
- Web: `dashboardHeatmap.test.ts`, `renders dashboard stats widgets` en `App.test.tsx`
