package formatter

import "encoding/json"

func init() {
	RegisterFormatter("json", func() Formatter {
		return &jsonFormatter{}
	})
}

type jsonFormatter struct {
}

func (j jsonFormatter) Initialize(data []byte) error {
	return nil
}

func (j jsonFormatter) Format(val interface{}) ([]byte, error) {
	return json.Marshal(val)
}
