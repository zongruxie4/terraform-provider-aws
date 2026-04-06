---
subcategory: "Security Hub"
layout: "aws"
page_title: "AWS: aws_securityhub_v2_account"
description: |-
  Enables Security Hub V2 for an AWS account.
---

# Resource: aws_securityhub_v2_account

Enables the unified Security Hub V2 for this AWS account.

~> **NOTE:** Destroying this resource will disable Security Hub V2 for this AWS account.

~> **NOTE:** This resource manages the unified Security Hub V2 service, which is distinct from the classic Security Hub CSPM managed by `aws_securityhub_account`. Both can coexist in the same account.

## Example Usage

### Basic

```terraform
resource "aws_securityhub_v2_account" "example" {}
```

### With Tags

```terraform
resource "aws_securityhub_v2_account" "example" {
  tags = {
    Environment = "production"
  }
}
```

## Argument Reference

This resource supports the following arguments:

* `tags` - (Optional) Map of tags to assign to the resource. If configured with a provider [`default_tags` configuration block](https://registry.terraform.io/providers/hashicorp/aws/latest/docs#default_tags-configuration-block) present, tags with matching keys will overwrite those defined at the provider-level.

## Attribute Reference

This resource exports the following attributes in addition to the arguments above:

* `hub_arn` - ARN of the Security Hub V2 resource created in the account.
* `id` - ARN of the Security Hub V2 resource.
* `tags_all` - Map of tags assigned to the resource, including those inherited from the provider [`default_tags` configuration block](https://registry.terraform.io/providers/hashicorp/aws/latest/docs#default_tags-configuration-block).

## Import

In Terraform v1.5.0 and later, use an [`import` block](https://developer.hashicorp.com/terraform/language/import) to import an existing Security Hub V2 enabled account. For example:

```terraform
import {
  to = aws_securityhub_v2_account.example
  id = "import"
}
```

Using `terraform import`, import an existing Security Hub V2 enabled account. For example:

```console
% terraform import aws_securityhub_v2_account.example import
```
