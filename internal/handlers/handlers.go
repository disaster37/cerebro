package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/sirupsen/logrus"

	"cerebro/internal/auth"
	"cerebro/internal/config"
	"cerebro/internal/dao"
	"cerebro/internal/elastic"
	"cerebro/internal/models"
)

type Handlers struct {
	Auth    auth.AuthService
	Hosts   *config.Hosts
	Elastic elastic.ElasticClient
	History *dao.SQLiteDAO
	HistMax int
	Log     *logrus.Logger
	Secret  string
}

type CerebroRequest struct {
	Target models.ElasticServer
	Body   json.RawMessage
	User   *models.User
}

type MissingRequiredParamError struct{ Field string }

func (e *MissingRequiredParamError) Error() string { return e.Field + " is required" }

type RequestFailedError struct {
	Path   string
	Status int
	Body   string
}

func (e *RequestFailedError) Error() string { return "Failure for [" + e.Path + "]" }

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

type Processor func(ctx context.Context, req CerebroRequest) (models.Response, error)

func (h *Handlers) process(c context.Context, ctx *app.RequestContext, fn Processor) {
	var user *models.User
	if u, ok := ctx.Get("user"); ok {
		user, _ = u.(*models.User)
	}

	var body json.RawMessage
	if len(ctx.Request.Body()) > 0 {
		body = ctx.Request.Body()
	}

	hostName := extractField(body, "host")
	if hostName == "" {
		ctx.JSON(http.StatusOK, models.ErrorResponse(400, "missing host"))
		return
	}

	targetHost, found := h.Hosts.GetHost(hostName)
	if !found {
		targetHost = models.Host{Name: hostName}
	}

	reqUsername := extractField(body, "username")
	reqPassword := extractField(body, "password")
	if reqUsername != "" && reqPassword != "" {
		// Per-request credentials override the host auth. Assign a NEW struct:
		// targetHost.Auth aliases the shared Hosts registry entry, and mutating
		// it would leak these credentials into other requests (and race).
		targetHost.Auth = &models.ESAuth{Username: reqUsername, Password: reqPassword}
	}

	var headers []models.KeyValue
	for _, hname := range targetHost.HeadersWhitelist {
		if val := ctx.GetHeader(hname); len(val) > 0 {
			headers = append(headers, models.KeyValue{Key: hname, Value: string(val)})
		}
	}

	target := models.ElasticServer{Host: targetHost, Headers: headers}
	req := CerebroRequest{Target: target, Body: body, User: user}

	resp, err := fn(c, req)
	if err != nil {
		switch e := err.(type) {
		case *RequestFailedError:
			body, _ := json.Marshal(map[string]string{"error": e.Error()})
			ctx.JSON(http.StatusOK, models.Response{Status: e.Status, Body: body})
			return
		case *MissingRequiredParamError:
			ctx.JSON(http.StatusOK, models.ErrorResponse(400, e.Error()))
			return
		default:
			h.Log.Errorf("Error processing request: %v", err)
			ctx.JSON(http.StatusOK, models.ErrorResponse(500, err.Error()))
			return
		}
	}

	ctx.JSON(http.StatusOK, resp)
}

func extractField(body json.RawMessage, key string) string {
	var m map[string]json.RawMessage
	_ = json.Unmarshal(body, &m)
	v := m[key]
	var s string
	_ = json.Unmarshal(v, &s)
	return s
}
