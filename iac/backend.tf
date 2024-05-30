# Copyright 2024 The Sigstore Authors
# SPDX-License-Identifier: Apache-2.0

terraform {
  backend "gcs" {
    bucket = "sigstore-devops-tools-tfstate"
    prefix = "/devops"
  }
  required_providers {
    ko = {
      source = "ko-build/ko"
    }
  }
}
