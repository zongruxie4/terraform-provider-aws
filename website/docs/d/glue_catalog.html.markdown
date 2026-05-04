---
subcategory: "Glue"
layout: "aws"
page_title: "AWS: aws_glue_catalog"
description: |-
  Provides details about an AWS Glue Catalog.
---

# Data Source: aws_glue_catalog

Provides details about an AWS Glue Catalog.

## Example Usage

```terraform
data "aws_glue_catalog" "example" {
  name = "example"
}
```

## Argument Reference

This data source supports the following arguments:

* `region` - (Optional) Region where this resource will be [managed](https://docs.aws.amazon.com/general/latest/gr/rande.html#regional-endpoints). Defaults to the Region set in the [provider configuration](https://registry.terraform.io/providers/hashicorp/aws/latest/docs#aws-configuration-reference).
* `name` - (Required) Name of the catalog to look up.

## Attribute Reference

This data source exports the following attributes in addition to the arguments above:

* `allow_full_table_external_data_access` - Whether third-party engines can access data in Amazon S3 locations that are registered with Lake Formation.
* `arn` - ARN of the Glue Catalog.
* `catalog_id` - The ID of the parent catalog.
* `create_time` - The time at which the catalog was created.
* `description` - Description of the catalog.
* `parameters` - Map of key-value pairs that define parameters and properties of the catalog.
* `tags` - Key-value map of resource tags.
* `update_time` - The time at which the catalog was last updated.

The following nested blocks are also exported:

### catalog_properties

* `custom_properties` - Map of custom key-value pairs for the catalog properties.
* `data_lake_access_properties` - Data lake access properties. See below.
* `iceberg_optimization_properties` - Iceberg optimization properties. See below.

#### data_lake_access_properties

* `catalog_type` - The type of the catalog.
* `data_lake_access` - Whether data lake access is enabled.
* `data_transfer_role` - The ARN of the IAM role used for data transfer.
* `kms_key` - The ARN of the KMS key used for encryption.
* `managed_workgroup_name` - The managed workgroup name.
* `managed_workgroup_status` - The managed workgroup status.
* `redshift_database_name` - The Redshift database name.
* `status_message` - A status message.

#### iceberg_optimization_properties

* `iceberg_retention_policy_enabled` - Whether Iceberg retention policy optimization is enabled.
* `iceberg_unreferenced_file_removal_enabled` - Whether Iceberg unreferenced file removal optimization is enabled.

### federated_catalog

* `connection_name` - The name of the connection to the external metastore.
* `connection_type` - The type of connection used to access the federated catalog.
* `identifier` - A unique identifier for the federated catalog.

### target_redshift_catalog

* `catalog_arn` - The ARN of the target Redshift catalog.

### create_database_default_permissions

* `permissions` - The permissions that are granted to the principal.
* `principal` - The principal who is granted permissions. See below.

### create_table_default_permissions

* `permissions` - The permissions that are granted to the principal.
* `principal` - The principal who is granted permissions. See below.

#### principal

* `data_lake_principal_identifier` - An identifier for the Lake Formation principal.
