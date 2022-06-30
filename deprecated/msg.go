package deprecated

import (
	"encoding/json"
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

	contentMap := map[string]interface{}{}
	contentMap["args"] = contentJson
	contentMapBytes, err := json.Marshal(contentMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal contentMap: %w", err)
	}

	parentJson, err := simplejson.NewJson(contentMapBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to convert contentMap to simpleJson: %w", err)
	}

	return &data{parentJson}, nil
}
