---
subcategory: "EBS (EC2)"
layout: "aws"
page_title: "AWS: aws_ebs_volume_copy"
description: |-
  Copies an existing Amazon EBS volume.
---

# Resource: aws_ebs_volume_copy

Creates a copy of an existing Amazon EBS volume.

## Example Usage

```terraform
resource "aws_ebs_volume" "source" {
  availability_zone = "us-west-2a"
  size              = 8
}

resource "aws_ebs_volume_copy" "example" {
  source_volume_id = aws_ebs_volume.source.id
  volume_type      = "gp3"
  size             = 20
  iops             = 3000
  throughput       = 125

  tags = {
    Name = "example-copy"
  }
}
```

## Argument Reference

This resource supports the following arguments:

- `region` - (Optional) Region where this resource will be [managed](https://docs.aws.amazon.com/general/latest/gr/rande.html#regional-endpoints). Defaults to the Region set in the [provider configuration](https://registry.terraform.io/providers/hashicorp/aws/latest/docs#aws-configuration-reference). This must match the Region of the source EBS volume referenced by `source_volume_id`.
- `source_volume_id` - (Optional) ID of the source EBS volume to copy. This argument must be set when creating a new EBS volume copy. Changing this value forces replacement of the resource.
- `size` - (Optional) Size of the copied volume, in GiB.
- `iops` - (Optional) Provisioned IOPS for the copied volume. Use only with volume types that support provisioned IOPS, such as `gp3`.
- `throughput` - (Optional) Throughput for the copied volume, in MiB/s. Valid only when `volume_type` is `gp3`.
- `volume_type` - (Optional) Type of the copied EBS volume. Valid values include `gp2`, `gp3`, `io1`, `io2`, `sc1`, `st1`, and `standard`.
- `tags` - (Optional) A map of tags to assign to the copied volume. If configured with a provider [`default_tags` configuration block](https://registry.terraform.io/providers/hashicorp/aws/latest/docs#default_tags-configuration-block) present, tags with matching keys will overwrite those defined at the provider-level.

~> **NOTE:** When changing the `size`, `iops` or `type` of a volume, there are [considerations](https://docs.aws.amazon.com/ebs/latest/userguide/ebs-volume-types.html) to be aware of.

## Attribute Reference

This resource exports the following attributes in addition to the arguments above:

- `arn` - Amazon Resource Name (ARN) of the copied EBS volume.
- `availability_zone` - Availability Zone for the copied volume.
- `id` - ID of the copied EBS volume.
- `source_volume_id` - ID of the source EBS volume from which the copy was created.
- `tags_all` - A map of tags assigned to the resource, including those inherited from the provider [`default_tags` configuration block](https://registry.terraform.io/providers/hashicorp/aws/latest/docs#default_tags-configuration-block).

## Timeouts

[Configuration options](https://developer.hashicorp.com/terraform/language/resources/syntax#operation-timeouts):

- `create` - (Default `5m`)
- `update` - (Default `5m`)
- `delete` - (Default `10m`)

## Import

In Terraform v1.5.0 and later, use an [`import` block](https://developer.hashicorp.com/terraform/language/import) to import EBS volume copies using the copied volume ID. For example:

```terraform
import {
  to = aws_ebs_volume_copy.example
  id = "vol-049df61146c4d7901"
}
```

Using `terraform import`, import EBS volume copies using the copied volume ID. For example:

```console
% terraform import aws_ebs_volume_copy.example vol-049df61146c4d7901
```
