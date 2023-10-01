locals {
    service_name = "rattz-xyz"
    repository_id = "app-repository"
    keep_num = 3
    keep_days = 7
    service_list = [
        "cloudresourcemanager.googleapis.com",
        "iam.googleapis.com",
        "cloudkms.googleapis.com",
        "storage.googleapis.com",
        "cloudbuild.googleapis.com",
        "artifactregistry.googleapis.com",
        "run.googleapis.com",
        "compute.googleapis.com"
    ]
    app_host = "0.0.0.0"
    app_port = 80
}
