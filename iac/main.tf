# Copyright 2024 The Sigstore Authors
# SPDX-License-Identifier: Apache-2.0

provider "google" { project = var.project_id }
provider "google-beta" { project = var.project_id }
provider "ko" { repo = "us-docker.pkg.dev/sigstore-support-tooling/${var.project_id}" }

// Create a network with several regional subnets
module "networking" {
  source  = "chainguard-dev/common/infra//modules/networking"
  version = "0.6.106"

  name          = var.name
  project_id    = var.project_id
  regions       = var.regions
  netnum_offset = 1
}


# For slack need to create the notification manually - https://github.com/hashicorp/terraform-provider-google/issues/11346
data "google_monitoring_notification_channel" "devops_slack" {
  display_name = "Slack Sigstore Devops Notification"
}

locals {
  notification_channels = [
    data.google_monitoring_notification_channel.devops_slack.name
  ]
}

// this secret is used in three services: slack-slash-pagerduty, github-issue-assigner and pagerduty-notify-change
data "google_secret_manager_secret" "pagerduty_secret" {
  secret_id = "PD_API_KEY"
}

// the services slack-slash-pagerduty / github-issue-assigner / pagerduty-notify-change need access to the Pagerduty api key
// in order to fetch who is oncall
resource "google_secret_manager_secret_iam_binding" "pagerduty_secret_binding" {
  secret_id = data.google_secret_manager_secret.pagerduty_secret.id
  role      = "roles/secretmanager.secretAccessor"
  members = [
    "serviceAccount:${google_service_account.slack_slash_pg_sa.email}",
  ]
}

// this secret is used in three services: slack-slash-pagerduty and pagerduty-notify-change
data "google_secret_manager_secret" "slack_api_secret" {
  secret_id = "SLACK_API_KEY"
}

resource "google_secret_manager_secret_iam_binding" "slack_secret_binding" {
  secret_id = data.google_secret_manager_secret.slack_api_secret.id
  role      = "roles/secretmanager.secretAccessor"
  members = [
    "serviceAccount:${google_service_account.slack_slash_pg_sa.email}",
  ]
}
