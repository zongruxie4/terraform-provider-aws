---
subcategory: "Security Hub"
layout: "aws"
page_title: "AWS: aws_securityhub_security_controls"
description: |-
  Lists all of the security controls that apply to a specified standard.
---

# Data Source: aws_securityhub_security_controls

Lists all of the security controls that apply to a specified standard.

## Example Usage

```terraform
data "aws_securityhub_security_controls" "example" {
  current_region_availability = "AVAILABLE"
}
```

## Argument Reference

This data source supports the following arguments:

* `current_region_availability` - (Optional) Whether a security control is available in the current AWS Region. Valid values: `AVAILABLE`, `UNAVAILABLE`.
* `region` - (Optional) Region where this resource will be [managed](https://docs.aws.amazon.com/general/latest/gr/rande.html#regional-endpoints). Defaults to the Region set in the [provider configuration](https://registry.terraform.io/providers/hashicorp/aws/latest/docs#aws-configuration-reference).
* `severity_rating` - (Optional) Severity of a security control. Valid values: `LOW`, `MEDIUM`, `HIGH`, `CRITICAL`.
* `standards_arn` - (Optional) ARN of the standard that you want to list controls for.

## Attribute Reference

This data source exports the following attributes in addition to the arguments above:

* `security_control_ids` - List of security control IDs.

