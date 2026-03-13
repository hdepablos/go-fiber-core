terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.12"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.24"
    }
  }
}

provider "aws" {
  region = var.aws_region

  access_key                  = var.environment == "local" ? "test" : null
  secret_key                  = var.environment == "local" ? "test" : null
  skip_credentials_validation = var.environment == "local"
  skip_metadata_api_check     = var.environment == "local"
  skip_requesting_account_id  = var.environment == "local"

  dynamic "endpoints" {
    for_each = var.environment == "local" ? [1] : []
    content {
      apigateway     = var.localstack_endpoint_base
      cloudwatch     = var.localstack_endpoint_base
      lambda         = var.localstack_endpoint_base
      s3             = var.localstack_s3_endpoint
      sqs            = var.localstack_endpoint_base
      eventbridge    = var.localstack_endpoint_base
      iam            = var.localstack_endpoint_base
      cloudwatchlogs = var.localstack_endpoint_base
      sts            = var.localstack_endpoint_base
      dynamodb       = var.localstack_endpoint_base
    }
  }

  default_tags {
    tags = {
      Project     = "GoFiberCore"
      Environment = var.environment
      ManagedBy   = "Terraform"
    }
  }
}

provider "helm" {
  kubernetes {
    config_path    = "~/.kube/config"
    config_context = var.environment == "local" ? var.kube_context : null
  }
}

provider "kubernetes" {
  config_path    = "~/.kube/config"
  config_context = var.environment == "local" ? var.kube_context : null
}
