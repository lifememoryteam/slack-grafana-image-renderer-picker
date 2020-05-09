package slack

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/atpons/slack-grafana-image-renderer-picker/pkg/config"
	"github.com/atpons/slack-grafana-image-renderer-picker/pkg/grafana"
	"github.com/nlopes/slack"
	"github.com/pkg/errors"
)

const (
	InvokeSlackGrafanaImageRenderCommand = "/graph"
)

type Slack struct {
	Token   string
	Secret  string
	grafana *grafana.Client

	slack  *slack.Client
	server *http.Server
}

func NewSlackServer(grafana *grafana.Client, token, secret, addr string) *Slack {
	s := &Slack{}
	s.grafana = grafana
	s.Token = token
	s.Secret = secret
	s.slack = slack.New(token)

	mux := http.NewServeMux()
	mux.HandleFunc("/slash", s.slashHandler)
	s.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return s
}

func (s *Slack) Start() error {
	return s.server.ListenAndServe()
}

func (s *Slack) slashHandler(w http.ResponseWriter, r *http.Request) {
	verifier, err := slack.NewSecretsVerifier(r.Header, s.Secret)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	r.Body = ioutil.NopCloser(io.TeeReader(r.Body, &verifier))
	slackRes, err := slack.SlashCommandParse(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err = verifier.Ensure(); err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	log.Println(slackRes)

	switch slackRes.Command {
	case InvokeSlackGrafanaImageRenderCommand:
		log.Printf("%s", slackRes.Text)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		params := &slack.Msg{}
		if _, err := config.GetDashboard(slackRes.Text); err != nil {
			params.Text = "no graph"
		} else {
			params.Text = "ok, let's take graph..."
			go func() {
				graph, err := s.grafana.GetDsolo(slackRes.Text)
				if err != nil {
					log.Println(err)
					return
				}
				if err := s.uploadGraph(slackRes.ChannelID, graph); err != nil {
					log.Println(err)
				}
			}()
		}
		b, err := json.Marshal(params)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	}
}

func (s *Slack) uploadGraph(channel string, graph *grafana.Graph) error {
	name := fmt.Sprintf("graph_%d.png", time.Now().UnixNano())
	params := slack.FileUploadParameters{
		InitialComment: graph.URL,
		Reader:         graph.Graph,
		Filename:       name,
		Channels:       []string{channel},
	}
	_, err := s.slack.UploadFile(params)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}
