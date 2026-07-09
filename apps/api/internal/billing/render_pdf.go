package billing

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jung-kurt/gofpdf"
)

type PDFRenderer struct{}

func (r *PDFRenderer) RenderPreviewHTML(ctx context.Context, snapshot DocumentSnapshot) ([]byte, error) {
	return NewHTMLRenderer().RenderPreviewHTML(ctx, snapshot)
}

func (r *PDFRenderer) RenderPDFs(ctx context.Context, snapshot DocumentSnapshot, outputDir string) (RenderedPDFs, error) {
	_ = ctx
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return RenderedPDFs{}, fmt.Errorf("create pdf output dir: %w", err)
	}

	invoicePath := filepath.Join(outputDir, "invoice.pdf")
	if err := r.renderInvoicePDF(snapshot, invoicePath); err != nil {
		return RenderedPDFs{}, err
	}

	protocolPath := filepath.Join(outputDir, "work-protocol.pdf")
	if err := r.renderWorkProtocolPDF(snapshot, protocolPath); err != nil {
		return RenderedPDFs{}, err
	}

	return RenderedPDFs{
		InvoicePath:      invoicePath,
		WorkProtocolPath: protocolPath,
	}, nil
}

func (r *PDFRenderer) renderInvoicePDF(snapshot DocumentSnapshot, targetPath string) error {
	pdf := gofpdf.New("P", "mm", "Letter", "")
	pdf.SetMargins(15, 15, 15)
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 18)
	pdf.Cell(100, 10, fmt.Sprintf("Invoice # %s", snapshot.Invoice.Number))
	pdf.SetFont("Helvetica", "", 10)
	pdf.SetXY(120, 15)
	pdf.MultiCell(75, 5, strings.Join(filterEmpty([]string{
		snapshot.Invoice.SellerName,
		snapshot.Invoice.SellerTaxID,
		snapshot.Invoice.SellerAddress,
	}), "\n"), "", "R", false)

	pdf.SetXY(15, 35)
	pdf.SetFont("Helvetica", "", 10)
	pdf.MultiCell(180, 5, strings.Join(filterEmpty([]string{
		snapshot.Invoice.ClientName,
		snapshot.Invoice.ClientTaxID,
		snapshot.Invoice.ClientAddress,
	}), "\n"), "", "L", false)

	pdf.SetY(60)
	pdf.SetFont("Helvetica", "B", 9)
	pdf.CellFormat(90, 7, "Description", "1", 0, "L", false, 0, "")
	pdf.CellFormat(30, 7, "Rate Hour", "1", 0, "R", false, 0, "")
	pdf.CellFormat(30, 7, "Qty", "1", 0, "R", false, 0, "")
	pdf.CellFormat(40, 7, "Amount", "1", 1, "R", false, 0, "")

	pdf.SetFont("Helvetica", "", 9)
	for _, line := range snapshot.Invoice.Lines {
		pdf.CellFormat(90, 7, truncate(line.Description, 60), "1", 0, "L", false, 0, "")
		pdf.CellFormat(30, 7, formatMoneyMinor(line.UnitRateMinor, snapshot.Invoice.Currency), "1", 0, "R", false, 0, "")
		pdf.CellFormat(30, 7, fmt.Sprintf("%.2f", float64(line.QuantityMinutes)/60), "1", 0, "R", false, 0, "")
		pdf.CellFormat(40, 7, formatMoneyMinor(line.SubtotalMinor, snapshot.Invoice.Currency), "1", 1, "R", false, 0, "")
	}

	pdf.Ln(4)
	pdf.CellFormat(150, 7, "Total", "0", 0, "R", false, 0, "")
	pdf.CellFormat(40, 7, formatMoneyMinor(snapshot.Invoice.TotalMinor, snapshot.Invoice.Currency), "0", 1, "R", false, 0, "")

	if snapshot.Invoice.PaymentInstructions != "" {
		pdf.Ln(6)
		pdf.MultiCell(180, 5, "Payment instructions:\n"+snapshot.Invoice.PaymentInstructions, "", "L", false)
	}

	if snapshot.Invoice.Preview {
		pdf.Ln(6)
		pdf.SetTextColor(170, 0, 0)
		pdf.Cell(0, 8, "DRAFT PREVIEW")
	}

	return pdf.OutputFileAndClose(targetPath)
}

func (r *PDFRenderer) renderWorkProtocolPDF(snapshot DocumentSnapshot, targetPath string) error {
	pdf := gofpdf.New("P", "mm", "Letter", "")
	pdf.SetMargins(15, 15, 15)
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 18)
	pdf.Cell(0, 10, fmt.Sprintf("Work Protocol # %s", snapshot.WorkProtocol.Number))
	pdf.Ln(12)

	pdf.SetFont("Helvetica", "B", 9)
	pdf.CellFormat(35, 7, "Date", "1", 0, "L", false, 0, "")
	pdf.CellFormat(25, 7, "Qty", "1", 0, "R", false, 0, "")
	pdf.CellFormat(130, 7, "Tasks", "1", 1, "L", false, 0, "")

	pdf.SetFont("Helvetica", "", 9)
	for _, row := range snapshot.WorkProtocol.Rows {
		taskText := row.ProjectNames
		if len(row.Items) > 0 {
			taskText = strings.Join(row.Items, "\n")
		}
		pdf.CellFormat(35, 7, row.Date, "1", 0, "L", false, 0, "")
		pdf.CellFormat(25, 7, row.Hours, "1", 0, "R", false, 0, "")
		pdf.MultiCell(130, 5, taskText, "1", "L", false)
	}

	return pdf.OutputFileAndClose(targetPath)
}

func filterEmpty(values []string) []string {
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			filtered = append(filtered, value)
		}
	}
	return filtered
}

func truncate(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) <= max {
		return value
	}
	return value[:max-3] + "..."
}
