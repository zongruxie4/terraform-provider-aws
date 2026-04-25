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
data "aws_securityhub_security_controls" "example" {}
```

## Argument Reference

This data source supports the following arguments:

* `region` - (Optional) Region where this resource will be [managed](https://docs.aws.amazon.com/general/latest/gr/rande.html#regional-endpoints). Defaults to the Region set in the [provider configuration](https://registry.terraform.io/providers/hashicorp/aws/latest/docs#aws-configuration-reference).

## Attribute Reference

This data source exports the following attributes in addition to the arguments above:

* `security_control_definitions` - Metadata for security controls. See below for details.

### `security_control_definitions`

* `title` - Title of a security control.
