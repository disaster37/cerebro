package domain

import (
	"encoding/json"
)

type CerebroRequest struct {
	Target ElasticServer
	Body   json.RawMessage
	User   *User
}

func (r CerebroRequest) Get(name string) (string, error) {
	var m map[string]json.RawMessage
	if json.Unmarshal(r.Body, &m) != nil {
		return "", &MissingRequiredParamError{Field: name}
	}
	v, ok := m[name]
	if !ok {
		return "", &MissingRequiredParamError{Field: name}
	}
	var s string
	if json.Unmarshal(v, &s) != nil {
		return "", &MissingRequiredParamError{Field: name}
	}
	return s, nil
}

func (r CerebroRequest) GetOpt(name string) string {
	var m map[string]json.RawMessage
	_ = json.Unmarshal(r.Body, &m)
	v := m[name]
	var s string
	_ = json.Unmarshal(v, &s)
	return s
}

func (r CerebroRequest) GetInt(name string) (int, error) {
	var m map[string]json.RawMessage
	_ = json.Unmarshal(r.Body, &m)
	v, ok := m[name]
	if !ok {
		return 0, &MissingRequiredParamError{Field: name}
	}
	var n int
	if json.Unmarshal(v, &n) == nil {
		return n, nil
	}
	return 0, &MissingRequiredParamError{Field: name}
}

func (r CerebroRequest) GetBool(name string) (bool, error) {
	var m map[string]json.RawMessage
	_ = json.Unmarshal(r.Body, &m)
	v, ok := m[name]
	if !ok {
		return false, &MissingRequiredParamError{Field: name}
	}
	var b bool
	if json.Unmarshal(v, &b) == nil {
		return b, nil
	}
	return false, &MissingRequiredParamError{Field: name}
}

func (r CerebroRequest) GetObj(name string) (json.RawMessage, error) {
	var m map[string]json.RawMessage
	if json.Unmarshal(r.Body, &m) != nil {
		return nil, &MissingRequiredParamError{Field: name}
	}
	v, ok := m[name]
	if !ok {
		return nil, &MissingRequiredParamError{Field: name}
	}
	return v, nil
}

func (r CerebroRequest) GetObjOpt(name string) json.RawMessage {
	var m map[string]json.RawMessage
	_ = json.Unmarshal(r.Body, &m)
	return m[name]
}

func (r CerebroRequest) GetArray(name string) ([]string, error) {
	var m map[string]json.RawMessage
	_ = json.Unmarshal(r.Body, &m)
	v, ok := m[name]
	if !ok {
		return nil, &MissingRequiredParamError{Field: name}
	}
	var arr []string
	if json.Unmarshal(v, &arr) == nil {
		return arr, nil
	}
	return nil, &MissingRequiredParamError{Field: name}
}

func (r CerebroRequest) GetStringArray(name string) []string {
	arr, _ := r.GetArray(name)
	return arr
}
