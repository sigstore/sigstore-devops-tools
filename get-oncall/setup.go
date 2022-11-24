package slack

import (
	"context"
	"os"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/slack-go/slack"
)

type Client struct {
	ctx         context.Context
	pdClient    *pagerduty.Client
	slackClient *slack.Client
	slackSecret string
}

func setup(ctx context.Context) Client {
	pagerDutyKey := os.Getenv("PD_API_KEY")
	slackAPIKey := os.Getenv("SLACK_API_KEY")
	slackSecret := os.Getenv("SLACK_SECRET")

	pdClient := pagerduty.NewClient(pagerDutyKey)

	slackClient := slack.New(slackAPIKey)

	return Client{
		ctx:         ctx,
		pdClient:    pdClient,
		slackClient: slackClient,
		slackSecret: slackSecret,
	}
}
