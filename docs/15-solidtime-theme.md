# Solidtime-like UI Theme

Este documento define el look & feel base de `leotime` para el layout `solid`. La referencia visual es una interfaz de tracking densa, oscura y operativa: la pantalla principal debe sentirse como una herramienta diaria de trabajo, no como una landing ni como un dashboard decorativo.

## Principios visuales

- La app abre en una vista de tiempo utilizable: sidebar, timer activo y lista de entradas.
- La densidad es alta pero legible. Las filas de trabajo rondan 44-52 px de alto en escritorio.
- Las superficies son oscuras, con separadores finos y contraste moderado.
- No se usan imagenes decorativas. La identidad visual sale de datos, iconos, avatares de iniciales, puntos de color y estados.
- Los botones principales son compactos. Las acciones repetidas usan iconos circulares o iconos simples con tooltip.

## Paleta

- Fondo global: casi negro, `#0c0d10`.
- Sidebar: negro profundo, `#08090b`.
- Superficie principal: `#0f1014`.
- Paneles y filas activas: `#121318`, `#17181e`, `#1d1e26`.
- Bordes: `#24262e` y `#30323b`.
- Texto principal: `#e7e7eb`.
- Texto secundario: `#a3a4ad`.
- Texto tenue e iconos secundarios: `#6f717b`.
- Accento azul de reproducir/billable: `#5fb3d9`.
- Stop/error: `#e36a6a`.
- Dots de proyecto: colores saturados pequenos, por ejemplo RTVE `#ff714b`, ENACT `#45aaf2`, Atempora `#ffb02e`.

## Tipografia

- Familia: Inter con fallback system UI.
- Titulo de vista: 15-16 px, peso 800.
- Descripciones de entradas: 14 px, peso 800, una linea con ellipsis.
- Labels, grupos y estados: 12-13 px, peso 750-850.
- Relojes, duraciones y rangos horarios: tabular nums, peso 750-850.
- No usar tamaños hero dentro de la app. Esta UI prioriza escaneo rapido.

## Layout

### Sidebar

La sidebar mide aproximadamente 272 px en escritorio y queda fija. Orden esperado:

1. Selector de organizacion con avatar inicial, nombre y chevron.
2. Current Timer con tiempo grande y boton circular de stop.
3. Navegacion principal: Dashboard, Time activo, Reporting con hijos.
4. Grupo Manage: Projects, Clients, Members, Tags.
5. Grupo Admin: Import / Export, Settings.
6. Footer con idioma, Profile Settings e iniciales del usuario.

Los items activos usan una superficie apenas mas clara, no gradientes ni colores fuertes.

### Timer superior

La fila superior del tracker tiene:

1. Descripcion larga en un campo oscuro de una linea.
2. Badge de proyecto/cliente como punto de color + texto.
3. Icono de tag.
4. Icono de facturable.
5. Reloj activo en formato `HH:MM:SS`.
6. Boton circular rojo de stop.
7. Boton `Manual time entry`.

El timer debe ser la pieza visual mas clara de la pantalla, pero sin hero ni tarjeta grande.

### Lista de tiempo

La lista se agrupa por dia:

- Header de dia: icono calendario, weekday, fecha y total diario a la derecha.
- Fila: checkbox, contador opcional, descripcion, badge de proyecto, iconos de tag/billable, rango horario, duracion, play circular y menu.
- La fila seleccionada tiene fondo un poco mas claro y borde interno.
- Los badges no son pills grandes. Son punto de color + texto, alineados con la fila.
- El estado billable activo usa azul. Los iconos inactivos quedan grises.

## Formularios y gestion

Clientes y proyectos deben conservar el mismo lenguaje oscuro:

- Paneles sin gradientes decorativos.
- Inputs oscuros con borde fino.
- Validaciones inline visibles y accesibles.
- Estados activos como chips pequenos.
- Formularios compactos, con labels claros y botones alineados a la derecha en escritorio.

## Responsive

- En tablet/mobile la sidebar pasa arriba y la navegacion se compacta.
- El timer activo se apila sin perder jerarquia.
- Las filas de tiempo pasan a layout de varias lineas, manteniendo descripcion, proyecto, rango, duracion y acciones visibles.
- Ningun texto debe salirse de botones, badges o filas; usar ellipsis donde una entrada pueda ser larga.

## No hacer

- No usar landing pages, heroes, cards promocionales ni imagenes stock para la app autenticada.
- No usar gradientes ornamentales, orbs, fondos ilustrados o paletas de un solo tono.
- No esconder los datos importantes detras de tarjetas grandes.
- No introducir iconos custom si Lucide tiene un equivalente.
