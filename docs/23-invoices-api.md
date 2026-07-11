# Invoices API

Las facturas permiten crear borradores desde tiempo facturable, previsualizar el documento, emitir PDFs oficiales inmutables con serie fiscal, y exportar HTML, CSV o JSON.

## Rutas HTTP

Todas las rutas requieren cookie de sesion valida.

```text
GET    /api/v1/invoice-series
POST   /api/v1/invoice-series
PATCH  /api/v1/invoice-series/{seriesID}

GET    /api/v1/invoices
POST   /api/v1/invoices/draft-from-time
GET    /api/v1/invoices/{invoiceID}
PATCH  /api/v1/invoices/{invoiceID}
POST   /api/v1/invoices/{invoiceID}/status
DELETE /api/v1/invoices/{invoiceID}
GET    /api/v1/invoices/{invoiceID}/export

POST   /api/v1/invoices/{invoiceID}/preview
POST   /api/v1/invoices/{invoiceID}/issue
POST   /api/v1/invoices/{invoiceID}/cancel
GET    /api/v1/invoices/{invoiceID}/documents
GET    /api/v1/invoices/{invoiceID}/documents/{documentID}/download
```

## Series fiscales

Cada usuario tiene al menos una serie fiscal. El bootstrap crea `MAIN` como serie por defecto.

```http
GET /api/v1/invoice-series
```

Respuesta:

```json
{
  "series": [
    {
      "id": "ser_...",
      "code": "MAIN",
      "name": "Principal",
      "pattern": "{YYYY}-{SEQ:04}",
      "nextSequence": 1,
      "resetPolicy": "yearly",
      "active": true,
      "default": true
    }
  ]
}
```

El patron admite `{YYYY}`, `{YY}` y `{SEQ:NN}` para el siguiente numero oficial.

## Crear borrador desde tiempo

```http
POST /api/v1/invoices/draft-from-time
Content-Type: application/json

{
  "clientId": "cli_...",
  "from": "2026-07-01T00:00:00Z",
  "to": "2026-07-31T23:59:59Z",
  "taxRateBasisPoints": 2100,
  "withholdingMinor": 0,
  "sellerName": "Leonardo",
  "sellerTaxId": "12345678Z",
  "sellerAddress": "Madrid",
  "notes": "Gracias por confiar en nosotros",
  "dueAt": "2026-08-15T00:00:00Z",
  "seriesId": "ser_...",
  "workProtocolDetail": "standard"
}
```

| Campo | Regla |
| --- | --- |
| `clientId` | Obligatorio. Cliente cuyo tiempo facturable se incluira. |
| `from`, `to` | Obligatorios, RFC3339. Filtran por `started_at`. |
| `taxRateBasisPoints` | IVA u otro impuesto por linea. `2100` = 21%. Default `0`. |
| `withholdingMinor` | Retencion IRPF u otra en unidades menores. Default `0`. |
| `sellerName` | Opcional. Default: nombre del usuario autenticado. |
| `seriesId` | Opcional. Serie fiscal para la emision oficial. Default: serie marcada como `default`. |
| `workProtocolDetail` | `summary`, `standard` o `detailed`. Controla el anexo de protocolo de trabajo. |
| Entradas | Solo tiempo facturable, finalizado y aun no incluido en otra factura no anulada. |

Cada entrada genera una linea con:

- descripcion compuesta (proyecto, tarea, texto),
- minutos redondeados desde `duration_seconds`,
- tarifa horaria resuelta (proyecto > cliente > 0),
- subtotal en unidades menores.

Los borradores usan un numero provisional `DRAFT-{id}` hasta la emision oficial.

## Estados

| Estado | Regla |
| --- | --- |
| `draft` | Editable y eliminable. Permite preview sin numero fiscal. |
| `issued` | Congelada. Numero fiscal asignado, snapshot guardado, PDFs oficiales en disco. |
| `paid` | Cobrada. |
| `cancelled` | Anulada con motivo. Los PDFs emitidos siguen descargables; las entradas vuelven a estar disponibles para facturar. |

### Emision oficial

```http
POST /api/v1/invoices/{invoiceID}/issue
```

- Asigna el siguiente numero de la serie fiscal.
- Congela un snapshot JSON en la factura.
- Genera `invoice_pdf` y `work_protocol_pdf` bajo `LEOTIME_DOCUMENT_ROOT`.
- Guarda metadatos (`sha256`, tamano, ruta) en `billing_documents`.
- La factura deja de ser editable.

### Vista previa

```http
POST /api/v1/invoices/{invoiceID}/preview
```

Devuelve HTML imprimible con numero `PREVIEW-*`. No consume secuencia fiscal ni escribe PDFs.

### Anulacion

```http
POST /api/v1/invoices/{invoiceID}/cancel
Content-Type: application/json

{ "reason": "Error en el periodo facturado" }
```

### Cambio de estado legacy

```http
POST /api/v1/invoices/{invoiceID}/status
Content-Type: application/json

{ "status": "paid" }
```

`POST /status` es una ruta de compatibilidad para marcar facturas emitidas como
cobradas (`issued -> paid`). Rechaza `draft`, `issued`, `cancelled` y cualquier
otro atajo. La emision oficial solo puede hacerse con `POST /issue`; la anulacion
solo con `POST /cancel`.

## Documentos oficiales

```http
GET /api/v1/invoices/{invoiceID}/documents
GET /api/v1/invoices/{invoiceID}/documents/{documentID}/download
```

`GET /invoices/{id}` incluye `documents[]` con `downloadUrl` cuando existen PDFs emitidos.

Los archivos viven en `LEOTIME_DOCUMENT_ROOT` (default `/data/documents`). SQLite guarda solo metadatos y hashes SHA-256.

## Exportacion auxiliar

```text
GET /api/v1/invoices/{invoiceID}/export?format=html
GET /api/v1/invoices/{invoiceID}/export?format=csv
GET /api/v1/invoices/{invoiceID}/export?format=json
```

- **HTML:** documento imprimible adicional (no sustituye al PDF oficial).
- **CSV:** cabecera de totales + lineas.
- **JSON:** factura completa con lineas.

## Frontend

El panel `#invoices` permite:

- elegir cliente, serie fiscal y detalle del protocolo de trabajo,
- definir IVA y retencion,
- crear borrador,
- previsualizar, emitir oficialmente, marcar pagada o anular,
- descargar PDFs oficiales y exportar HTML, CSV o JSON.

Los importes se muestran con `Intl.NumberFormat` segun moneda de la factura.

## Tests

- Store: `apps/api/internal/store/invoice_test.go`, `invoice_series_test.go`, `invoice_documents_test.go`
- Billing: `apps/api/internal/billing/*_test.go`
- HTTP: `TestInvoiceBillingIssuePreviewAndDownload` en `router_test.go`
- Web: `renders the invoice panel` en `App.test.tsx`
