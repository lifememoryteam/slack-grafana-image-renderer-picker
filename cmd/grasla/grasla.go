package main

import (
	"log"
	"os"

	"github.com/atpons/slack-grafana-image-renderer-picker/pkg/config"
	"github.com/atpons/slack-grafana-image-renderer-picker/pkg/grafana"
	"github.com/atpons/slack-grafana-image-renderer-picker/pkg/slack"
)

func main() {
	log.Println(os.Getenv("CONFIG_FILE"))
	if err := config.Load(os.Getenv("CONFIG_FILE")); err != nil {
		log.Println(err.Error())
		panic(err)
	}
	g := grafana.NewClient(config.Global.Grafana.Endpoint, config.Global.Grafana.AuthHeader, config.Global.Grafana.AuthID)
	server := slack.NewSlackServer(g, config.Global.Slack.Token, config.Global.Slack.Secret, config.Global.Slack.Addr)
	if err := server.Start(); err != nil {
		panic(err)
	}
}
