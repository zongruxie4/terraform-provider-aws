provider "aws" {
  region = "us-east-1"
}

data "aws_account_regions" "all" {
  region_opt_status_contains = ["ENABLED_BY_DEFAULT","DISABLED"]
}

output "all_regions" {
  value = data.aws_account_regions.all.regions
}