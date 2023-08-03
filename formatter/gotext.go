package formatter

import (
	"bytes"
	"text/template"
)

func init() {
	RegisterFormatter("gotext", func() Formatter {
		return &goTextFormatter{
			Template: template.New(""),
		}
	})
}

type goTextFormatter struct {
	*template.Template
}

func (f goTextFormatter) Initialize(data []byte) error {
	var err error
	f.Template, err = f.Template.Parse(string(data))
	if err != nil {
		return err
	}
	return nil
}

func (f goTextFormatter) Format(val interface{}) ([]byte, error) {
	buffer := &bytes.Buffer{}
	if err := f.Template.Execute(buffer, val); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
