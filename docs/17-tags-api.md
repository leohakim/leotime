# API de Tags

Los tags son la cuarta slice CRUD de producto en `leotime`. Sirven como taxonomia para clasificar entradas de tiempo en informes y en la UI diaria. Cada tag pertenece a un usuario, tiene nombre unico entre tags activos (sin distinguir mayusculas/minusculas) y un color visible.

## Rutas HTTP

Todas las rutas requieren una cookie de sesion valida.

```text
GET    /api/v1/tags
POST   /api/v1/tags
GET    /api/v1/tags/{tagID}
PATCH  /api/v1/tags/{tagID}
DELETE /api/v1/tags/{tagID}
POST   /api/v1/tags/{tagID}/restore
```

### Filtros de listado

Incluir tags archivados:

```text
GET /api/v1/tags?includeArchived=true
```

## Cuerpo de peticion

Crear y actualizar usan el mismo cuerpo JSON:

```json
{
  "name": "Deep Work",
  "color": "#64748b"
}
```

- `color` es opcional en la peticion. Si falta, el backend usa `#64748b`.

## Validacion

| Campo | Regla |
| --- | --- |
| `name` | Obligatorio. Se recortan espacios. Unico por usuario entre tags activos comparando con `lower(name)`. |
| `color` | Debe ser un color hex como `#64748b`. Por defecto `#64748b`. |

Errores de validacion o nombre duplicado devuelven `400 Bad Request`.

## Borrado

`DELETE` archiva el tag estableciendo `archived_at`. Las filas en `time_entry_tags` se mantienen para preservar el historial de entradas.

`POST /restore` limpia `archived_at` y devuelve el tag restaurado.

## Respuesta de ejemplo

```json
{
  "id": "tag_...",
  "name": "Deep Work",
  "color": "#2563eb",
  "archivedAt": "",
  "createdAt": "2026-07-05T12:00:00.000000000Z",
  "updatedAt": "2026-07-05T12:00:00.000000000Z"
}
```

## Frontend

El dashboard incluye un panel de tags (`#tags`) que permite:

- Listar tags activos y archivados (`includeArchived=true`).
- Crear tags con nombre y color.
- Editar tags existentes.
- Archivar tags desde la lista o desmarcando **Tag activo** en el formulario de edicion.
- Reactivar tags archivados desde el formulario de edicion.

Tras cada mutacion, la UI invalida las queries `tags` y `overview`.

## Donde leer el comportamiento

| Capa | Ubicacion |
| --- | --- |
| Persistencia y reglas | `apps/api/internal/store/tag.go` |
| Tests de store | `apps/api/internal/store/tag_test.go` |
| Handlers HTTP | `apps/api/internal/httpapi/tags.go` |
| Tests de integracion HTTP | `apps/api/internal/httpapi/router_test.go` (`TestTagHTTPLifecycle`) |
| Cliente API web | `apps/web/src/lib/api.ts` |
| UI y validacion local | `apps/web/src/App.tsx` (`TagPanel`) |
| Tests de UI | `apps/web/src/App.test.tsx` |

## Comprobaciones recomendadas

Backend:

```bash
make test-api
```

Frontend:

```bash
make test-web
make build-web
```

Slice completa:

```bash
make test
make smoke
```
