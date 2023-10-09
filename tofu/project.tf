resource "google_project_service" "project_services" {
    project = var.project_id
    for_each = toset(local.service_list)
    service = each.key
}

data "google_iam_policy" "tofu" {
    binding {
        role = "projects/${var.project_id}/roles/tofu"
        members = [
            "serviceAccount:${google_service_account.tofu_sa.email}"
        ]
    }
}

resource "google_service_account" "tofu_sa" {
    account_id   = "tofu-sa"
    display_name = "Tofu Service Account"
    project      = var.project_id

    depends_on = [ google_project_service.project_services ]
}


resource "google_service_account_iam_policy" "tofu_sa_iam" {
    service_account_id = google_service_account.tofu_sa.id
    policy_data = data.google_iam_policy.tofu.policy_data

    depends_on = [ google_project_service.project_services ]
}

resource "google_iam_workload_identity_pool" "ci_pool" {
  workload_identity_pool_id = "ci-pool"
  display_name              = "CI Pool"
  description               = "Identity pool for GitHub Actions"
}

resource "google_iam_workload_identity_pool_provider" "gha_oidc" {
  workload_identity_pool_id          = google_iam_workload_identity_pool.ci_pool.workload_identity_pool_id
  workload_identity_pool_provider_id = "github-actions"
  display_name                       = "GitHub Actions OIDC Provider"
  description                        = "OIDC identity pool provider for GitHub Actions"

  attribute_mapping                  = {
    "google.subject"                  = "assertion.sub"
    "attribute.actor"                   = "assertion.actor"
    "attribute.repository"            = "assertion.repository"
  }
  oidc {
    allowed_audiences = [ "https://github.com/octo-org" ]
    issuer_uri        = "https://token.actions.githubusercontent.com"
  }
}

resource "google_service_account_iam_binding" "tofu_sa_pool_iam" {
  service_account_id = google_service_account.tofu_sa.id
  role               = "roles/iam.workloadIdentityUser"

  members = [ "principalSet://iam.googleapis.com/${google_iam_workload_identity_pool.ci_pool.name}/attribute.repository/${var.repository}" ]
}
