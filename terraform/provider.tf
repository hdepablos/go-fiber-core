terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
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
      apigateway     = "http://localhost:4566"
      cloudwatch     = "http://localhost:4566"
      lambda         = "http://localhost:4566"
      s3             = "http://s3.localhost.localstack.cloud:4566"
      sqs            = "http://localhost:4566"
      eventbridge    = "http://localhost:4566"
      iam            = "http://localhost:4566"
      cloudwatchlogs = "http://localhost:4566"
      sts            = "http://localhost:4566"
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
