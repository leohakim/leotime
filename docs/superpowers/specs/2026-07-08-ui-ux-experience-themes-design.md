# UI/UX Experience Themes Design

## Estado

Aprobado para planificacion por conversacion el 2026-07-08.

Esta spec define la direccion de UI/UX para mejorar la estetica, navegacion,
claridad y responsividad de `leotime` mediante experiencias visuales
configurables. No implementa cambios por si misma.

## Estado de implementación

**Sprints 1–4 completados** (2026-07-11). La auditoria responsive y el mapa
de fricciones estan publicados en [UI/UX Visual Audit](../../36-ui-ux-visual-audit.md).
Sprint 2 aplica los atributos raiz, tokens semanticos y estado local de
experiencia. Sprint 3 expone el selector de experiencia y Sprint 4 refactoriza
el shell con navegacion inferior en tablet/mobile. Los sprints 5–10 siguen sin
implementar.

## Contexto

`leotime` ya tiene una UI funcional con sidebar, timer, timesheet, calendar,
dashboard, reports, invoices, settings, themes y layout modes. La base actual
cumple el MVP, pero todavia se siente como un scaffold operativo y no como una
herramienta suficientemente pulida para ofrecer incluso gratis.

El producto debe seguir siendo un workbench rapido de tracking e invoicing para
un owner primero. La mejora visual no debe convertir la app en una landing, un
dashboard decorativo ni una experiencia SaaS pesada.

## Objetivos

- Hacer la app mas atractiva sin perder velocidad diaria.
- Hacer que las pantallas principales sean faciles de usar, navegar y entender.
- Mejorar desktop, tablet y mobile como experiencias de primera clase.
- Permitir variedad real de personalidad visual, no solo cambios de color.
- Separar tema, layout, navegacion y presets para que la evolucion sea
  mantenible.
- Dejar una referencia versionada para imitar y actualizar SolidTime en el
  futuro.

## No objetivos

- No agregar complejidad SaaS, multiusuario o permisos por esta iniciativa.
- No cambiar reglas de negocio de time entries, timers, reports o invoices salvo
  que una mejora de UX lo requiera y se especifique en un sprint posterior.
- No copiar logos, marca, assets protegidos o datos de SolidTime. El preset
  SolidTime Exact debe imitar la experiencia visual y de layout usando recursos
  propios de leotime.
- No introducir imagenes decorativas, heroes, orbs, fondos promocionales ni
  componentes que tapen el flujo principal de captura de tiempo.

## Modelo de experiencia

La UI se configurara con controles separados:

- `theme`: paleta, contraste, superficies, estados y tono visual.
- `layout`: densidad, spacing, jerarquia de paneles, listas, cards y tablas.
- `nav`: tipo de navegacion.
- `preset`: combinacion sugerida de theme, layout y nav.

Ejemplo esperado en el DOM:

```html
<html data-theme="focus-dark" data-layout="compact" data-nav="sidebar-compact" data-preset="compact-power">
```

Los presets aplican combinaciones recomendadas. Si el usuario modifica un
control individual despues de aplicar un preset, el estado pasa a
`custom`/`Personalizado`.

## Presets iniciales

### Workbench Pro

Experiencia profesional, sobria y densa para uso diario en desktop. Debe sentirse
como una herramienta seria de trabajo, con alta legibilidad, datos al frente y
pocas distracciones.

### Calm Light

Experiencia clara, amable y con mas aire. Debe servir para usuarios nuevos o
personas que prefieren baja carga visual. El contraste debe seguir siendo fuerte
en textos, controles y estados.

### Focus Dark

Experiencia oscura, elegante y enfocada. El timer, el timesheet y las acciones
repetidas deben tener prioridad visual. Debe reducir ruido sin ocultar
informacion importante.

### Compact Power

Experiencia de alta densidad para usuarios intensivos. Debe optimizar escaneo,
edicion inline y navegacion rapida. No debe sacrificar targets tactiles minimos
en tablet/mobile.

### Mobile Flow

Experiencia mobile/tablet-first. Debe usar acciones grandes, navegacion inferior
cuando corresponda, layouts apilados estables y flujos tactiles claros para
iniciar/parar tiempo y revisar entradas.

### SolidTime Exact

Preset de referencia para imitar la UI/UX actual de SolidTime de forma
versionada.

Referencia fijada:

- Repo: https://github.com/solidtime-io/solidtime
- Release base: `v0.15.1`
- URL release: https://github.com/solidtime-io/solidtime/releases/tag/v0.15.1
- Fecha de release: 2026-06-24
- Commit de release: `ab9f6e6`
- Fecha de verificacion para esta spec: 2026-07-08
- Referencia visual publica: https://www.solidtime.io/

Al implementar o actualizar este preset, debe documentarse el tag o commit de
SolidTime usado como fuente. Cuando SolidTime cambie su UI/UX, se podra crear un
sprint de refresh que compare contra el nuevo release y actualice solo este
preset sin mezclarlo con los demas.

## Arquitectura UI

La implementacion debe evitar una clase gigante combinando cada caso, como
`.layout-compact .theme-dark`. La direccion preferida es una capa de tokens:

- Foundation tokens: colores base, tipografia, spacing, radius, shadows.
- Semantic tokens: `--surface`, `--text`, `--accent`, `--danger`, `--nav-bg`.
- Component tokens: timer, sidebar, tables, cards, forms, calendar, dashboard.
- Preset overrides: diferencias minimas necesarias para que cada experiencia
  se sienta distinta.

`DashboardShell` debe seguir siendo el shell principal, pero se recomienda
separar piezas para variar experiencias sin duplicar la app:

- `SidebarNav`
- `TopNav`
- `MobileBottomNav`
- `ShellToolbar`
- `ExperienceSwitcher`

La navegacion debe conservar el hash routing existente y no duplicar logica de
rutas por cada preset.

## Superficies principales a redisenar

1. Login y boot states.
2. Shell, sidebar, toolbar y navegacion responsive.
3. Timer activo y start timer flow.
4. Manual time entry.
5. Weekly timesheet.
6. Calendar view.
7. Dashboard.
8. Reports.
9. Invoices.
10. Settings, profile y selectors de experiencia.

## Roadmap de 10 sprints

### Sprint 1: Auditoria visual y UX

Capturar desktop, tablet y mobile de las pantallas principales. Producir un mapa
priorizado de friccion visual, navegacion, densidad, responsive, comprension,
estados vacios, loading y errores.

### Sprint 2: Arquitectura de experiencias

Introducir la base tecnica de `data-theme`, `data-layout`, `data-nav`,
`data-preset`, tokens y estado `custom`. Mantener compatibilidad con los valores
actuales de `leotime.theme` y `leotime.layout` durante la migracion.

### Sprint 3: Selector de experiencia

Agregar controles separados para theme, layout y nav, mas presets sugeridos.
Exponerlos en toolbar/settings y persistirlos localmente en todos los casos.
Sincronizar con perfil los campos que ya existan en la API. Si el plan de
implementacion decide persistir `nav` y `preset` en perfil, debe agregar la
migracion, API, tests y documentacion correspondientes en el mismo sprint. Si un
control se cambia manualmente, marcar como `Personalizado`.

### Sprint 4: Shell y navegacion

Refactorizar la navegacion en componentes enfocados. Mejorar sidebar completa,
sidebar compacta, topbar y bottom nav mobile/tablet sin romper rutas ni acceso a
timer, settings, idioma, offline status o logout.

### Sprint 5: Timer y captura rapida

Redisenar el timer como pieza central del producto. Mejorar start/stop, edicion
de hora de inicio, proyecto, tarea, tags, billable y manual entry en desktop,
tablet y mobile.

### Sprint 6: Timesheet

Redisenar la lista semanal, filas, grupos por dia, edicion inline, seleccion,
acciones rapidas, estados vacios, loading y errores. El resultado debe ser mas
escaneable y mas claro en mobile.

### Sprint 7: Calendar y Dashboard

Mejorar legibilidad del calendario mensual, detalle de dia, navegacion temporal,
dashboard cards, heatmap, barras semanales y project breakdown. El dashboard
debe ser util para inspeccionar actividad, no solo decorativo.

### Sprint 8: Reports e Invoices

Pulir formularios, tablas, previews, acciones de exportacion, estados de factura
y lectura de totales. Estas pantallas deben sentirse profesionales y confiables
para facturacion.

### Sprint 9: Preset Pack Inicial

Pulir y verificar los seis presets: Workbench Pro, Calm Light, Focus Dark,
Compact Power, Mobile Flow y SolidTime Exact. Cada preset debe tener desktop,
tablet y mobile revisados.

### Sprint 10: QA visual y documentacion

Crear screenshots comparativos, checklist responsive, chequeos de accesibilidad
basica, docs de diseno y guia para agregar futuros presets. Documentar como
actualizar SolidTime Exact contra un nuevo release.

## Verificacion esperada por sprint

Cada sprint debe terminar con algo visible y verificable. Segun el alcance, usar
la menor verificacion relevante durante el trabajo y cerrar cambios de codigo con
`make pre-commit`. Para cambios de UI con comportamiento responsive, agregar
capturas Playwright o pasos manuales reproducibles para desktop, tablet y mobile.

Antes de cerrar una entrega grande de UI, ejecutar:

```bash
make test
make build-web
make smoke
```

Si el cambio modifica despliegue, comandos o expectativas operativas, tambien
ejecutar `make deploy-check` y actualizar documentacion.

## Riesgos y decisiones

- Riesgo: multiplicar CSS por preset. Mitigacion: tokens semanticos y component
  tokens antes de overrides especificos.
- Riesgo: mobile quede como adaptacion secundaria. Mitigacion: Mobile Flow y QA
  responsive desde el primer sprint visual.
- Riesgo: SolidTime Exact se vuelva ambiguo. Mitigacion: versionar release,
  commit, fecha de verificacion y fuentes.
- Riesgo: estetica sobre productividad. Mitigacion: timer, timesheet, reporting
  e invoicing siguen siendo las pantallas de referencia para aprobar cambios.
- Decision: empezar con cinco presets propios muy pulidos mas SolidTime Exact,
  no con 8-10 presets experimentales.
- Decision: enfoque hibrido. Primero arquitectura minima, luego avance visible
  por pantallas y presets.

## Fuentes de referencia

- SolidTime repository: https://github.com/solidtime-io/solidtime
- SolidTime release v0.15.1: https://github.com/solidtime-io/solidtime/releases/tag/v0.15.1
- SolidTime public visual reference: https://www.solidtime.io/
- Existing leotime Solidtime-like theme doc: `docs/15-solidtime-theme.md`
- Existing theme selector doc: `docs/25-theme-selector.md`
