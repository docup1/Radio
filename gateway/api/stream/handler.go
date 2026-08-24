package stream

import (
	"net/http/httputil"

	"radio/gateway/infra"
)

type Handler struct {
	proxy *httputil.ReverseProxy
	auth  *infra.AuthService
}

func New(proxy *httputil.ReverseProxy, auth *infra.AuthService) *Handler {
	return &Handler{proxy: proxy, auth: auth}
}
