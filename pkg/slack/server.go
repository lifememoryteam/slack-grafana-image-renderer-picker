package slack

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/nlopes/slack"
	"github.com/pkg/errors"

	"github.com/atpons/slack-grafana-image-renderer-picker/pkg/config"
	"github.com/atpons/slack-grafana-image-renderer-picker/pkg/grafana"
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
		args := strings.Split(slackRes.Text, " ")
		var from string
		if len(args) >= 2 {
			from, err = grafana.ParseTimeRange(args[1])
			if err != nil {
				s.responseWithMessage("time range is invalid", w)
				return
			}
		}

		if _, err := config.GetDashboard(args[0]); err != nil {
			s.responseWithMessage("no graph", w)
			return
		} else {
			go func() {
				graph, err := s.getGraphDsolo(args[0], from)
				if err != nil {
					log.Println(err)
					return
				}
				if err := s.uploadGraph(slackRes.ChannelID, graph); err != nil {
					log.Println(err)
				}
			}()
			s.responseWithMessage("taking graph...", w)
		}

	}
}

func (s *Slack) getGraphDsolo(graphName, from string) (*grafana.Graph, error) {
	if from == "" {
		return s.grafana.GetDsolo(graphName)
	}
	return s.grafana.GetDsolo(graphName, grafana.From(from), grafana.To("now"))
}

func (s *Slack) responseWithMessage(message string, w http.ResponseWriter) {
	params := &slack.Msg{}
	params.Text = message
	b, err := json.Marshal(params)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
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
