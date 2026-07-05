# Weekly Timesheet View

La vista semanal es la primera slice de producto despues del timer. Muestra una cuadricula de 7 dias con totales diarios y semanal, navegacion por semana, y edicion inline de cada entrada.

## Comportamiento

- La semana empieza en **lunes** (locale ES/EN del dashboard).
- El timesheet (`#timesheet`) carga entradas con `GET /api/v1/time-entries?from=...&to=...` acotadas a la semana visible.
- Siempre se renderizan **7 grupos de dia**, aunque no haya entradas (total `0h 00min`), con el **dia mas reciente arriba** (domingo/hoy primero, lunes al final).
- Las entradas dentro de cada dia se ordenan por hora de inicio **descendente** (la mas reciente primero).
- La barra superior incluye:
  - semana anterior / siguiente
  - rango de fechas localizado
  - boton **Hoy** cuando la semana visible no es la actual
  - **Total semana** con suma de duraciones
- La semana seleccionada se persiste en `localStorage` (`leotime.timesheetWeek`).

## Edicion

Cada fila del timesheet sigue siendo editable inline (descripcion, proyecto, inicio, fin) con debounce y validacion existentes en `timeEntryUi.tsx`.

## Iconos del timer (Solidtime-like)

- **Play**: triangulo relleno (`Play` de Lucide) en verde para iniciar.
- **Stop**: cuadrado blanco relleno (`Square`) dentro del boton circular rojo.
- Sidebar sin timer activo: indicador play atenuado; con timer activo: boton stop claro.

## Donde leer el codigo

| Capa | Ubicacion |
| --- | --- |
| Semana (utilidades) | `apps/web/src/lib/timesheetWeek.ts` |
| Tests de semana | `apps/web/src/lib/timesheetWeek.test.ts` |
| UI timesheet | `apps/web/src/lib/timeEntryUi.tsx` |
| Iconos play/stop | `apps/web/src/lib/timerIcons.tsx`, `apps/web/src/lib/timerUi.tsx` |
| Estado semana | `apps/web/src/App.tsx` |
| Cliente API | `apps/web/src/lib/api.ts` (`fetchTimeEntries` con `from`/`to`) |
| Tests UI | `apps/web/src/App.test.tsx` |

## Comprobaciones recomendadas

```bash
make test-web
make build-web
```
