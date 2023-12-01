package helper

import (
	"bytes"
	"github.com/SebastiaanKlippert/go-wkhtmltopdf"
	"html/template"
	"io/ioutil"
	"os"
	"strings"
)

//pdf requestpdf struct
type RequestPdf struct {
	body string
}

//new request to pdf function
func NewRequestPdf(body string) *RequestPdf {
	return &RequestPdf{
		body: body,
	}
}

//parsing template function
func (r *RequestPdf) ParseTemplate(templateFileName string, data interface{}) error {

	t, err := template.ParseFiles(templateFileName)
	if err != nil {
		return err
	}
	buf := new(bytes.Buffer)
	if err = t.Execute(buf, data); err != nil {

		return err
	}
	r.body = buf.String()
	return nil
}

//generate pdf function
func (r *RequestPdf) GeneratePDF(pdfPath string, outputPath string) error {
	// write whole the body
	err := ioutil.WriteFile(outputPath, []byte(r.body), os.ModePerm)
	if err != nil {
		return err
	}

	b, err := os.ReadFile(outputPath)
	if err != nil {
		return err
	}

	pdfg, err := wkhtmltopdf.NewPDFGenerator()
	if err != nil {
		return err
	}
	pdfg.AddPage(wkhtmltopdf.NewPageReader(bytes.NewReader(b)))
	//pdfg.PageSize.Set(wkhtmltopdf.PageSizeA4)
	//pdfg.Dpi.Set(100)
	err = pdfg.Create()
	if err != nil {
		return err
	}

	splitStr := strings.Split(pdfPath, "/")
	folder := strings.Replace(pdfPath, "/"+splitStr[len(splitStr)-1], "", 1)

	if err = os.MkdirAll(folder, os.ModePerm); err != nil {
		return err
	}
	err = pdfg.WriteFile(pdfPath)
	if err != nil {
		return err
	}

	return nil
}

func (r *RequestPdf) GeneratePDFByBody(pdfPath string) error {
	// write whole the body
	pdfg, err := wkhtmltopdf.NewPDFGenerator()
	if err != nil {
		return err
	}
	pdfg.ResetPages()
	pdfg.AddPage(wkhtmltopdf.NewPageReader(strings.NewReader(r.body)))
	pdfg.PageSize.Set(wkhtmltopdf.PageSizeA4)
	pdfg.Dpi.Set(100)
	if err = pdfg.Create(); err != nil {
		return err
	}

	splitStr := strings.Split(pdfPath, "/")
	folder := strings.Replace(pdfPath, "/"+splitStr[len(splitStr)-1], "", 1)

	if err = os.MkdirAll(folder, os.ModePerm); err != nil {
		return err
	}

	if err = pdfg.WriteFile(pdfPath); err != nil {
		return err
	}

	return nil
}
