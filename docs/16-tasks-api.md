# API de Tareas

Las tareas son la tercera slice CRUD de producto en `leotime`. Representan unidades de trabajo que luego se asignan a entradas de tiempo. Cada tarea pertenece a un usuario, puede vincularse opcionalmente a un proyecto activo y define si el tiempo registrado será facturable por defecto.

## Rutas HTTP

Todas las rutas requieren una cookie de sesión valida.

```text
GET    /api/v1/tasks
POST   /api/v1/tasks
GET    /api/v1/tasks/{taskID}
PATCH  /api/v1/tasks/{taskID}
DELETE /api/v1/tasks/{taskID}
```

### Filtros de listado

Incluir tareas archivadas:

```text
GET /api/v1/tasks?includeArchived=true
```

Filtrar por proyecto:

```text
GET /api/v1/tasks?projectId=prj_...
```

## Cuerpo de peticion

Crear y actualizar usan el mismo cuerpo JSON:

```json
{
  "projectId": "prj_...",
  "name": "Refactor API",
  "billable": true
}
```

- `projectId` puede ir vacio si la configuracion del usuario no exige proyecto.
- `billable` indica el valor por defecto para futuras entradas de tiempo.

## Validacion

| Campo | Regla |
| --- | --- |
| `name` | Obligatorio. Se recorta espacios al inicio y al final. |
| `projectId` | Opcional por defecto. Debe referenciar un proyecto activo del mismo usuario cuando se envia. |
| `projectId` | Obligatorio cuando `app_settings.task_project_required = 1` para ese usuario. |
| `billable` | Booleano. Se persiste como `0` o `1` en SQLite. |

Errores de validacion devuelven `400 Bad Request` con un mensaje de texto plano.

## Borrado

`DELETE` no elimina la fila. Marca `archived_at` con la fecha actual para mantener historial de entradas e importaciones. Si la tarea ya estaba archivada, la fecha no cambia.

## Respuesta de ejemplo

```json
{
  "id": "tsk_...",
  "projectId": "prj_...",
  "projectName": "Portal Web",
  "projectColor": "#2563eb",
  "name": "Refactor API",
  "billable": true,
  "archivedAt": "",
  "createdAt": "2026-07-05T12:00:00.000000000Z",
  "updatedAt": "2026-07-05T12:00:00.000000000Z"
}
```

`projectName` y `projectColor` vienen del join con `projects` para facilitar la UI.

## Frontend

El dashboard incluye un panel de tareas (`#tasks`) que permite:

- Listar tareas activas.
- Crear tareas con nombre, proyecto opcional y checkbox facturable.
- Editar tareas existentes.
- Archivar tareas.

Tras cada mutacion, la UI invalida las queries `tasks` y `overview` para mantener contadores alineados.

## Donde leer el comportamiento

| Capa | Ubicacion |
| --- | --- |
| Persistencia y reglas | `apps/api/internal/store/task.go` |
| Tests de store | `apps/api/internal/store/task_test.go` |
| Handlers HTTP | `apps/api/internal/httpapi/tasks.go` |
| Tests de integracion HTTP | `apps/api/internal/httpapi/router_test.go` (`TestTaskHTTPLifecycle`) |
| Cliente API web | `apps/web/src/lib/api.ts` |
| UI y validacion local | `apps/web/src/App.tsx` (`TaskPanel`) |
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
