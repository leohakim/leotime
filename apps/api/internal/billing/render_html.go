package billing

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"strings"
)

func (r *HTMLRenderer) RenderPreviewHTML(ctx context.Context, snapshot DocumentSnapshot) ([]byte, error) {
	_ = ctx
	var buf bytes.Buffer
	if err := documentTemplate.Execute(&buf, snapshotViewModel(snapshot)); err != nil {
		return nil, fmt.Errorf("render preview html: %w", err)
	}
	return buf.Bytes(), nil
}

func (r *HTMLRenderer) RenderPDFs(ctx context.Context, snapshot DocumentSnapshot, outputDir string) (RenderedPDFs, error) {
	return NewPDFRenderer().RenderPDFs(ctx, snapshot, outputDir)
}

type snapshotView struct {
	Snapshot DocumentSnapshot
	Invoice  InvoiceSnapshot
	Protocol WorkProtocolSnapshot
}

func snapshotViewModel(snapshot DocumentSnapshot) snapshotView {
	return snapshotView{
		Snapshot: snapshot,
		Invoice:  snapshot.Invoice,
		Protocol: snapshot.WorkProtocol,
	}
}

var documentTemplate = template.Must(template.New("billing-documents").Funcs(template.FuncMap{
	"money": formatMoneyMinor,
	"hours": func(minutes int) string {
		return fmt.Sprintf("%.2f", float64(minutes)/60)
	},
}).Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>{{.Invoice.Number}}</title>
<style>
body{font-family:Helvetica,Arial,sans-serif;color:#111;margin:40px}
header{display:flex;justify-content:space-between;gap:24px;margin-bottom:24px}
h1{margin:0;font-size:28px}
.seller{text-align:right;font-size:13px;line-height:1.5}
.meta{font-size:13px;color:#444;margin-bottom:24px}
table{width:100%;border-collapse:collapse;margin:16px 0 24px}
th,td{border:1px solid #111;padding:8px;font-size:13px;text-align:left}
th{background:#f5f5f5}
.num{text-align:right;white-space:nowrap}
.totals{margin-left:auto;width:320px;font-size:13px}
.totals div{display:flex;justify-content:space-between;padding:4px 0}
.section{margin-top:32px}
.draft{color:#a00;font-weight:bold}
</style>
</head>
<body>
<header>
  <div>
    <h1>Invoice # {{.Invoice.Number}}</h1>
    <div class="meta">Date: {{.Invoice.IssuedAt}}</div>
    {{if .Invoice.Preview}}<div class="draft">DRAFT PREVIEW</div>{{end}}
  </div>
  <div class="seller">
    <strong>{{.Invoice.SellerName}}</strong><br>
    {{.Invoice.SellerTaxID}}<br>
    {{.Invoice.SellerAddress}}
  </div>
</header>
<div class="meta">
  <strong>{{.Invoice.ClientName}}</strong><br>
  {{.Invoice.ClientTaxID}}<br>
  {{.Invoice.ClientAddress}}
</div>
<table>
  <thead>
    <tr><th>Description</th><th class="num">Rate Hour</th><th class="num">Qty</th><th class="num">Amount</th></tr>
  </thead>
  <tbody>
    {{range .Invoice.Lines}}
    <tr>
      <td>{{.Description}}</td>
      <td class="num">{{money .UnitRateMinor $.Invoice.Currency}}</td>
      <td class="num">{{hours .QuantityMinutes}}</td>
      <td class="num">{{money .SubtotalMinor $.Invoice.Currency}}</td>
    </tr>
    {{end}}
  </tbody>
</table>
<div class="totals">
  <div><span>Subtotal</span><span>{{money .Invoice.SubtotalMinor .Invoice.Currency}}</span></div>
  <div><span>Tax</span><span>{{money .Invoice.TaxMinor .Invoice.Currency}}</span></div>
  <div><strong>Total</strong><strong>{{money .Invoice.TotalMinor .Invoice.Currency}}</strong></div>
</div>
{{if .Invoice.PaymentInstructions}}<div class="meta"><strong>Payment instructions</strong><br>{{.Invoice.PaymentInstructions}}</div>{{end}}
<div class="section">
  <h2>Work Protocol # {{.Protocol.Number}}</h2>
  <table>
    <thead><tr><th>Date</th><th class="num">Qty</th><th>Tasks</th></tr></thead>
    <tbody>
      {{range .Protocol.Rows}}
      <tr>
        <td>{{.Date}}</td>
        <td class="num">{{.Hours}}</td>
        <td>
          {{if .ProjectNames}}{{.ProjectNames}}{{end}}
          {{range .Items}}<div>• {{.}}</div>{{end}}
        </td>
      </tr>
      {{end}}
    </tbody>
  </table>
</div>
</body>
</html>`))

func formatMoneyMinor(amountMinor int64, currency string) string {
	sign := ""
	if amountMinor < 0 {
		sign = "-"
		amountMinor = -amountMinor
	}
	whole := amountMinor / 100
	fraction := amountMinor % 100
	return fmt.Sprintf("%s%s %d.%02d", sign, strings.ToUpper(strings.TrimSpace(currency)), whole, fraction)
}
