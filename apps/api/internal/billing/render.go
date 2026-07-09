package billing

import "context"

const RenderVersion = "billing-documents-v1"

type RenderedPDFs struct {
	InvoicePath      string
	WorkProtocolPath string
}

type Renderer interface {
	RenderPreviewHTML(ctx context.Context, snapshot DocumentSnapshot) ([]byte, error)
	RenderPDFs(ctx context.Context, snapshot DocumentSnapshot, outputDir string) (RenderedPDFs, error)
}

type HTMLRenderer struct{}

func NewHTMLRenderer() *HTMLRenderer {
	return &HTMLRenderer{}
}

func NewPDFRenderer() *PDFRenderer {
	return &PDFRenderer{}
}

func NewRenderer() Renderer {
	return NewPDFRenderer()
}
