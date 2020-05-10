package grafana

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"

	"github.com/pkg/errors"
	"golang.org/x/crypto/pkcs12"

	"github.com/LifeMemoryTeam/slack-grafana-image-renderer-picker/pkg/config"
)

type Graph struct {
	Graph *bytes.Buffer
	URL   string
}

type Client struct {
	endpoint   string
	authId     string
	authHeader string
	client     *http.Client
}

func NewClient(endpoint string) *Client {
	return &Client{
		endpoint: endpoint,
		client:   &http.Client{},
	}
}

type DsoloParams struct {
	OrgId   string
	PanelId string
	From    string
	To      string
}

type Request http.Request

type Option func(*url.Values) *url.Values

func PanelId(panelId string) Option {
	return func(v *url.Values) *url.Values {
		v.Add("panelId", panelId)
		return v
	}
}

func From(from string) Option {
	return func(v *url.Values) *url.Values {
		v.Add("from", from)
		return v
	}
}

func To(to string) Option {
	return func(v *url.Values) *url.Values {
		v.Add("to", to)
		return v
	}
}

func OrgId(orgid string) Option {
	return func(v *url.Values) *url.Values {
		v.Add("orgId", orgid)
		return v
	}
}

func (c *Client) LoadP12(keyPath, password string) error {
	fb, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return errors.WithStack(err)
	}

	b, err := pkcs12.ToPEM(fb, password)
	if err != nil {
		return errors.WithStack(err)
	}

	if len(b) < 3 {
		return errors.New("p12 key needs 3 fields")
	}

	cert, err := tls.X509KeyPair(pem.EncodeToMemory(b[0]), pem.EncodeToMemory(b[2]))
	if err != nil {
		return errors.WithStack(err)
	}

	caCertPool, _ := x509.SystemCertPool()
	caCertPool.AppendCertsFromPEM(b[1].Bytes)

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	}
	tlsConfig.BuildNameToCertificate()
	transport := &http.Transport{TLSClientConfig: tlsConfig}

	c.client.Transport = transport
	return nil
}

func (c *Client) GetDsolo(name string, opts ...Option) (*Graph, error) {
	d, err := config.GetDashboard(name)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	o := []Option{OrgId(d.OrgID), PanelId(d.PanelID)}
	for _, v := range opts {
		o = append(o, v)
	}
	return c.getDsolo(d.DashboardID, d.DashboardName, o...)
}

func (c *Client) getDsolo(dashboardId, dashboardName string, option ...Option) (*Graph, error) {
	endpoint, err := url.Parse(c.endpoint)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	endpoint.Path = path.Join("/render/d-solo/", dashboardId, "/", dashboardName)
	req := &Request{URL: endpoint, Method: http.MethodGet}
	if c.authHeader != "" {
		req.Header = make(http.Header)
		req.Header.Set(c.authHeader, c.authId)
	}
	params := req.URL.Query()
	for _, v := range option {
		v(&params)
	}
	req.URL.RawQuery = params.Encode()
	resp, err := c.client.Do((*http.Request)(req))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &Graph{
		Graph: bytes.NewBuffer(data),
		URL:   endpoint.String(),
	}, nil
}
