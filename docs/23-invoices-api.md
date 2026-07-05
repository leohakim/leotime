# Invoices API

Las facturas permiten crear borradores desde tiempo facturable, congelar datos del cliente y exportar HTML imprimible, CSV o JSON.

## Rutas HTTP

Todas las rutas requieren cookie de sesion valida.

```text
GET    /api/v1/invoices
POST   /api/v1/invoices/draft-from-time
GET    /api/v1/invoices/{invoiceID}
PATCH  /api/v1/invoices/{invoiceID}
POST   /api/v1/invoices/{invoiceID}/status
DELETE /api/v1/invoices/{invoiceID}
GET    /api/v1/invoices/{invoiceID}/export
```

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
  "dueAt": "2026-08-15T00:00:00Z"
}
```

| Campo | Regla |
| --- | --- |
| `clientId` | Obligatorio. Cliente cuyo tiempo facturable se incluira. |
| `from`, `to` | Obligatorios, RFC3339. Filtran por `started_at`. |
| `taxRateBasisPoints` | IVA u otro impuesto por linea. `2100` = 21%. Default `0`. |
| `withholdingMinor` | Retencion IRPF u otra en unidades menores. Default `0`. |
| `sellerName` | Opcional. Default: nombre del usuario autenticado. |
| Entradas | Solo tiempo facturable, finalizado y aun no incluido en otra factura no anulada. |

Cada entrada genera una linea con:

- descripcion compuesta (proyecto, tarea, texto),
- minutos redondeados desde `duration_seconds`,
- tarifa horaria resuelta (proyecto > cliente > 0),
- subtotal en unidades menores.

El numero de factura se genera como `INV-{YYYY}-{secuencia}`.

## Estados

| Estado | Regla |
| --- | --- |
| `draft` | Editable y eliminable. |
| `issued` | Congelada. `issued_at` se rellena al emitir si estaba vacio. |
| `paid` | Cobrada. |
| `cancelled` | Anulada. Las entradas vuelven a estar disponibles para facturar. |

```http
POST /api/v1/invoices/{invoiceID}/status
Content-Type: application/json

{ "status": "issued" }
```

## Exportacion

```text
GET /api/v1/invoices/{invoiceID}/export?format=html
GET /api/v1/invoices/{invoiceID}/export?format=csv
GET /api/v1/invoices/{invoiceID}/export?format=json
```

- **HTML:** documento imprimible listo para guardar como PDF desde el navegador.
- **CSV:** cabecera de totales + lineas.
- **JSON:** factura completa con lineas.

## Frontend

El panel `#invoices` permite:

- elegir cliente y rango de fechas,
- definir IVA y retencion,
- crear borrador,
- emitir, marcar pagada o anular,
- descargar HTML, CSV o JSON.

Los importes se muestran con `Intl.NumberFormat` segun moneda de la factura.

## Tests

- Store: `apps/api/internal/store/invoice_test.go`
- HTTP: `TestInvoiceDraftFromTimeAndExport` en `router_test.go`
- Web: `renders the invoice panel` en `App.test.tsx`
