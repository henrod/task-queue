package deprecated

import (
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
	if json, err := simplejson.NewJson([]byte(content)); err != nil {
		return nil, err
	} else {
		return &data{json}, nil
	}
}
