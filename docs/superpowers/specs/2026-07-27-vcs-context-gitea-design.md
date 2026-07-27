# Contexto VCS por cliente para resúmenes con IA — Gitea primero

> **Estado:** propuesto para el primer corte de la Fase 10  
> **Fecha:** 2026-07-27  
> **Depende de:** resúmenes diarios aprobables (Fase 9) y `projects.client_id`

## Objetivo

Incorporar actividad real de control de versiones al prompt del resumen diario
con IA, sin mezclar clientes ni entregar código fuente a la IA. El primer
adaptador será Gitea, con soporte explícito para instancias autoalojadas como
Osoigo. La base de datos y el contrato interno quedarán preparados para sumar
GitHub y GitLab después sin reescribir el flujo de resúmenes.

El resultado útil es un resumen de un cliente que puede decir, por ejemplo,
qué commits, pull requests, revisiones e incidencias se trabajaron ese día, pero
solo si esos hechos proceden de repositorios relacionados con ese cliente.

## Decisiones

### Alcance del primer corte

- Conexiones Gitea de solo lectura, con URL base configurable para Osoigo u
  otra instancia autoalojada.
- Repositorios Gitea relacionados con un cliente y, opcionalmente, con un
  proyecto de ese cliente.
- Consulta de commits, pull requests, revisiones y referencias de incidencias
  del día para formar un contexto factual y compacto.
- Vista de configuración para crear, probar, editar, desactivar y eliminar una
  conexión, y para enlazar sus repositorios a clientes/proyectos.
- Vista previa del contexto normalizado que recibirá el prompt antes de gastar
  créditos de Cursor.
- GitHub y GitLab no se conectan en este corte; se integrarán después a través
  del mismo contrato de proveedor.

### Modelo de relaciones

Una **conexión VCS** representa un servicio y sus credenciales: proveedor
(`gitea` inicialmente), URL base, identidad del propietario, token cifrado,
estado y cliente predeterminado opcional.

Un **repositorio VCS** pertenece a una conexión y debe resolver a un único
cliente efectivo:

1. usa su `client_id` explícito si se configura; o
2. hereda el cliente predeterminado de su conexión.

Si ninguna vía produce exactamente un cliente, el repositorio no puede
participar en un resumen. Un repositorio puede enlazarse opcionalmente a un
proyecto del mismo cliente; la validación rechaza enlaces a proyectos de otro
cliente. Una conexión Gitea compartida puede, por tanto, contener repositorios
de varios clientes sin confundir sus actividades. Una conexión Gitea dedicada
puede fijar un cliente predeterminado y reducir la configuración repetida.

La conexión no sustituye `projects.git_remote_url`: ese campo heredado sigue
siendo útil para el enricher local, pero no se usa como autorización ni como
identidad del proveedor remoto.

### Contrato interno

El backend expone un adaptador pequeño, independiente del proveedor:

```go
type VCSProvider interface {
    TestConnection(ctx context.Context, connection VCSConnection) error
    CollectDayContext(ctx context.Context, request VCSCollectionRequest) (VCSContext, error)
}
```

`VCSCollectionRequest` contiene el día, identidad del propietario, repositorios
ya filtrados por cliente/proyecto y los límites de recogida. `VCSContext`
normaliza estos hechos, sin filtrar detalles propios de Gitea al prompt:

- commits: hash corto, asunto, autor, hora, rama disponible, nombres de archivo
  y estadísticas agregadas;
- pull requests: número, título, estado, autor, etiquetas, revisores, resultado
  de merge y commits vinculados;
- revisiones: estado, revisor y pull request asociado;
- incidencias: número, título y estado, solo cuando estén explícitamente
  vinculadas a la actividad recogida.

El adaptador Gitea traduce sus respuestas HTTP a este contrato. GitHub y GitLab
implementarán el mismo contrato; no se filtrará su JSON directamente a la UI ni
al prompt.

### Flujo de enriquecimiento

1. El usuario elige fecha, cliente y proyecto opcional en el resumen diario.
2. El backend carga solo los repositorios con cliente efectivo coincidente;
   después aplica el filtro de proyecto si existe.
3. El adaptador Gitea recoge y normaliza la actividad del día para la identidad
   configurada. Deduplica commits y revisiones antes de construir el resultado.
4. El backend añade una sección `Actividad VCS verificada` al contexto de
   enriquecimiento, junto con entradas de tiempo, nota manual y contexto local
   de Cursor cuando esté disponible.
5. La UI muestra la vista previa factual; el usuario puede excluir categorías
   antes de pulsar «Enriquecer con IA».
6. El flujo existente permite editar, guardar y aprobar el texto. La IA recibe
   hechos, no permisos ni tokens.

Una caída del servicio, límite de tasa o token inválido devuelve estado de
contexto parcial. La generación de plantilla, edición y aprobación siguen
funcionando sin VCS.

## Seguridad y privacidad

- El token se cifra con el mecanismo existente de secretos, nunca se devuelve
  desde la API, se muestra solo como «configurado» y nunca se registra en logs.
- Se solicitan exclusivamente permisos de lectura de repositorios, pull
  requests, revisiones e incidencias. No se clona, escribe, aprueba, comenta ni
  instala webhooks.
- La URL de una instancia autoalojada se valida estrictamente: `https` en
  producción, host y puerto normalizados y una política de hosts permitidos. La
  conexión no puede convertirse en un proxy hacia servicios internos ajenos.
- El contexto tiene topes por repositorio y por resumen. No incluye diffs,
  contenido de archivos, discusiones completas, secretos ni texto ilimitado.
- Los errores y métricas identifican proveedor y clase de error, pero no URL
  completa, token, cuerpo de respuesta ni prompt.

## Persistencia y API

El primer corte agrega migraciones forward-only para conexiones y repositorios
VCS. Las relaciones con cliente/proyecto se validan en la capa `store`, igual
que las demás referencias de leotime.

La API autenticada permite gestionar conexiones y enlaces, probar una conexión
y obtener la vista previa de contexto para un resumen. Las respuestas públicas
contienen `tokenConfigured`, nunca el token. El endpoint de enriquecimiento
recibe solo identificadores y opciones de inclusión; el servidor resuelve las
credenciales y recopila el contexto internamente.

## Pruebas y criterios de aceptación

- Pruebas de store para herencia de cliente, cliente explícito, enlace de
  proyecto del mismo cliente y rechazo de combinaciones ambiguas o cruzadas.
- Pruebas HTTP con un servidor Gitea simulado para éxito, token inválido,
  límites de tasa y servicio no disponible. Las fixtures son sintéticas.
- Pruebas del adaptador que normalicen y dedupliquen commits, pull requests,
  revisiones e incidencias sin incluir cuerpos de respuesta sensibles.
- Pruebas del prompt que confirmen que un resumen de Cliente A no puede recibir
  hechos de Cliente B y que el contexto se puede excluir o degradar sin romper
  el borrador.
- Pruebas de UI para gestión de conexión, vínculo de repositorio, cliente,
  proyecto y vista previa.
- Pruebas end-to-end con datos sintéticos: conectar Gitea, enlazar un
  repositorio a un cliente, generar un resumen por cliente y comprobar que la
  vista previa y el prompt contienen solo su actividad.

## Fuera de alcance

- GitHub, GitLab, Bitbucket y Azure DevOps como adaptadores funcionales; el
  contrato se diseña para ellos, pero Gitea/Osoigo es el único proveedor activo
  en este corte.
- Importar diffs o archivos fuente al prompt.
- Escritura en el proveedor, webhooks, clonación o sincronización de
  repositorios.
- Inferir el cliente de una URL Git, nombre de organización o texto de commit:
  el vínculo debe ser configurado y verificable.
- Cambiar la facturación o convertir automáticamente los resúmenes en Work
  Protocols.
