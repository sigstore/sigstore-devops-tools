# Copyright 2024 The Sigstore Authors
# SPDX-License-Identifier: Apache-2.0

variable "project_id" {
  description = "The project ID where all resources created will reside."
}

variable "name" {
  description = "Name indicator, prefixed to resources created."
  default     = "devops"
}

variable "regions" {
  description = "Regions where this environment's services should live."
  type        = list(string)
  default     = []
}
