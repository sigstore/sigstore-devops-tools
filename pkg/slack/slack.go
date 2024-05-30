// Copyright 2024 The Sigstore Authors
// SPDX-License-Identifier: Apache-2.0

package slack

import (
	"github.com/slack-go/slack"
)

func FormatSlackMessage(msg, scheduleName, timezone, slackUser string) *slack.Msg {
	message := &slack.Msg{
		ResponseType: "in_channel",
		Attachments: []slack.Attachment{
			{
				Color: "#13A554",
				Title: scheduleName,
				Text:  msg,
				Fields: []slack.AttachmentField{
					{
						Title: "Timezone",
						Value: timezone,
						Short: true,
					},
					{
						Title: "Slack User",
						Value: slackUser,
						Short: true,
					},
				},
			},
		},
	}
	return message
}

func SetSimpleSlackMessage(title, msg, color string) *slack.Msg {
	message := &slack.Msg{
		ResponseType: "in_channel",
		Attachments: []slack.Attachment{
			{
				Color: color,
				Title: title,
				Text:  msg,
			},
		},
	}
	return message
}
