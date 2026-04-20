---
subcategory: "EC2 (Elastic Compute Cloud)"
layout: "aws"
page_title: "AWS: aws_launch_template"
description: |-
  Lists EC2 Launch Template resources.
---

# List Resource: aws_launch_template

Lists EC2 Launch Template resources.

## Example Usage

```terraform
list "aws_launch_template" "example" {
  provider = aws
}
```

## Argument Reference

This list resource supports the following arguments:

* `region` - (Optional) Region to query. Defaults to provider region.
