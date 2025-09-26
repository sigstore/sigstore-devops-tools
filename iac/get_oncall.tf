# Copyright 2024 The Sigstore Authors
# SPDX-License-Identifier: Apache-2.0

resource "google_service_account" "slack_slash_pg_sa" {
  project = var.project_id

  account_id   = "${var.name}-get-oncall"
  display_name = "GetOncall - Slack slash Pagerduty"
  description  = "Dedicated service account for the Slack slash Pagerduty service."
}

data "google_secret_manager_secret" "slack_secret" {
  secret_id = "SLACK_SECRET"
}

resource "google_secret_manager_secret_iam_binding" "slack_api_secret_binding" {
  secret_id = data.google_secret_manager_secret.slack_secret.id
  role      = "roles/secretmanager.secretAccessor"
  members = [
    "serviceAccount:${google_service_account.slack_slash_pg_sa.email}",
  ]
}

module "slack_slash_pg_service" {
  source  = "chainguard-dev/common/infra//modules/regional-go-service"
  version = "0.7.3"

  project_id    = var.project_id
  name          = "${var.name}-slack-slash-pg"
  regions       = module.networking.regional-networks

  ingress = "INGRESS_TRAFFIC_ALL"
  // This needs to egress in order to talk to Slack and PagerDuty
  egress = "PRIVATE_RANGES_ONLY"

  service_account = google_service_account.slack_slash_pg_sa.email
  containers = {
    "slack-slash-pg" = {
      source = {
        working_dir = "${path.module}/../"
        importpath  = "github.com/sigstore/sigstore-devops-tools/cmd/get_oncall"
      }

      ports = [{ container_port = 8080 }]

      env = [
        {
          name = "SLACK_API_KEY"
          value_source = {
            secret_key_ref = {
              secret  = data.google_secret_manager_secret.slack_api_secret.secret_id
              version = "latest"
            }
          }
        },
        {
          name = "SLACK_SECRET"
          value_source = {
            secret_key_ref = {
              secret  = data.google_secret_manager_secret.slack_secret.secret_id
              version = "latest"
            }
          }
        },
        {
          name = "PD_API_KEY"
          value_source = {
            secret_key_ref = {
              secret  = data.google_secret_manager_secret.pagerduty_secret.secret_id
              version = "latest"
            }
          }
        },
      ]
    }
  }

  notification_channels = local.notification_channels

  depends_on = [
    google_secret_manager_secret_iam_binding.slack_api_secret_binding,
    google_secret_manager_secret_iam_binding.slack_secret_binding,
    google_service_account.slack_slash_pg_sa
  ]
}
