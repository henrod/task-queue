package deprecated

import (
	"fmt"
	"github.com/bitly/go-simplejson"
)

type data struct {
	*simplejson.Json
}

type Msg struct {
	*data
	original string
}

func NewMsg(content string) (*Msg, error) {
	if d, err := newData(content); err != nil {
		return nil, err
	} else {
		return &Msg{d, content}, nil
	}
}

func newData(content string) (*data, error) {
	contentJson, err := simplejson.NewJson([]byte(content))
	if err != nil {
		return nil, fmt.Errorf("failed to convert content to simpleJson: %w", err)
	}

	dataJSON := simplejson.New()
	dataJSON.Set("args", contentJson)

	return &data{dataJSON}, nil
}
