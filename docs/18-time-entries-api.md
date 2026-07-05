# API de Entradas de Tiempo Manuales

Las entradas manuales son la quinta slice CRUD de producto en `leotime`. Permiten registrar bloques de tiempo finalizados con precision de un minuto, enlazar cliente/proyecto/tarea/tags y marcar solapamientos sin bloquear la operacion.

## Rutas HTTP

Todas las rutas requieren una cookie de sesion valida.

```text
GET    /api/v1/time-entries
POST   /api/v1/time-entries
GET    /api/v1/time-entries/{timeEntryID}
PATCH  /api/v1/time-entries/{timeEntryID}
DELETE /api/v1/time-entries/{timeEntryID}
```

### Filtros de listado

```text
GET /api/v1/time-entries?from=2026-06-01T00:00:00Z
GET /api/v1/time-entries?to=2026-06-30T23:59:59Z
GET /api/v1/time-entries?clientId=cli_...
GET /api/v1/time-entries?projectId=prj_...
GET /api/v1/time-entries?taskId=tsk_...
```

La lista devuelve solo entradas finalizadas (`ended_at IS NOT NULL`), ordenadas por `started_at` descendente, maximo 500 filas.

## Cuerpo de peticion

Crear y actualizar usan el mismo cuerpo JSON:

```json
{
  "clientId": "cli_...",
  "projectId": "prj_...",
  "taskId": "tsk_...",
  "tagIds": ["tag_..."],
  "description": "Refactor API",
  "startedAt": "2026-06-29T08:04:00Z",
  "endedAt": "2026-06-29T10:55:00Z",
  "billable": true
}
```

## Validacion

| Campo | Regla |
| --- | --- |
| `startedAt`, `endedAt` | Obligatorios, formato RFC3339. Se truncan al minuto en UTC. |
| Duracion | Minimo 1 minuto. `endedAt` debe ser posterior a `startedAt`. |
| `clientId`, `projectId`, `taskId` | Opcionales, pero deben referenciar recursos activos del usuario. |
| Relaciones | Si hay tarea/proyecto, se infieren y validan cliente/proyecto coherentes. |
| `tagIds` | Cada tag debe existir para el usuario. |
| Solapamiento | Se calcula `overlapWarning`, pero **no bloquea** crear ni editar. |

`durationSeconds` se calcula en backend a partir de las marcas normalizadas.

## Borrado

`DELETE` elimina la entrada manual finalizada. Las filas en `time_entry_tags` se eliminan en cascada.

## Respuesta de ejemplo

```json
{
  "id": "ten_...",
  "clientId": "cli_...",
  "clientName": "Acme",
  "projectId": "prj_...",
  "projectName": "Portal Web",
  "projectColor": "#2563eb",
  "taskId": "tsk_...",
  "taskName": "Refactor API",
  "description": "Refactor API",
  "startedAt": "2026-06-29T08:04:00.000000000Z",
  "endedAt": "2026-06-29T10:55:00.000000000Z",
  "durationSeconds": 10260,
  "billable": true,
  "overlapWarning": false,
  "source": "manual",
  "tags": [
    { "id": "tag_...", "name": "Deep Work", "color": "#64748b" }
  ],
  "createdAt": "2026-07-05T12:00:00.000000000Z",
  "updatedAt": "2026-07-05T12:00:00.000000000Z"
}
```

## Frontend

- El timesheet (`#timesheet`) muestra entradas reales agrupadas por dia.
- El panel `#manual-time-entry` permite crear, editar y eliminar entradas manuales.
- El boton **Entrada manual** hace scroll al formulario.

## Donde leer el comportamiento

| Capa | Ubicacion |
| --- | --- |
| Persistencia y reglas | `apps/api/internal/store/time_entry.go` |
| Tests de store | `apps/api/internal/store/time_entry_test.go` |
| Handlers HTTP | `apps/api/internal/httpapi/time_entries.go` |
| Tests de integracion HTTP | `apps/api/internal/httpapi/router_test.go` (`TestTimeEntryHTTPLifecycle`) |
| Cliente API web | `apps/web/src/lib/api.ts` |
| UI timesheet + formulario | `apps/web/src/lib/timeEntryUi.tsx` |
| Tests de UI | `apps/web/src/App.test.tsx` |

## Comprobaciones recomendadas

```bash
make pre-commit
```
