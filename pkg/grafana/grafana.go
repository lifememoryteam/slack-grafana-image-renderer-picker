package grafana

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"

	"github.com/atpons/slack-grafana-image-renderer-picker/pkg/config"
	"github.com/pkg/errors"
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

func NewClient(endpoint, authHeader, authId string) *Client {
	return &Client{
		endpoint:   endpoint,
		authHeader: authHeader,
		authId:     authId,
		client:     &http.Client{},
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

func (c *Client) GetDsolo(name string) (*Graph, error) {
	d, err := config.GetDashboard(name)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return c.getDsolo(d.DashboardID, d.DashboardName, OrgId(d.OrgID), PanelId(d.PanelID))
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
