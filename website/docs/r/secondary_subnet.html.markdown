---
subcategory: "EC2 (Elastic Compute Cloud)"
layout: "aws"
page_title: "AWS: aws_secondary_subnet"
description: |-
  Provides an EC2 Secondary Subnet resource.
---

# Resource: aws_secondary_subnet

Provides an EC2 Secondary Subnet resource.

A secondary subnet is a subnet within a secondary network that provides high-performance networking capabilities for specialized workloads such as RDMA (Remote Direct Memory Access) applications.

## Example Usage

### Basic Usage

```terraform
resource "aws_secondary_network" "example" {
  ipv4_cidr_block = "10.0.0.0/16"
  network_type    = "rdma"

  tags = {
    Name = "example-secondary-network"
  }
}

resource "aws_secondary_subnet" "example" {
  secondary_network_id = aws_secondary_network.example.id
  ipv4_cidr_block      = "10.0.1.0/24"
  availability_zone    = "us-west-2a"

  tags = {
    Name = "example-secondary-subnet"
  }
}
```

### Using Availability Zone ID

```terraform
data "aws_availability_zones" "available" {
  state = "available"

  filter {
    name   = "opt-in-status"
    values = ["opt-in-not-required"]
  }
}

resource "aws_secondary_network" "example" {
  ipv4_cidr_block = "10.0.0.0/16"
  network_type    = "rdma"

  tags = {
    Name = "example-secondary-network"
  }
}

resource "aws_secondary_subnet" "example" {
  secondary_network_id = aws_secondary_network.example.id
  ipv4_cidr_block      = "10.0.1.0/24"
  availability_zone_id = data.aws_availability_zones.available.zone_ids[0]

  tags = {
    Name = "example-secondary-subnet"
  }
}
```

## Argument Reference

This resource supports the following arguments:

* `secondary_network_id` - (Required) The ID of the secondary network in which to create the secondary subnet.
* `ipv4_cidr_block` - (Required) The IPv4 CIDR block for the secondary subnet. The CIDR block size must be between /12 and /28.
* `availability_zone` - (Optional) The Availability Zone for the secondary subnet. Cannot be specified with `availability_zone_id`.
* `availability_zone_id` - (Optional) The ID of the Availability Zone for the secondary subnet. This option is preferred over `availability_zone` as it provides a consistent identifier across AWS accounts. Cannot be specified with `availability_zone`.
* `tags` - (Optional) A map of tags to assign to the resource. If configured with a provider [`default_tags` configuration block](https://registry.terraform.io/providers/hashicorp/aws/latest/docs#default_tags-configuration-block) present, tags with matching keys will overwrite those defined at the provider-level.

## Attribute Reference

This resource exports the following attributes in addition to the arguments above:

* `arn` - The ARN of the secondary subnet.
* `id` - The ID of the secondary subnet.
* `owner_id` - The ID of the AWS account that owns the secondary subnet.
* `secondary_network_type` - The type of the secondary network (e.g., `rdma`).
* `secondary_subnet_id` - The ID of the secondary subnet.
* `state` - The state of the secondary subnet.
* `tags_all` - A map of tags assigned to the resource, including those inherited from the provider [`default_tags` configuration block](https://registry.terraform.io/providers/hashicorp/aws/latest/docs#default_tags-configuration-block).

The following attributes are exported in the `ipv4_cidr_block_associations` block:

* `association_id` - The association ID for the IPv4 CIDR block.
* `cidr_block` - The IPv4 CIDR block.
* `state` - The state of the IPv4 CIDR block association.

## Timeouts

[Configuration options](https://developer.hashicorp.com/terraform/language/resources/syntax#operation-timeouts):

* `create` - (Default `30m`)
* `update` - (Default `30m`)
* `delete` - (Default `30m`)

## Import

In Terraform v1.5.0 and later, use an [`import` block](https://developer.hashicorp.com/terraform/language/import) to import EC2 Secondary Subnets using the secondary subnet ID. For example:

```terraform
import {
  to = aws_secondary_subnet.example
  id = "ss-0123456789abcdef0"
}
```

Using `terraform import`, import EC2 Secondary Subnets using the secondary subnet ID. For example:

```console
% terraform import aws_secondary_subnet.example ss-0123456789abcdef0
```