// DO NOT EDIT!
// Code generated by ffjson <https://github.com/pquerna/ffjson>
// source: app.go
// DO NOT EDIT!

package model

import (
	"bytes"

	"encoding/json"
)

func (mj *App) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.Grow(1024)
	err := mj.MarshalJSONBuf(&buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
func (mj *App) MarshalJSONBuf(buf *bytes.Buffer) error {
	var err error
	var obj []byte
	var first bool = true
	_ = obj
	_ = err
	_ = first
	buf.WriteString(`{`)
	if first == true {
		first = false
	} else {
		buf.WriteString(`,`)
	}
	buf.WriteString(`"components":`)
	/* Falling back. type=map[string]model.AppComponent kind=map */
	obj, err = json.Marshal(mj.Components)
	if err != nil {
		return err
	}
	buf.Write(obj)
	if first == true {
		first = false
	} else {
		buf.WriteString(`,`)
	}
	buf.WriteString(`"name":`)
	ffjson_WriteJsonString(buf, mj.Name)
	buf.WriteString(`}`)
	return nil
}