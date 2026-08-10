package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/sirupsen/logrus"

	"cerebro/internal/domain"
	"cerebro/internal/service"
)

// HostResolver resolves a configured target host by name. Satisfied by
// *config.Hosts so handler does not import config.
type HostResolver interface {
	GetHost(name string) (domain.Host, bool)
	GetHostNames() []string
}

// Handlers is the aggregate HTTP adapter: auth/host/session plumbing plus
// one field per feature service. The feature methods are one-liner
// delegations to the services via process.
type Handlers struct {
	Auth   domain.AuthService
	Hosts  HostResolver
	Log    *logrus.Logger
	Secret string

	Connect         *service.ConnectService
	Overview        *service.OverviewService
	Nodes           *service.NodesService
	Navbar          *service.NavbarService
	Rest            *service.RestService
	Aliases         *service.AliasesService
	Analysis        *service.AnalysisService
	Cat             *service.CatService
	ClusterChanges  *service.ClusterChangesService
	ClusterSettings *service.ClusterSettingsService
	Commons         *service.CommonsService
	CreateIndex     *service.CreateIndexService
	IndexSettings   *service.IndexSettingsService
	Repositories    *service.RepositoriesService
	Snapshots       *service.SnapshotsService
	Templates       *service.TemplatesService
}

// Processor is the uniform service-method signature: every service method
// wired through process is assignable to it, so handlers delegate with a
// plain method value (h.process(ctx, c, h.Xxx.SomeMethod)).
type Processor func(ctx context.Context, req domain.CerebroRequest) (domain.Response, error)

// process builds a domain.CerebroRequest from the HTTP request (session
// user, JSON body, resolved target host with per-request credentials and
// whitelisted headers), runs fn, and writes the {status, body} envelope.
func (h *Handlers) process(c context.Context, ctx *app.RequestContext, fn Processor) {
	var user *domain.User
	if u, ok := ctx.Get("user"); ok {
		user, _ = u.(*domain.User)
	}

	var body json.RawMessage
	if len(ctx.Request.Body()) > 0 {
		body = ctx.Request.Body()
	}

	hostName := extractField(body, "host")
	if hostName == "" {
		ctx.JSON(http.StatusOK, domain.ErrorResponse(400, "missing host"))
		return
	}

	targetHost, found := h.Hosts.GetHost(hostName)
	if !found {
		targetHost = domain.Host{Name: hostName}
	}

	reqUsername := extractField(body, "username")
	reqPassword := extractField(body, "password")
	if reqUsername != "" && reqPassword != "" {
		// Per-request credentials override the host auth. Assign a NEW struct:
		// targetHost.Auth aliases the shared Hosts registry entry, and mutating
		// it would leak these credentials into other requests (and race).
		targetHost.Auth = &domain.ESAuth{Username: reqUsername, Password: reqPassword}
	}

	var headers []domain.KeyValue
	for _, hname := range targetHost.HeadersWhitelist {
		if val := ctx.GetHeader(hname); len(val) > 0 {
			headers = append(headers, domain.KeyValue{Key: hname, Value: string(val)})
		}
	}

	target := domain.ElasticServer{Host: targetHost, Headers: headers}
	req := domain.CerebroRequest{Target: target, Body: body, User: user}

	resp, err := fn(c, req)
	if err != nil {
		switch e := err.(type) {
		case *domain.RequestFailedError:
			body, _ := json.Marshal(map[string]string{"error": e.Error()})
			ctx.JSON(http.StatusOK, domain.Response{Status: e.Status, Body: body})
			return
		case *domain.MissingRequiredParamError:
			ctx.JSON(http.StatusOK, domain.ErrorResponse(400, e.Error()))
			return
		default:
			h.Log.Errorf("Error processing request: %v", err)
			ctx.JSON(http.StatusOK, domain.ErrorResponse(500, err.Error()))
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
