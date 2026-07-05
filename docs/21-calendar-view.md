# Calendar View

La vista de calendario permite inspeccionar y editar entradas de tiempo en una cuadricula mensual, reutilizando la edicion inline del timesheet.

## Comportamiento

- Alternar entre **Timesheet** y **Calendario** con el selector encima de la lista o desde la sidebar (`#calendar`).
- La cuadricula muestra semanas completas (lunes a domingo) con dias del mes anterior/siguiente atenuados.
- Cada celda muestra el numero del dia, total trabajado y recuento de entradas cuando aplica.
- Al seleccionar un dia, debajo aparecen sus entradas editables inline (descripcion, proyecto, inicio, fin).
- Navegacion mensual con mes anterior / siguiente, boton **Hoy** fuera del mes actual, y **Total mes**.
- Las entradas se cargan con `GET /api/v1/time-entries?from=...&to=...` acotadas al mes visible.
- El mes y el dia seleccionado se persisten en `localStorage` (`leotime.calendarMonth`, `leotime.calendarDay`).

## Donde leer el codigo

| Capa | Ubicacion |
| --- | --- |
| Mes (utilidades) | `apps/web/src/lib/calendarMonth.ts` |
| Tests de mes | `apps/web/src/lib/calendarMonth.test.ts` |
| UI calendario | `apps/web/src/lib/calendarUi.tsx` |
| Filas editables reutilizadas | `apps/web/src/lib/timeEntryUi.tsx` (`TimesheetEntryRow`) |
| Vista activa y queries | `apps/web/src/App.tsx` |
| Tests UI | `apps/web/src/App.test.tsx` |

## Comprobaciones recomendadas

```bash
make test-web
make build-web
```
