---
subcategory: "Outposts"
layout: "aws"
page_title: "AWS: aws_outposts_capacity_task"
description: |-
  Terraform resource for managing an AWS Outposts Capacity Task.
---

# Resource: aws_outposts_capacity_task

Terraform resource for managing an AWS Outposts Capacity Task.

A capacity task redistributes the instance pools available on an Outpost rack or server to match the `instance_pool` configuration declared in the resource. Starting a capacity task is a long-running, asynchronous operation — Terraform waits for it to reach a terminal state (`COMPLETED`, `CANCELLED`, or `FAILED`) before finishing the apply.

## Example Usage

### Minimal

```terraform
data "aws_outposts_outposts" "example" {}

resource "aws_outposts_capacity_task" "example" {
  outpost_identifier = tolist(data.aws_outposts_outposts.example.arns)[0]

  instance_pool {
    instance_type = "m5.large"
    count         = 2
  }
}
```

### Multiple instance pools, excluded instances, and a specified blocking-instance action

```terraform
resource "aws_outposts_capacity_task" "example" {
  outpost_identifier                = "op-1234567890abcdef"
  task_action_on_blocking_instances = "WAIT_FOR_EVACUATION"

  instance_pool {
    instance_type = "m5.large"
    count         = 4
  }

  instance_pool {
    instance_type = "c5.xlarge"
    count         = 2
  }

  # Instance IDs the capacity task must not stop when re-balancing capacity.
  instances_to_exclude {
    instances = ["i-0123456789abcdef0", "i-0fedcba9876543210"]
  }

  timeouts {
    create = "90m"
    delete = "15m"
  }
}
```

## Argument Reference

This resource supports the following arguments:

* `region` - (Optional) Region where this resource will be [managed](https://docs.aws.amazon.com/general/latest/gr/rande.html#regional-endpoints). Defaults to the Region set in the [provider configuration](https://registry.terraform.io/providers/hashicorp/aws/latest/docs#aws-configuration-reference).
* `outpost_identifier` - (Required) ID or ARN of the Outpost on which to run the capacity task. Both forms are accepted; the provider normalizes the value internally. Changing this value forces a new resource.
* `instance_pool` - (Required) One or more `instance_pool` blocks defining the desired instance-type layout for the Outpost. See [below](#instance_pool). At least one block is required. Changing any value forces a new resource.
* `order_id` - (Optional) ID of the Amazon Web Services Outposts order associated with the capacity task. Changing this value forces a new resource.
* `task_action_on_blocking_instances` - (Optional) Action to take if running instances block the capacity task. Valid values are `WAIT_FOR_EVACUATION` and `FAIL_TASK`. Changing this value forces a new resource.
* `instances_to_exclude` - (Optional) Single `instances_to_exclude` block specifying user-owned running instances that must not be stopped to free up capacity. See [below](#instances_to_exclude). Note: AWS does not return this value via the Get/Describe API; after import, you must add the block back to your configuration manually — see [Import](#import).
* `timeouts` - (Optional) Configuration block with timeouts. See [below](#timeouts).

### instance_pool

* `instance_type` - (Required) Instance type for this pool entry. Must be an instance type supported by the target Outpost. Changing this value forces a new resource.
* `count` - (Required) Number of instances of `instance_type` that should be present after the task completes. Must be at least `1`. Changing this value forces a new resource.

### instances_to_exclude

* `instances` - (Required) Set of EC2 instance IDs (of user-owned instances running on the Outpost) that the capacity task must not stop. At least one instance ID is required.

### timeouts

[Configuration options](https://developer.hashicorp.com/terraform/language/resources/syntax#operation-timeouts):

* `create` - (Default `60m`)
* `delete` - (Default `10m`)

## Attribute Reference

This resource exports the following attributes in addition to the arguments above:

* `capacity_task_id` - ID assigned by AWS to the capacity task (for example, `cap-1a2b3c4d5e6f7g8h9`).
* `status` - Current status of the capacity task. One of `REQUESTED`, `IN_PROGRESS`, `WAITING_FOR_EVACUATION`, `CANCELLATION_IN_PROGRESS`, `COMPLETED`, `CANCELLED`, or `FAILED`. See the [AWS documentation](https://docs.aws.amazon.com/outposts/latest/APIReference/API_GetCapacityTask.html) for semantics.
* `creation_date` - RFC 3339 timestamp at which the capacity task was created.
* `completion_date` - RFC 3339 timestamp at which the capacity task reached a terminal state (if any).
* `failure_reason` - Human-readable reason reported by AWS when the capacity task failed. `null` unless the terminal state is `FAILED`.

## Lifecycle

Because every argument of this resource is marked as forces-new, any change to the configuration results in destroying and re-creating the capacity task. Tasks that are already in a terminal state (`COMPLETED` or `CANCELLED`) are left in place on destroy and only removed from Terraform state; tasks still in flight are cancelled and Terraform waits for them to reach `CANCELLED`. If a task reaches the terminal state `FAILED` during `delete`, the provider tolerates the "already in a terminal state" error returned by `CancelCapacityTask` and considers the resource successfully destroyed.

If a create operation produces a `FAILED` task, the resource is not written to Terraform state (the `failure_reason` is surfaced in the diagnostic instead), so no follow-up destroy is required.

## Import

In Terraform v1.12.0 and later, the [`import` block](https://developer.hashicorp.com/terraform/language/import) can be used with the `identity` attribute:

```terraform
import {
  to = aws_outposts_capacity_task.example
  identity = {
    outpost_identifier = "op-1234567890abcdef"
    capacity_task_id   = "cap-1a2b3c4d5e6f7g8h9"
  }
}

resource "aws_outposts_capacity_task" "example" {
  ### Configuration omitted for brevity ###
}
```

### Identity Schema

#### Required

* `outpost_identifier` (String) Outpost identifier supplied when the task was created (ID or ARN).
* `capacity_task_id` (String) AWS-assigned capacity task ID.

#### Optional

* `account_id` (String) AWS Account where this resource is managed.
* `region` (String) Region where this resource is managed.

In Terraform v1.5.0 and later, use an [`import` block](https://developer.hashicorp.com/terraform/language/import) to import a Capacity Task using the `outpost_identifier` and `capacity_task_id` joined by a forward slash (`/`). For example:

```terraform
import {
  to = aws_outposts_capacity_task.example
  id = "op-1234567890abcdef/cap-1a2b3c4d5e6f7g8h9"
}
```

Using `terraform import`, import a Capacity Task using the same composite ID:

```console
% terraform import aws_outposts_capacity_task.example op-1234567890abcdef/cap-1a2b3c4d5e6f7g8h9
```

**Note:** `instances_to_exclude` is write-only and is not returned by the AWS API. After importing a capacity task, add the `instances_to_exclude` block back to your configuration manually if it was set at creation time — otherwise Terraform will show a drift the first time you plan.
