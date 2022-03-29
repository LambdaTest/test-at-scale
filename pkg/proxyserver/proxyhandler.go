package proxyserver

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/LambdaTest/test-at-scale/pkg/global"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
	"github.com/spf13/viper"
)

// ProxyHandler defines struct for proxy handler
type ProxyHandler struct {
	remote *url.URL
	logger lumber.Logger
}

const synapseURL = "/synapse"

// NewProxyHandler returns pointer of new instace of ProxyHandler
func NewProxyHandler(logger lumber.Logger) (*ProxyHandler, error) {
	remote, err := url.Parse(global.TASCloudURL[viper.GetString("env")])
	if err != nil {
		return nil, err
	}

	return &ProxyHandler{
		remote: remote,
		logger: logger,
	}, nil
}

// HandlerProxy handles the proxy server
func (ph *ProxyHandler) HandlerProxy(w http.ResponseWriter, r *http.Request) {

	proxy := httputil.NewSingleHostReverseProxy(ph.remote)
	proxy.Director = func(req *http.Request) {
		req.Header = r.Header

		encodedSecret := base64.StdEncoding.EncodeToString([]byte(viper.GetString("Lambdatest.SecretKey")))
		req.Header.Add("Lambdatest-SecretKey", encodedSecret)
		req.Host = ph.remote.Host
		req.URL.Scheme = ph.remote.Scheme
		req.URL.Host = ph.remote.Host
		req.URL.Path = fmt.Sprintf("%s%s", synapseURL, r.URL.Path)

		ph.logger.Debugf("proxying to url: %s", req.URL.Path)
	}

	proxy.ServeHTTP(w, r)
}
