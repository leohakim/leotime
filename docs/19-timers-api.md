# API de Timers

Los timers son entradas de tiempo con `source='timer'` y `ended_at` nulo mientras estan activos. Permiten iniciar, actualizar y parar seguimiento en vivo con awareness de solapamientos.

## Rutas HTTP

Todas las rutas requieren una cookie de sesion valida.

```text
GET    /api/v1/timers
POST   /api/v1/timers
PATCH  /api/v1/timers/{timeEntryID}
POST   /api/v1/timers/{timeEntryID}/stop
DELETE /api/v1/timers/{timeEntryID}
```

## Iniciar timer

```json
POST /api/v1/timers
{
  "clientId": "cli_...",
  "projectId": "prj_...",
  "taskId": "tsk_...",
  "tagIds": ["tag_..."],
  "description": "Refactor API",
  "billable": true
}
```

- `startedAt` se fija en backend al minuto actual (UTC).
- Se permiten varios timers abiertos a la vez.
- Si hay solapamiento con otra entrada (incluidos otros timers abiertos), se marca `overlapWarning` sin bloquear.

## Parar timer

```text
POST /api/v1/timers/{timeEntryID}/stop
```

- `endedAt` se fija en backend al minuto actual (UTC).
- Duracion minima: 1 minuto.
- Recalcula `overlapWarning` al cerrar.
- La entrada pasa a formar parte del timesheet (`ended_at IS NOT NULL`).

## Actualizar timer abierto

```text
PATCH /api/v1/timers/{timeEntryID}
```

Permite cambiar descripcion, cliente/proyecto/tarea, tags, billable y `startedAt` mientras el timer sigue activo.

- `startedAt` es opcional en el PATCH; debe ser RFC3339 y no puede estar en el futuro.
- `endedAt` no es editable mientras el timer sigue corriendo.

## Descartar timer

```text
DELETE /api/v1/timers/{timeEntryID}
```

Elimina un timer abierto sin crear entrada finalizada.

## Listar timers abiertos

```text
GET /api/v1/timers
```

Devuelve `{ "timers": [...] }` ordenados por `started_at` descendente.

## Aviso por timer largo (email)

Si un timer abierto supera el umbral configurado (default **8 horas**), leotime envia **un solo email** al correo del usuario.

- El scheduler corre dentro del mismo proceso Go (no hay contenedor aparte).
- La deteccion ocurre cada 10 minutos por defecto (`LEOTIME_SCHEDULER_SCAN_INTERVAL`).
- El umbral vive en `app_settings.timer_still_running_hours` y se edita en Ajustes (`timerStillRunningHours`, 1–24).
- Se puede desactivar con `timerStillRunningEnabled` en Ajustes o `timer_still_running_enabled` en base de datos.
- Tras un envio exitoso se guarda `time_entries.still_active_email_sent_at`.
- En local, `LEOTIME_MAIL_MODE=log` escribe el mail en los logs del contenedor.

Documentacion completa: `docs/29-email-notifications.md`.

## Frontend

- La barra principal (`TimerCommandRow`) muestra el timer activo mas reciente o el formulario de inicio.
- El sidebar (`SidebarTimer`) muestra el reloj en vivo y el boton de parar.
- El reloj se actualiza en cliente cada segundo; la lista de timers se refresca cada 30 s mientras haya timers abiertos.

## Donde leer el comportamiento

| Capa | Ubicacion |
| --- | --- |
| Persistencia y reglas | `apps/api/internal/store/timer.go` |
| Aviso timer largo | `apps/api/internal/store/still_running.go`, `apps/api/internal/notify/` |
| Tests de store | `apps/api/internal/store/timer_test.go` |
| Handlers HTTP | `apps/api/internal/httpapi/timers.go` |
| Tests de integracion HTTP | `apps/api/internal/httpapi/router_test.go` (`TestTimerHTTPLifecycle`) |
| Cliente API web | `apps/web/src/lib/api.ts` |
| UI timer | `apps/web/src/lib/timerUi.tsx` |
| Tests de UI | `apps/web/src/App.test.tsx` |

## Comprobaciones recomendadas

```bash
make pre-commit
```
