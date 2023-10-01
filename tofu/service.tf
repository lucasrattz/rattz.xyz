resource "google_artifact_registry_repository" "app-repository" {
  provider     = google-beta
  location      = var.region
  repository_id = local.repository_id
  description   = "Repository for the portfolio application Docker images"
  format        = "DOCKER"

  cleanup_policies {
    id = "delete_older_than_${local.keep_days}d"
    action = "DELETE"
    condition {
      older_than = "${local.keep_days * 86400}s"
    }
  }

  cleanup_policies {
    id = "keep_last_${local.keep_num}"
    action = "KEEP"
    most_recent_versions {
      keep_count = local.keep_num
    }
  }
}

resource "google_cloud_run_v2_service" "app" {
  name = local.service_name
  location = var.region
  ingress = "INGRESS_TRAFFIC_ALL"

  template {
    containers {
      name = local.service_name
      image = data.external.image_digest.result.image

      ports {
        container_port = local.app_port
      }

      env {
        name = "HOST"
        value = local.app_host
      }

      env {
        name = "REMOTE_PROFILE_URL"
        value = var.remote_profile_url
      }
    }
  }

  traffic {
    type = "TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST"
    percent = 100
  }
}

data "google_iam_policy" "noauth" {
  binding {
    role = "roles/run.invoker"
    members = [
      "allUsers",
    ]
  }
}

resource "google_cloud_run_service_iam_policy" "noauth" {
  location = google_cloud_run_v2_service.app.location
  project  = google_cloud_run_v2_service.app.project
  service  = google_cloud_run_v2_service.app.name

  policy_data = data.google_iam_policy.noauth.policy_data
  depends_on  = [ google_cloud_run_v2_service.app ]
}

data "external" "image_digest" {
  program = ["bash", "./scripts/get_latest_tag.sh", var.project_id, var.region, local.repository_id, local.service_name]
}
