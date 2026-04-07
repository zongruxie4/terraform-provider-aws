---
subcategory: "Config"
layout: "aws"
page_title: "AWS: aws_configservice_config_rule"
description: |-
  Lists Config Config Rule resources.
---

# List Resource: aws_configservice_config_rule

Lists Config Config Rule resources.

## Example Usage

```terraform
list "aws_configservice_config_rule" "example" {
  provider = aws
}
```

## Argument Reference

This list resource supports the following arguments:

* `region` - (Optional) Region to query. Defaults to provider region.
