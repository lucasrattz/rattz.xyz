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
