<!-- Copyright IBM Corp. 2014, 2026 -->
<!-- SPDX-License-Identifier: MPL-2.0 -->

# Do Not Implement Import Support for Exclusive Resource Types

**Summary:** Exclusive resource types should not support import because import can give the wrong impression about what the resource manages and can lead users into choosing exclusive behavior by accident.
**Created**: 2026-05-04
**Author**: [@taruntej-a](https://github.com/taruntej-a)

---

## Background

Exclusive resource types manage a whole set of relationships, not just one remote object. Their job is to make the full set of relationships in AWS match the Terraform configuration by adding missing relationships and removing ones that are not configured. This pattern is described in [Exclusive Relationship Management Resources](./exclusive-relationship-management-resources.md).

That creates a design question for import support. In most cases, import is understood as a way to bring an existing remote object under Terraform management. For exclusive resources, import would do more than that because these resources manage a whole relationship set, not a single object.

The provider also has a mix of newer exclusive resources that use the `_exclusive` suffix and older resources with the same behavior that were created before that naming pattern existed. Newer resources are easier to find because the `_exclusive` suffix makes their behavior more obvious. A few older exclusive resources do not have that suffix. One example is `aws_iam_policy_attachment`. Because of that, contributors can run into an exclusive resource and reasonably assume import should be added, since the resource name does not clearly show that it manages the full relationship set.

This led to a broader question for the provider: should exclusive resource types support import at all?

## Decision

The Terraform AWS Provider will not implement import support for exclusive resource types.

This applies both to newer resources with the `_exclusive` suffix and to older resource types with the same behavior.

Importing one of these resources would not just record existing infrastructure in Terraform state. It would also opt the user into a model that can add or remove relationships in ways they may not expect.

That creates real risk. A user may think they have only adopted existing infrastructure, but the next apply could remove relationships that are not managed in the configuration. Because of that risk, import support should not be added for exclusive resource types.

## Consequences/Future Work

- Contributors should not add import support to exclusive resource types, and existing or future work on resource identity for these resources should not be used to enable import.
- Older exclusive resources without the `_exclusive` suffix should be documented clearly so their behavior is not mistaken for standard attachment resources.
