# Reports And Exports API

Los informes de tiempo permiten previsualizar totales agrupados y exportar CSV o JSON para un rango de fechas.

## Rutas HTTP

Todas las rutas requieren cookie de sesion valida.

```text
GET /api/v1/reports/time
GET /api/v1/reports/time/export
```

### Parametros

```text
GET /api/v1/reports/time?from=2026-07-01T00:00:00Z&to=2026-07-31T23:59:59Z
GET /api/v1/reports/time?from=...&to=...&groupBy=project
GET /api/v1/reports/time?from=...&to=...&includeTimestamps=true
GET /api/v1/reports/time?from=...&to=...&billableOnly=true
GET /api/v1/reports/time/export?format=csv&from=...&to=...
GET /api/v1/reports/time/export?format=json&from=...&to=...
```

| Parametro | Regla |
| --- | --- |
| `from`, `to` | Obligatorios, RFC3339. Filtran por `started_at`. |
| `groupBy` | `day`, `client`, `project`, `task`. Default `project`. Ignorado cuando `includeTimestamps=true`. |
| `includeTimestamps` | `true` devuelve entradas detalladas con inicio/fin. `false` devuelve solo totales agrupados. |
| `billableOnly` | `true` excluye entradas no facturables. |
| `format` | Solo en `/export`: `csv` (default) o `json`. |

## Respuesta JSON (resumen)

```json
{
  "from": "2026-07-01T00:00:00.000000000Z",
  "to": "2026-07-31T23:59:59.000000000Z",
  "groupBy": "project",
  "includeTimestamps": false,
  "billableOnly": false,
  "totalSeconds": 5400,
  "entryCount": 2,
  "groups": [
    {
      "key": "prj_...",
      "label": "Portal Web",
      "totalSeconds": 5400,
      "entryCount": 2
    }
  ]
}
```

## CSV

- **Resumen:** columnas `group`, `label`, `entry_count`, `total_seconds`, `total_duration` + fila total.
- **Detallado:** columnas `description`, `client`, `project`, `task`, `started_at`, `ended_at`, `duration_seconds`, `billable`, `tags`.

## Frontend

- Panel `#overview` en el dashboard con filtros, vista previa y botones de exportacion.
- Ancla `#detailed` reservada para el modo detallado con marcas de tiempo.

## Donde leer el codigo

| Capa | Ubicacion |
| --- | --- |
| Agregacion | `apps/api/internal/store/report.go` |
| Tests de store | `apps/api/internal/store/report_test.go` |
| Handlers HTTP | `apps/api/internal/httpapi/reports.go` |
| Tests HTTP | `apps/api/internal/httpapi/router_test.go` (`TestTimeReportExport`) |
| Cliente API | `apps/web/src/lib/api.ts` |
| UI | `apps/web/src/lib/reportUi.tsx` |

## Comprobaciones recomendadas

```bash
make test
make build-web
```
