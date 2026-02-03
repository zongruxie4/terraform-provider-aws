---
subcategory: "EventBridge"
layout: "aws"
page_title: "AWS: aws_events_rule"
description: |-
  Lists EventBridge Rule resources.
---

# List Resource: aws_events_rule

Lists EventBridge Rule resources.

## Example Usage

```terraform
list "aws_events_rule" "example" {
  provider = aws
}
```

## Argument Reference

This list resource supports the following arguments:

* `region` - (Optional) Region to query. Defaults to provider region.
