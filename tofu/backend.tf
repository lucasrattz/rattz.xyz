terraform {
    backend "gcs" {
        bucket = "tfstate-bucket-3a7ee372adbd9520"
        prefix = "terraform/state"
    }
}

resource "random_id" "bucket_prefix" {
    byte_length = 8

    keepers = {
        project_id = var.project_id
    }

    lifecycle {
        prevent_destroy = true
    }
}

resource "google_kms_key_ring" "tf_state" {
    name     = "tfstate-bucket-${random_id.bucket_prefix.hex}"
    location = var.region

    lifecycle {
        prevent_destroy = true
    }

    depends_on = [ google_project_service.project_services ]
}

resource "google_kms_crypto_key" "tf_state_bucket" {
    name = "tf-state-bucket"
    key_ring = google_kms_key_ring.tf_state.id

    lifecycle {
        prevent_destroy = true
    }
}


resource "google_storage_bucket" "tf_state_bucket" {
    name          = "tfstate-bucket-${random_id.bucket_prefix.hex}"
    force_destroy = false
    location      = upper(var.region)
    storage_class = "STANDARD"

    versioning {
        enabled = true
    }

    encryption {
        default_kms_key_name = google_kms_crypto_key.tf_state_bucket.id
    }

    lifecycle {
        prevent_destroy = true
    }
}
