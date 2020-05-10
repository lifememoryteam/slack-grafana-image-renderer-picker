package config

import (
	"io/ioutil"
	"sync"

	"github.com/goccy/go-yaml"
	"github.com/pkg/errors"
)

var Global *Config

func init() {
	graph = make(map[string]Dashboard, 0)
}

type Config struct {
	Slack struct {
		Token  string `yaml:"token"`
		Secret string `yaml:"secret"`
		Addr   string `yaml:"addr"`
	} `yaml:"slack"`
	Grafana struct {
		UseClientAuth bool   `yaml:"use_client_auth"`
		ClientAuthP12 string `yaml:"client_auth_p12"`
		Endpoint      string `yaml:"endpoint"`
	} `yaml:"grafana"`
	Dashboards []Dashboard `yaml:"dashboards"`
}

type Dashboard struct {
	Name          string `yaml:"name"`
	DashboardID   string `yaml:"dashboardId"`
	DashboardName string `yaml:"dashboardName"`
	OrgID         string `yaml:"orgId"`
	PanelID       string `yaml:"panelId"`
}

var graph map[string]Dashboard
var graphMu sync.RWMutex

func Load(path string) error {
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return errors.WithStack(err)
	}
	config := &Config{}
	if err := yaml.Unmarshal(buf, config); err != nil {
		return errors.WithStack(err)
	}
	Global = config

	graphMu.Lock()
	defer graphMu.Unlock()
	for _, v := range config.Dashboards {
		graph[v.Name] = v
	}

	return nil
}

func GetDashboard(name string) (*Dashboard, error) {
	graphMu.RLock()
	defer graphMu.RUnlock()
	v, ok := graph[name]
	if !ok {
		return nil, errors.New("no graph")
	}
	return &v, nil
}
