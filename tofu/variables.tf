variable credentials_file {
    description = "The path to the service account credentials file"
    type = string
}

variable project_id {
    description = "The project ID"
    type = string
}

variable region {
    description = "The region to deploy resources to"
    type = string
}

variable zone {
    description = "The zone to deploy resources to"
    type = string
}

variable repository {
    description = "The GitHub repository name"
    type = string
}

variable remote_profile_url {
    description = "The URL to the remote profile JSON file"
    type = string
}