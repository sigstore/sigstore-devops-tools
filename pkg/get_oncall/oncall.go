// Copyright 2024 The Sigstore Authors
// SPDX-License-Identifier: Apache-2.0

package get_oncall

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/chainguard-dev/clog"
	"github.com/kelseyhightower/envconfig"
	"github.com/slack-go/slack"

	sg_slack "github.com/sigstore/sigstore-devops-tools/pkg/slack"
)

type oldTimeStampError struct {
	s string
}

func (e *oldTimeStampError) Error() string {
	return e.s
}

const (
	version                     = "v0"
	slackRequestTimestampHeader = "X-Slack-Request-Timestamp"
	slackSignatureHeader        = "X-Slack-Signature"
)

type Config struct {
	Port string `envconfig:"PORT" default:"8080"`

	PagerDutyAPIKey string `envconfig:"PD_API_KEY" required:"true"`
	SlackAPIKey     string `envconfig:"SLACK_API_KEY" required:"true"`
	SlackSecret     string `envconfig:"SLACK_SECRET" required:"true"`
}

type Client struct {
	ctx         context.Context
	pdClient    *pagerduty.Client
	slackClient *slack.Client
	slackSecret string
	port        string
}

func New(ctx context.Context) (*Client, error) {
	config, err := getConfig()
	if err != nil {
		clog.FromContext(ctx).Errorf(err.Error())
		return nil, err
	}

	return &Client{
		ctx:         ctx,
		pdClient:    pagerduty.NewClient(config.PagerDutyAPIKey),
		slackClient: slack.New(config.SlackAPIKey),
		slackSecret: config.SlackSecret,
		port:        config.Port,
	}, nil
}

func getConfig() (*Config, error) {
	var c Config
	err := envconfig.Process("", &c)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *Client) StartServer() {
	log := clog.FromContext(c.ctx)

	http.HandleFunc("/", c.GetOncall)

	// Start HTTP server.
	log.Infof("Listening on port %s", c.port)
	if err := http.ListenAndServe(":"+c.port, nil); err != nil {
		log.Fatal("failed to listen and serve apk-events server", err)
	}
}

// KGSearch uses the Knowledge Graph API to search for a query provided
// by a Slack command.
func (c *Client) GetOncall(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Fatalf("Couldn't read request body: %v", err)
	}
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	if r.Method != "POST" {
		http.Error(w, "Only POST requests are accepted", 405)
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Couldn't parse form", 400)
		log.Fatalf("ParseForm: %v", err)
	}

	// Reset r.Body as ParseForm depletes it by reading the io.ReadCloser.
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	result, err := verifyWebHook(r, c.slackSecret)
	if err != nil {
		log.Fatalf("verifyWebhook: %v", err)
	}
	if !result {
		log.Fatalf("signatures did not match.")
	}

	onCallResponse, err := c.getOncall()
	if err != nil {
		log.Fatalf("getOncall: %v", err)
	}
	w.Header().Set("Content-Type", "application/json")
	if err = json.NewEncoder(w).Encode(onCallResponse); err != nil {
		log.Fatalf("json.Marshal: %v", err)
	}
}

func (c *Client) getOncall() (*slack.Msg, error) {
	var opts pagerduty.ListOnCallOptions
	eps, err := c.pdClient.ListOnCallsWithContext(context.Background(), opts)
	if err != nil {
		panic(err)
	}

	if len(eps.OnCalls) == 0 {
		return sg_slack.SetSimpleSlackMessage("OnCall", "No one is Oncall at this time", "#13A554"), nil
	}

	for _, p := range eps.OnCalls {
		if p.EscalationLevel == 1 {
			u, err := c.pdClient.GetUserWithContext(context.Background(), p.User.ID, pagerduty.GetUserOptions{})
			if err != nil {
				panic(err)
			}
			date, err := time.Parse(time.RFC3339, p.End)
			if err != nil {
				panic(err)
			}
			currentTime := time.Now()
			difference := date.Sub(currentTime)
			total := int(difference.Seconds())
			days := int(total / (60 * 60 * 24))
			hours := int(total / (60 * 60) % 24)
			minutes := int(total/60) % 60
			endTimeMsg := fmt.Sprintf("%s is on-call for the next %d days %d hours", u.Name, days, hours)
			if days == 0 {
				endTimeMsg = fmt.Sprintf("%s is on-call for the next %d hours %d minutes", u.Name, hours, minutes)
			}

			lContact, err := c.pdClient.ListUserContactMethodsWithContext(context.Background(), u.ID)
			if err != nil {
				panic(err)
			}

			var userEmails []string
			for _, contactMethod := range lContact.ContactMethods {
				if contactMethod.Type == "email_contact_method" {
					userEmails = append(userEmails, contactMethod.Address)
				}
			}

			slackUser := "not found email in slack, maybe it is using another one"
			for _, email := range userEmails {
				user, err := c.slackClient.GetUserByEmail(email)
				if err != nil && strings.Contains(err.Error(), "users_not_found") {
					continue
				} else if err != nil {
					panic(err)
				}
				slackUser = fmt.Sprintf("@%s", user.Name)
				break
			}

			return sg_slack.FormatSlackMessage(endTimeMsg, p.Schedule.Summary, u.Timezone, slackUser), nil
		}
	}

	return nil, fmt.Errorf("No schedules found")
}

// verifyWebHook verifies the request signature.
// See https://api.slack.com/docs/verifying-requests-from-slack.
func verifyWebHook(r *http.Request, slackSigningSecret string) (bool, error) {
	timeStamp := r.Header.Get(slackRequestTimestampHeader)
	slackSignature := r.Header.Get(slackSignatureHeader)

	t, err := strconv.ParseInt(timeStamp, 10, 64)
	if err != nil {
		return false, fmt.Errorf("strconv.ParseInt(%s): %v", timeStamp, err)
	}

	if ageOk, age := checkTimestamp(t); !ageOk {
		return false, &oldTimeStampError{fmt.Sprintf("checkTimestamp(%v): %v %v", t, ageOk, age)}
	}

	if timeStamp == "" || slackSignature == "" {
		return false, fmt.Errorf("either timeStamp or signature headers were blank")
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return false, fmt.Errorf("ioutil.ReadAll(%v): %v", r.Body, err)
	}

	// Reset the body so other calls won't fail.
	r.Body = io.NopCloser(bytes.NewBuffer(body))

	baseString := fmt.Sprintf("%s:%s:%s", version, timeStamp, body)

	signature := getSignature([]byte(baseString), []byte(slackSigningSecret))

	trimmed := strings.TrimPrefix(slackSignature, fmt.Sprintf("%s=", version))
	signatureInHeader, err := hex.DecodeString(trimmed)

	if err != nil {
		return false, fmt.Errorf("hex.DecodeString(%v): %v", trimmed, err)
	}

	return hmac.Equal(signature, signatureInHeader), nil
}

func getSignature(base []byte, secret []byte) []byte {
	h := hmac.New(sha256.New, secret)
	h.Write(base)

	return h.Sum(nil)
}

// Arbitrarily trusting requests time stamped less than 5 minutes ago.
func checkTimestamp(timeStamp int64) (bool, time.Duration) {
	t := time.Since(time.Unix(timeStamp, 0))

	return t.Minutes() <= 5, t
}
