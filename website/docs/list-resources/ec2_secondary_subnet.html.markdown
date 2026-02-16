---
subcategory: "EC2 (Elastic Compute Cloud)"
layout: "aws"
page_title: "AWS: aws_ec2_secondary_subnet"
description: |-
  Lists EC2 (Elastic Compute Cloud) Secondary Subnet resources.
---

# List Resource: aws_ec2_secondary_subnet

Lists EC2 (Elastic Compute Cloud) Secondary Subnet resources.

## Example Usage

```terraform
list "aws_ec2_secondary_subnet" "example" {
  provider = aws
}
```

## Argument Reference

This list resource supports the following arguments:

* `region` - (Optional) Region to query. Defaults to provider region.
