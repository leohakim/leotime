# Tutorial para contribuidores (desde Django/Python)

Guía paso a paso para tu **primer cambio real** en leotime si vienes de **Django/Python** y Go/React te resultan nuevos.

No asume que ya sabes Go. Sí asume que entiendes HTTP, SQL básico, tests y el flujo habitual de Django: modelo → vista → URL → plantilla/JSON.

## Qué vas a conseguir

Al terminar este tutorial deberías poder:

1. Clonar el repo, levantar la app y cargar datos de demo.
2. Seguir una petición HTTP desde el router hasta SQLite y vuelta.
3. Leer tests de Go sin perderte en la sintaxis.
4. Implementar un endpoint pequeño siguiendo las convenciones del proyecto.
5. Pasar `make pre-commit` y proponer un commit con mensaje claro.

**Tiempo estimado:** 2–4 horas la primera vez (más si instalas herramientas desde cero).

## Mapa rápido: Django → leotime

| En Django | En leotime | Archivo típico |
| --- | --- | --- |
| `manage.py` | Binario CLI + arranque del servidor | `apps/api/cmd/leotime/` |
| `settings.py` | Config desde variables de entorno | `apps/api/internal/config/` |
| `urls.py` | Router HTTP | `apps/api/internal/httpapi/router.go` |
| Vista / DRF view | Handler | `apps/api/internal/httpapi/*.go` |
| `models.Model` + ORM | Struct + SQL explícito | `apps/api/internal/store/` |
| Migraciones Django | SQL embebido numerado | `apps/api/internal/db/migrations/` |
| `TestCase` + cliente | `testing.T` + `httptest` | `*_test.go` junto al código |
| Plantillas / React en Django | SPA React + Vite | `apps/web/src/` |
| `gettext` / i18n | `apps/web/src/lib/i18n.ts` | claves ES/EN |

**Diferencia clave:** en leotime casi no hay “magia”. No hay ORM que infiera queries, no hay auto-reload del servidor Go, y las migraciones son archivos `.sql` que lees tal cual.

---

## Parte 0 — Prerrequisitos

### Herramientas

| Herramienta | Versión orientativa | Para qué |
| --- | --- | --- |
| Git | cualquiera reciente | clonar y proponer cambios |
| Go | la del `apps/api/go.mod` | backend y tests |
| Node.js | la del `.nvmrc` o CI (25) | frontend y Playwright |
| Docker | opcional pero recomendado | stack completo sin pelear con puertos |
| Editor | VS Code / Cursor / GoLand | con soporte Go ayuda mucho |

Comprueba:

```bash
go version
node --version
npm --version
docker --version   # opcional
```

### Conocimientos que ya tienes si usas Django

- Request/response HTTP, códigos de estado, JSON.
- Concepto de migración y restricciones SQL.
- Tests que levantan DB temporal o usan fixtures.
- Separar “lógica de negocio” de “capa HTTP” (aunque en Django a veces se mezcle).

### Conocimientos que **no** necesitas al empezar

- Goroutines avanzadas, canales, reflection.
- Dominar todo el ecosistema React.
- Escribir SQL complejo sin mirar ejemplos del repo.

---

## Parte 1 — Setup del entorno (30–45 min)

### 1.1 Clonar e instalar

```bash
git clone <url-del-repo> leotime
cd leotime
make setup
```

`make setup` hace:

- `npm install` (monorepo: API empaquetada aparte, web en workspace).
- Instala el hook `pre-commit` en `.githooks/` (cada commit ejecuta el gate de calidad).

### 1.2 Arrancar en desarrollo (dos terminales o `make dev`)

**Opción A — Todo local (rápido para iterar en Go):**

```bash
make dev
```

Levanta:

- API Go en `http://127.0.0.1:8080`
- Vite en `http://127.0.0.1:5173` (proxy `/api` → backend)

**Opción B — Docker (más parecido a producción):**

```bash
docker compose up --build
```

Abre `http://127.0.0.1:8080`.

### 1.3 Credenciales locales por defecto

```text
Email:    admin@example.com
Password: change-me-now
```

Cámbialas antes de exponer la app a internet.

### 1.4 Cargar datos de demo

```bash
make seed
```

Crea clientes, proyectos, tareas, etiquetas (tags), entradas de tiempo y un timer abierto. Es idempotente: si ya hay clientes, **se salta** sin romper nada.

Para una línea temporal fija (útil con capturas UI):

```bash
LEOTIME_SEED_NOW=2026-07-11T12:00:00Z make seed
```

### 1.5 Comprobar que todo vive

```bash
curl -fsS http://127.0.0.1:8080/api/health
make smoke BASE_URL=http://127.0.0.1:8080
```

Si `make smoke` falla, revisa que el puerto 8080 esté libre o usa Docker.

### 1.6 Instalar hooks (si no corriste `make setup`)

```bash
make setup-hooks
```

A partir de aquí, **cada `git commit` ejecuta `make pre-commit`**. Si falla, el commit se rechaza hasta que arregles el problema.

---

## Parte 2 — Tour del repositorio (20 min)

```text
.
├── apps/
│   ├── api/                 # Backend Go
│   │   ├── cmd/leotime/     # main, subcomandos seed/import/backup
│   │   └── internal/
│   │       ├── httpapi/     # Rutas y handlers (= vistas)
│   │       ├── store/       # Acceso a datos (= services + queries)
│   │       ├── db/          # SQLite + migraciones
│   │       └── ...
│   └── web/                 # Frontend React
│       ├── src/
│       │   ├── lib/         # API client, paneles UI, i18n
│       │   └── features/    # Shell, dominios más grandes
│       └── e2e/             # Playwright
├── docs/                    # Producto, ops, ADRs, este tutorial
├── Makefile                 # Comandos que debes memorizar
└── docker-compose.yml
```

**Regla de oro del backend:** el handler **no** debería contener SQL largo. Llama a `store`. La validación de negocio vive en `store` o en paquetes de dominio (`billing`, `solidtimeimport`, etc.).

**Regla de oro del frontend:** `src/lib/api.ts` centraliza fetch; los paneles viven en `*Ui.tsx` o `features/`.

Lecturas complementarias (no hace falta leerlas todas hoy):

- [Development workflow](08-development-workflow.md)
- [Go architecture](02-architecture-go.md)
- [Testing strategy](05-testing-strategy.md)
- [Operations](10-operations.md)

---

## Parte 3 — Go para alguien que viene de Python (45 min)

Esta sección es **solo lectura**. Vuelve aquí cuando veas sintaxis rara.

### 3.1 Paquetes e imports

```go
package store   // nombre del directorio; como un módulo Python

import (
    "context"
    "errors"
    "github.com/leotime/leotime/apps/api/internal/db"
)
```

- Un directorio = un paquete.
- Lo exportado empieza en **mayúscula** (`Tag`, `CreateTag`). Lo privado en minúscula (`normalizeTagInput`).

### 3.2 Structs = dataclasses / modelos sin ORM

```go
type Tag struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Color string `json:"color"`
}
```

Los tags `` `json:"id"` `` son como `serializers` de DRF: controlan el JSON.

### 3.3 Métodos con receptor (= métodos de instancia)

```go
func (s *Store) CreateTag(ctx context.Context, userID string, input TagInput) (*Tag, error) {
    // ...
}
```

- `s *Store` es el receptor (como `self` en Python, pero tipado).
- Casi todas las funciones de DB reciben `context.Context` primero: cancelación y timeouts.

### 3.4 Errores: no hay try/except

```go
tag, err := s.store.CreateTag(ctx, userID, input)
if err != nil {
    return err
}
```

Patrones del repo:

- Errores sentinela: `var ErrTagNotFound = errors.New("tag not found")`.
- Comprobar con `errors.Is(err, store.ErrTagNotFound)`.
- Envolver con contexto: `fmt.Errorf("create tag: %w", err)`.

En Django a veces lanzas `ValidationError`. Aquí `store` devuelve `*ValidationError` o errores sentinela y `httpapi` los traduce a JSON `{ "error": { "code", "message", "fields?" } }`.

### 3.5 Interfaces (solo lo que necesitas ahora)

Go usa interfaces implícitas. En tests verás mocks pequeños, pero **en leotime los tests de DB usan SQLite real en un archivo temporal**, no mocks del ORM. Eso es parecido a `pytest-django` con `db` fixture.

### 3.6 Tests

```go
func TestTagLifecycle(t *testing.T) {
    ctx := context.Background()
    st, user := newTagTestStore(t, ctx)

    tag, err := st.CreateTag(ctx, user.ID, TagInput{Name: "Deep Work", Color: "#2563eb"})
    if err != nil {
        t.Fatalf("create tag: %v", err)
    }
    // ...
}
```

Equivalencias:

| Go | Python/pytest |
| --- | --- |
| `func TestX(t *testing.T)` | `def test_x():` |
| `t.Fatalf(...)` | `assert False, ...` / `pytest.fail` |
| `t.Helper()` | marcar helper de test |
| `httptest.NewRecorder()` | `client.get()` de Django test client |

Ejecutar solo tests de tags:

```bash
cd apps/api
go test ./internal/store -run TestTag -v
```

### 3.7 Formateo obligatorio

Antes de commit:

```bash
gofmt -w apps/api/internal/store/tag.go
```

`make pre-commit` falla si algún `.go` no está formateado.

### 3.8 Punteros (`*` y `&`) — la parte que más asusta

Regla práctica en este repo:

- `*Tag` = “puede ser nil”; útil para “no encontrado”.
- `&Tag{...}` = crear struct y pasar puntero.
- No hace falta dominar punteros el día 1; copia patrones de archivos vecinos.

---

## Parte 4 — Seguir una petición real: listar tags (30 min)

Vamos a trazar `GET /api/v1/tags` como harías con el depurador de Django.

### 4.1 Ruta registrada

En `apps/api/internal/httpapi/router.go`:

```go
r.Get("/tags", server.requireUser(server.listTags))
```

- `requireUser` = decorador que exige sesión (cookie HTTP-only).
- Equivalente Django: `@login_required` + vista.

### 4.2 Handler

En `apps/api/internal/httpapi/tags.go`, `listTags`:

1. Lee query `includeArchived`.
2. Llama `s.store.ListTags(ctx, user.ID, includeArchived)`.
3. Si error → `writeError` con código estable.
4. Si ok → `writeJSON` con `{ "tags": [...] }`.

**No hay serializer class:** el struct `store.Tag` ya lleva tags JSON.

### 4.3 Store

En `apps/api/internal/store/tag.go`, `ListTags` ejecuta SQL explícito:

```sql
SELECT id, name, color, archived_at, created_at, updated_at
FROM tags
WHERE user_id = ?
```

En Django escribirías `Tag.objects.filter(user=request.user)`. Aquí ves la query completa.

### 4.4 Probar desde terminal

Con sesión (después de login en el navegador es más fácil; para API pura usa el test HTTP del repo como referencia).

El test `TestTagHTTPLifecycle` en `router_test.go` muestra el patrón completo: login → cookie → POST /tags → PATCH → GET → DELETE → restore.

### 4.5 Test de integración HTTP

Ubicación: `apps/api/internal/httpapi/router_test.go`, función `TestTagHTTPLifecycle`.

Es el equivalente a:

```python
def test_tag_crud(self):
    self.client.login(...)
    res = self.client.post("/api/v1/tags", {...}, content_type="application/json")
    assert res.status_code == 201
```

---

## Parte 5 — Ejercicio guiado: resumen de tags (90–120 min)

Este es el **“first issue”** recomendado del tutorial: añadir un endpoint de solo lectura que devuelva cuántas etiquetas activas y archivadas tiene el usuario.

**Producto:** `GET /api/v1/tags/summary` → `{ "active": 3, "archived": 1 }`

**Por qué este ejercicio:** no toca migraciones, practica store + handler + router + tests, y es fácil de revisar en un PR.

> Si ya existe en el repo cuando leas esto, el mantenedor te asignará otro issue pequeño con la misma estructura de capas.

### 5.1 Crear rama

```bash
git checkout -b feat/tags-summary-endpoint
```

### 5.2 Paso A — Store (`apps/api/internal/store/tag.go`)

Añade un struct de respuesta y un método:

```go
type TagSummary struct {
    Active   int `json:"active"`
    Archived int `json:"archived"`
}

func (s *Store) TagSummary(ctx context.Context, userID string) (*TagSummary, error) {
    const query = `
        SELECT
            SUM(CASE WHEN archived_at IS NULL THEN 1 ELSE 0 END) AS active_count,
            SUM(CASE WHEN archived_at IS NOT NULL THEN 1 ELSE 0 END) AS archived_count
        FROM tags
        WHERE user_id = ?
    `
    var active, archived int
    if err := s.db.QueryRowContext(ctx, query, userID).Scan(&active, &archived); err != nil {
        return nil, fmt.Errorf("tag summary: %w", err)
    }
    return &TagSummary{Active: active, Archived: archived}, nil
}
```

**Checklist store:**

- [ ] SQL con `user_id = ?` (nunca concatenar IDs en la query).
- [ ] Errores envueltos con contexto (`fmt.Errorf("...: %w", err)`).
- [ ] Struct exportado con tags JSON.

### 5.3 Paso B — Test de store (`apps/api/internal/store/tag_test.go`)

Añade:

```go
func TestTagSummaryCountsActiveAndArchived(t *testing.T) {
    ctx := context.Background()
    st, user := newTagTestStore(t, ctx)

    tag, err := st.CreateTag(ctx, user.ID, TagInput{Name: "Deep Work", Color: "#2563eb"})
    if err != nil {
        t.Fatalf("create tag: %v", err)
    }
    if err := st.ArchiveTag(ctx, user.ID, tag.ID); err != nil {
        t.Fatalf("archive tag: %v", err)
    }
    if _, err := st.CreateTag(ctx, user.ID, TagInput{Name: "Focus", Color: "#0f7a5b"}); err != nil {
        t.Fatalf("create second tag: %v", err)
    }

    summary, err := st.TagSummary(ctx, user.ID)
    if err != nil {
        t.Fatalf("tag summary: %v", err)
    }
    if summary.Active != 1 || summary.Archived != 1 {
        t.Fatalf("unexpected summary: %+v", summary)
    }
}
```

Ejecuta:

```bash
cd apps/api
go test ./internal/store -run TestTagSummary -v
```

**Si falla:** lee el mensaje completo. Los fallos de constraint SQL suelen indicar migración o query mal escrita.

### 5.4 Paso C — Handler (`apps/api/internal/httpapi/tags.go`)

```go
func (s *Server) tagSummary(w http.ResponseWriter, r *http.Request, user *store.User) {
    summary, err := s.store.TagSummary(r.Context(), user.ID)
    if err != nil {
        writeError(w, http.StatusInternalServerError, "tags_summary_failed", "load tag summary failed")
        return
    }
    writeJSON(w, http.StatusOK, summary)
}
```

Patrón igual que `listTags`: poco código, errores con **código estable** en inglés (`tags_summary_failed`).

### 5.5 Paso D — Ruta (`router.go`)

Junto al bloque de tags:

```go
r.Get("/tags/summary", server.requireUser(server.tagSummary))
```

**Orden importante:** en routers tipo Chi, rutas más específicas (`/tags/summary`) deben registrarse **antes** de `/tags/{tagID}` si algún día hubiera conflicto. Hoy `/tags/summary` no choca con `{tagID}` porque el patrón es distinto, pero es buen hábito.

### 5.6 Paso E — Test HTTP (`router_test.go`)

Añade algo como:

```go
func TestTagSummaryRequiresAuthentication(t *testing.T) {
    router := newTestRouter(t)

    response := httptest.NewRecorder()
    request := httptest.NewRequest(http.MethodGet, "/api/v1/tags/summary", nil)
    router.ServeHTTP(response, request)

    if response.Code != http.StatusUnauthorized {
        t.Fatalf("expected 401, got %d", response.Code)
    }
}

func TestTagSummaryReturnsCounts(t *testing.T) {
    router := newTestRouter(t)
    cookies := loginCookies(t, router)

    // crea y archiva una tag con los helpers del test existente...

    response := httptest.NewRecorder()
    request := httptest.NewRequest(http.MethodGet, "/api/v1/tags/summary", nil)
    for _, cookie := range cookies {
        request.AddCookie(cookie)
    }
    router.ServeHTTP(response, request)

    if response.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
    }
    // decodifica y comprueba active/archived
}
```

Copia estilo de `TestTagHTTPLifecycle` para login y cookies.

Ejecuta:

```bash
cd apps/api
go test ./internal/httpapi -run TestTagSummary -v
```

### 5.7 Paso F — Reiniciar el servidor

Go **no** recarga solo. Si tenías `go run` o Docker, reinicia el proceso después de cambiar handlers.

```bash
# Ctrl+C en make dev, luego:
make dev
```

Prueba manual (con cookie de sesión del navegador o extensión REST):

```http
GET /api/v1/tags/summary
```

### 5.8 Paso G — Documentación API (opcional pero apreciado)

Si el proyecto tiene doc del dominio tags, añade una línea. Si no, una nota en el cuerpo del PR basta para endpoints pequeños.

### 5.9 Paso H — Gate completo

```bash
make pre-commit
```

Desglose:

| Paso | Qué hace |
| --- | --- |
| `fmt-check` | `gofmt` en todo `apps/api` |
| `test-api-vet` | análisis estático `go vet` |
| `test-api` | `go test ./...` |
| `test-web` | Vitest |
| `build-web` | `tsc` + build Vite |

Si algo falla, **arregla solo lo que tu cambio rompió** más cualquier error que aparezca; no hagas refactors grandes en el mismo PR.

---

## Parte 6 — Frontend opcional (si quieres cerrar el circuito)

Solo después de que el backend funcione. Equivalente Django: añadir campo en template que consume nueva API.

1. **`apps/web/src/lib/api.ts`** — función `fetchTagSummary()`.
2. **Panel de tags** — mostrar “Activas: X / Archivadas: Y” bajo el título.
3. **`apps/web/src/lib/i18n.ts`** — claves ES/EN.
4. **Test Vitest** — render con mock de fetch (mira `App.test.tsx`).

No hace falta Playwright para este PR pequeño.

---

## Parte 7 — Commits y pull request

### Mensaje de commit (Conventional Commits)

```text
feat(api): add tag summary endpoint with store and HTTP tests

Expose GET /api/v1/tags/summary for active/archived counts so the
tags panel can show inventory without loading the full list.
```

### Cuerpo del PR

Incluye:

- **Qué** cambia para el usuario (aunque sea solo API).
- **Cómo probar** (comandos `go test`, curl o pasos UI).
- **Riesgos** (ninguno para lectura; permisos = usuario autenticado).

### Qué revisará un maintainer

- ¿SQL acotado por `user_id`?
- ¿Tests de store **y** HTTP?
- ¿Códigos de error estables?
- ¿`make pre-commit` verde?

---

## Parte 8 — Errores frecuentes (y cómo leerlos)

### `undefined: Store.TagSummary`

Olvidaste exportar el struct (mayúscula) o el import del paquete `store` en el handler.

### `404 not found` en ruta nueva

- Servidor viejo sin reiniciar.
- Ruta mal escrita o registrada fuera de `/api/v1`.
- Proxy de Vite apuntando a otro puerto.

### `go test` pasa pero el navegador falla

- Frontend aún no llama al endpoint (normal si solo hiciste backend).
- Cookie de sesión ausente (401).

### `make pre-commit` falla en gofmt

```bash
gofmt -w apps/api/ruta/al/archivo.go
```

### `database is locked` en tests

SQLite concurrente en tu máquina; re-ejecuta tests. Si persiste, cierra procesos `leotime` duplicados.

### Miedo a “romper producción”

- Los tests usan DB temporal en `t.TempDir()`.
- `make seed` no borra datos existentes.
- Usa siempre ramas y PRs.

---

## Parte 9 — Comandos que usarás cada día

```bash
make help              # lista todo
make dev               # API + Vite
make test-api          # solo Go
make test-web          # solo Vitest
make test-e2e          # Playwright smoke
make pre-commit        # gate antes de commit
make seed              # demo data
make smoke             # salud HTTP rápida
```

Explorar un paquete:

```bash
cd apps/api
go test ./internal/store -run TestTag -v
go test ./internal/httpapi -run TestTagHTTPLifecycle -v
```

---

## Parte 10 — Siguientes pasos según tu perfil

| Si te interesa… | Siguiente lectura / tarea |
| --- | --- |
| Más backend Go | [Go architecture](02-architecture-go.md), [Data model](03-data-model.md) |
| Facturación / PDFs | [Billing documents](32-billing-documents.md), ADR 0004 |
| Import Solidtime | [Solidtime import](09-solidtime-import.md) |
| UI / UX | [Theme selector](25-theme-selector.md), [Visual regression](39-visual-regression.md) |
| Operaciones | [Operations](10-operations.md), [VPS deploy](06-deploy-vps.md) |
| Issues pequeños reales | [Backlog](13-backlog.md), [Curated hardening](35-curated-hardening-backlog.md) |

---

## Checklist final del contribuidor

- [ ] `make setup` y app accesible en local
- [ ] `make seed` y login correcto
- [ ] Leído el flujo tags: router → handler → store → SQL
- [ ] Completado el ejercicio `tags/summary` (o issue equivalente)
- [ ] Tests store + HTTP en verde
- [ ] `make pre-commit` en verde
- [ ] PR con descripción de prueba y commit Conventional Commits

Bienvenido al repo. Si te atascas, abre un PR en borrador o un issue con el comando exacto que falló y el log completo: en Go el mensaje de error suele ser suficiente si se copia entero.
