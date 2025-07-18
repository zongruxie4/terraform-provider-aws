// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/YakDriver/regexache"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/hashicorp/aws-sdk-go-base/v2/endpoints"
	"github.com/hashicorp/aws-sdk-go-base/v2/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/enum"
	"github.com/hashicorp/terraform-provider-aws/internal/errs/sdkdiag"
	"github.com/hashicorp/terraform-provider-aws/internal/flex"
	"github.com/hashicorp/terraform-provider-aws/internal/sdkv2"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
	itypes "github.com/hashicorp/terraform-provider-aws/internal/types"
	"github.com/hashicorp/terraform-provider-aws/internal/verify"
	"github.com/hashicorp/terraform-provider-aws/names"
	"github.com/mitchellh/go-homedir"
)

// @SDKResource("aws_s3_object", name="Object")
// @Tags(identifierAttribute="arn", resourceType="Object")
// @IdentityAttribute("bucket")
// @IdentityAttribute("key")
// @IdAttrFormat("{bucket}/{key}")
// @ImportIDHandler("objectImportID")
// @Testing(existsType="github.com/aws/aws-sdk-go-v2/service/s3;s3.GetObjectOutput")
// @Testing(importIgnore="force_destroy")
// @Testing(plannableImportAction="NoOp")
// @Testing(preIdentityVersion="6.0.0")
func resourceObject() *schema.Resource {
	return &schema.Resource{
		CreateWithoutTimeout: resourceObjectCreate,
		ReadWithoutTimeout:   resourceObjectRead,
		UpdateWithoutTimeout: resourceObjectUpdate,
		DeleteWithoutTimeout: resourceObjectDelete,

		CustomizeDiff: customdiff.Sequence(
			resourceObjectCustomizeDiff,
			func(ctx context.Context, d *schema.ResourceDiff, meta any) error {
				if ignoreProviderDefaultTags(ctx, d) {
					return d.SetNew(names.AttrTagsAll, d.Get(names.AttrTags))
				}
				return nil
			},
		),

		Schema: map[string]*schema.Schema{
			"acl": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				ValidateDiagFunc: enum.Validate[types.ObjectCannedACL](),
			},
			names.AttrARN: {
				Type:     schema.TypeString,
				Computed: true,
			},
			names.AttrBucket: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.NoZeroValues,
			},
			"bucket_key_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},
			"cache_control": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"checksum_algorithm": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: enum.Validate[types.ChecksumAlgorithm](),
			},
			"checksum_crc32": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"checksum_crc32c": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"checksum_crc64nvme": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"checksum_sha1": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"checksum_sha256": {
				Type:     schema.TypeString,
				Computed: true,
			},
			names.AttrContent: {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{names.AttrSource, "content_base64"},
			},
			"content_base64": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{names.AttrSource, names.AttrContent},
			},
			"content_disposition": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"content_encoding": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"content_language": {
				Type:     schema.TypeString,
				Optional: true,
			},
			names.AttrContentType: {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"etag": {
				Type: schema.TypeString,
				// This will conflict with SSE-C and SSE-KMS encryption and multi-part upload
				// if/when it's actually implemented. The Etag then won't match raw-file MD5.
				// See http://docs.aws.amazon.com/AmazonS3/latest/API/RESTCommonResponseHeaders.html
				Optional:      true,
				Computed:      true,
				ConflictsWith: []string{names.AttrKMSKeyID},
			},
			names.AttrForceDestroy: {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			names.AttrKey: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.NoZeroValues,
			},
			names.AttrKMSKeyID: {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: verify.ValidARN,
			},
			"metadata": {
				Type:         schema.TypeMap,
				Optional:     true,
				Elem:         &schema.Schema{Type: schema.TypeString},
				ValidateFunc: validateMetadataIsLowerCase,
			},
			"object_lock_legal_hold_status": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: enum.Validate[types.ObjectLockLegalHoldStatus](),
			},
			"object_lock_mode": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: enum.Validate[types.ObjectLockMode](),
			},
			"object_lock_retain_until_date": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsRFC3339Time,
			},
			"override_provider": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"default_tags": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									names.AttrTags: {
										Type:             schema.TypeMap,
										Optional:         true,
										Elem:             &schema.Schema{Type: schema.TypeString},
										ValidateDiagFunc: verify.MapSizeBetween(0, 0),
									},
								},
							},
						},
					},
				},
			},
			"server_side_encryption": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				ValidateDiagFunc: enum.Validate[types.ServerSideEncryption](),
			},
			names.AttrSource: {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{names.AttrContent, "content_base64"},
			},
			"source_hash": {
				Type:     schema.TypeString,
				Optional: true,
			},
			names.AttrStorageClass: {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				ValidateDiagFunc: enum.Validate[types.ObjectStorageClass](),
			},
			names.AttrTags:    tftags.TagsSchema(),
			names.AttrTagsAll: tftags.TagsSchemaComputed(),
			"version_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"website_redirect": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceObjectCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var diags diag.Diagnostics
	return append(diags, resourceObjectUpload(ctx, d, meta)...)
}

func resourceObjectRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var diags diag.Diagnostics
	conn := meta.(*conns.AWSClient).S3Client(ctx)

	bucket := d.Get(names.AttrBucket).(string)
	if isDirectoryBucket(bucket) {
		conn = meta.(*conns.AWSClient).S3ExpressClient(ctx)
	}

	var optFns []func(*s3.Options)
	// Via S3 access point: "Invalid configuration: region from ARN `us-east-1` does not match client region `aws-global` and UseArnRegion is `false`".
	if arn.IsARN(bucket) && conn.Options().Region == endpoints.AwsGlobalRegionID {
		optFns = append(optFns, func(o *s3.Options) { o.UseARNRegion = true })
	}

	key := sdkv1CompatibleCleanKey(d.Get(names.AttrKey).(string))
	output, err := findObjectByBucketAndKey(ctx, conn, bucket, key, "", d.Get("checksum_algorithm").(string), optFns...)

	if !d.IsNewResource() && tfresource.NotFound(err) {
		log.Printf("[WARN] S3 Object (%s) not found, removing from state", d.Id())
		d.SetId("")
		return diags
	}

	if err != nil {
		return sdkdiag.AppendErrorf(diags, "reading S3 Object (%s): %s", d.Id(), err)
	}

	arn, err := newObjectARN(meta.(*conns.AWSClient).Partition(ctx), bucket, key)
	if err != nil {
		return sdkdiag.AppendErrorf(diags, "reading S3 Object (%s): %s", d.Id(), err)
	}
	d.Set(names.AttrARN, arn.String())

	d.Set("bucket_key_enabled", output.BucketKeyEnabled)
	d.Set("cache_control", output.CacheControl)
	d.Set("checksum_crc32", output.ChecksumCRC32)
	d.Set("checksum_crc32c", output.ChecksumCRC32C)
	d.Set("checksum_crc64nvme", output.ChecksumCRC64NVME)
	d.Set("checksum_sha1", output.ChecksumSHA1)
	d.Set("checksum_sha256", output.ChecksumSHA256)
	d.Set("content_disposition", output.ContentDisposition)
	d.Set("content_encoding", output.ContentEncoding)
	d.Set("content_language", output.ContentLanguage)
	d.Set(names.AttrContentType, output.ContentType)
	// See https://forums.aws.amazon.com/thread.jspa?threadID=44003
	d.Set("etag", strings.Trim(aws.ToString(output.ETag), `"`))
	if output.SSEKMSKeyId != nil { // nosemgrep:ci.helper-schema-ResourceData-Set-extraneous-nil-check
		d.Set(names.AttrKMSKeyID, output.SSEKMSKeyId)
	}
	d.Set("metadata", output.Metadata)
	d.Set("object_lock_legal_hold_status", output.ObjectLockLegalHoldStatus)
	d.Set("object_lock_mode", output.ObjectLockMode)
	d.Set("object_lock_retain_until_date", flattenObjectDate(output.ObjectLockRetainUntilDate))
	d.Set("server_side_encryption", output.ServerSideEncryption)
	// The "STANDARD" (which is also the default) storage
	// class when set would not be included in the results.
	d.Set(names.AttrStorageClass, types.ObjectStorageClassStandard)
	if output.StorageClass != "" {
		d.Set(names.AttrStorageClass, output.StorageClass)
	}
	d.Set("version_id", output.VersionId)
	d.Set("website_redirect", output.WebsiteRedirectLocation)

	return diags
}

func resourceObjectUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var diags diag.Diagnostics
	if hasObjectContentChanges(d) {
		return append(diags, resourceObjectUpload(ctx, d, meta)...)
	}

	conn := meta.(*conns.AWSClient).S3Client(ctx)

	bucket := d.Get(names.AttrBucket).(string)
	if isDirectoryBucket(bucket) {
		conn = meta.(*conns.AWSClient).S3ExpressClient(ctx)
	}

	var optFns []func(*s3.Options)
	// Via S3 access point: "Invalid configuration: region from ARN `us-east-1` does not match client region `aws-global` and UseArnRegion is `false`".
	if arn.IsARN(bucket) && conn.Options().Region == endpoints.AwsGlobalRegionID {
		optFns = append(optFns, func(o *s3.Options) { o.UseARNRegion = true })
	}

	key := sdkv1CompatibleCleanKey(d.Get(names.AttrKey).(string))

	if d.HasChange("acl") {
		input := &s3.PutObjectAclInput{
			ACL:    types.ObjectCannedACL(d.Get("acl").(string)),
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		}

		_, err := conn.PutObjectAcl(ctx, input, optFns...)

		if err != nil {
			return sdkdiag.AppendErrorf(diags, "putting S3 Object (%s) ACL: %s", d.Id(), err)
		}
	}

	if d.HasChange("object_lock_legal_hold_status") {
		input := &s3.PutObjectLegalHoldInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
			LegalHold: &types.ObjectLockLegalHold{
				Status: types.ObjectLockLegalHoldStatus(d.Get("object_lock_legal_hold_status").(string)),
			},
		}

		_, err := conn.PutObjectLegalHold(ctx, input, optFns...)

		if err != nil {
			return sdkdiag.AppendErrorf(diags, "putting S3 Object (%s) legal hold: %s", d.Id(), err)
		}
	}

	if d.HasChanges("object_lock_mode", "object_lock_retain_until_date") {
		input := &s3.PutObjectRetentionInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
			Retention: &types.ObjectLockRetention{
				Mode:            types.ObjectLockRetentionMode(d.Get("object_lock_mode").(string)),
				RetainUntilDate: expandObjectDate(d.Get("object_lock_retain_until_date").(string)),
			},
		}

		// Bypass required to lower or clear retain-until date.
		if d.HasChange("object_lock_retain_until_date") {
			oraw, nraw := d.GetChange("object_lock_retain_until_date")
			o, n := expandObjectDate(oraw.(string)), expandObjectDate(nraw.(string))

			if n == nil || (o != nil && n.Before(*o)) {
				input.BypassGovernanceRetention = aws.Bool(true)
			}
		}

		_, err := conn.PutObjectRetention(ctx, input, optFns...)

		if err != nil {
			return sdkdiag.AppendErrorf(diags, "putting S3 Object (%s) retention: %s", d.Id(), err)
		}
	}

	return append(diags, resourceObjectRead(ctx, d, meta)...)
}

func resourceObjectDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var diags diag.Diagnostics
	conn := meta.(*conns.AWSClient).S3Client(ctx)

	bucket := d.Get(names.AttrBucket).(string)
	if isDirectoryBucket(bucket) {
		conn = meta.(*conns.AWSClient).S3ExpressClient(ctx)
	}

	var optFns []func(*s3.Options)
	// Via S3 access point: "Invalid configuration: region from ARN `us-east-1` does not match client region `aws-global` and UseArnRegion is `false`".
	if arn.IsARN(bucket) && conn.Options().Region == endpoints.AwsGlobalRegionID {
		optFns = append(optFns, func(o *s3.Options) { o.UseARNRegion = true })
	}

	key := sdkv1CompatibleCleanKey(d.Get(names.AttrKey).(string))

	var err error
	if _, ok := d.GetOk("version_id"); ok {
		_, err = deleteAllObjectVersions(ctx, conn, bucket, key, d.Get(names.AttrForceDestroy).(bool), false, optFns...)
	} else {
		err = deleteObjectVersion(ctx, conn, bucket, key, "", false, optFns...)
	}

	if err != nil {
		return sdkdiag.AppendErrorf(diags, "deleting S3 Bucket (%s) Object (%s): %s", bucket, key, err)
	}

	return diags
}

func resourceObjectUpload(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var diags diag.Diagnostics
	conn := meta.(*conns.AWSClient).S3Client(ctx)

	bucket := d.Get(names.AttrBucket).(string)
	if isDirectoryBucket(bucket) {
		conn = meta.(*conns.AWSClient).S3ExpressClient(ctx)
	}

	var optFns []func(*s3.Options)
	// Via S3 access point: "Invalid configuration: region from ARN `us-east-1` does not match client region `aws-global` and UseArnRegion is `false`".
	if arn.IsARN(bucket) && conn.Options().Region == endpoints.AwsGlobalRegionID {
		optFns = append(optFns, func(o *s3.Options) { o.UseARNRegion = true })
	}

	var body io.ReadSeeker

	if v, ok := d.GetOk(names.AttrSource); ok {
		source := v.(string)
		path, err := homedir.Expand(source)
		if err != nil {
			return sdkdiag.AppendErrorf(diags, "expanding homedir in source (%s): %s", source, err)
		}
		file, err := os.Open(path)
		if err != nil {
			return sdkdiag.AppendErrorf(diags, "opening S3 object source (%s): %s", path, err)
		}

		body = file
		defer func() {
			err := file.Close()
			if err != nil {
				log.Printf("[WARN] Error closing S3 object source (%s): %s", path, err)
			}
		}()
	} else if v, ok := d.GetOk(names.AttrContent); ok {
		body = strings.NewReader(v.(string))
	} else if v, ok := d.GetOk("content_base64"); ok {
		// We can't do streaming decoding here (with base64.NewDecoder) because
		// the AWS SDK requires an io.ReadSeeker but a base64 decoder can't seek.
		v, err := itypes.Base64Decode(v.(string))
		if err != nil {
			return sdkdiag.AppendFromErr(diags, err)
		}
		body = bytes.NewReader(v)
	} else {
		body = bytes.NewReader([]byte{})
	}

	input := &s3.PutObjectInput{
		Body:   body,
		Bucket: aws.String(bucket),
		Key:    aws.String(sdkv1CompatibleCleanKey(d.Get(names.AttrKey).(string))),
	}

	if v, ok := d.GetOk("acl"); ok {
		input.ACL = types.ObjectCannedACL(v.(string))
	}

	if v, ok := d.GetOk("bucket_key_enabled"); ok {
		input.BucketKeyEnabled = aws.Bool(v.(bool))
	}

	if v, ok := d.GetOk("cache_control"); ok {
		input.CacheControl = aws.String(v.(string))
	}

	if v, ok := d.GetOk("checksum_algorithm"); ok {
		input.ChecksumAlgorithm = types.ChecksumAlgorithm(v.(string))
	}

	if v, ok := d.GetOk("content_disposition"); ok {
		input.ContentDisposition = aws.String(v.(string))
	}

	if v, ok := d.GetOk("content_encoding"); ok {
		input.ContentEncoding = aws.String(v.(string))
	}

	if v, ok := d.GetOk("content_language"); ok {
		input.ContentLanguage = aws.String(v.(string))
	}

	if v, ok := d.GetOk(names.AttrContentType); ok {
		input.ContentType = aws.String(v.(string))
	}

	if v, ok := d.GetOk(names.AttrKMSKeyID); ok {
		input.SSEKMSKeyId = aws.String(v.(string))
		input.ServerSideEncryption = types.ServerSideEncryptionAwsKms
	}

	if v, ok := d.GetOk("metadata"); ok {
		input.Metadata = flex.ExpandStringValueMap(v.(map[string]any))
	}

	if v, ok := d.GetOk("object_lock_legal_hold_status"); ok {
		input.ObjectLockLegalHoldStatus = types.ObjectLockLegalHoldStatus(v.(string))
	}

	if v, ok := d.GetOk("object_lock_mode"); ok {
		input.ObjectLockMode = types.ObjectLockMode(v.(string))
	}

	if v, ok := d.GetOk("object_lock_retain_until_date"); ok {
		input.ObjectLockRetainUntilDate = expandObjectDate(v.(string))
	}

	if v, ok := d.GetOk("server_side_encryption"); ok {
		input.ServerSideEncryption = types.ServerSideEncryption(v.(string))
	}

	if v, ok := d.GetOk(names.AttrStorageClass); ok {
		input.StorageClass = types.StorageClass(v.(string))
	}

	defaultTagsConfig := meta.(*conns.AWSClient).DefaultTagsConfig(ctx)
	tags := tftags.New(ctx, getContextTags(ctx))
	if ignoreProviderDefaultTags(ctx, d) {
		tags = tags.RemoveDefaultConfig(defaultTagsConfig)
	} else {
		tags = defaultTagsConfig.MergeTags(tftags.New(ctx, tags))
	}

	if len(tags) > 0 {
		// The tag-set must be encoded as URL Query parameters.
		input.Tagging = aws.String(tags.IgnoreAWS().URLEncode())
	}

	if v, ok := d.GetOk("website_redirect"); ok {
		input.WebsiteRedirectLocation = aws.String(v.(string))
	}

	if (input.ObjectLockLegalHoldStatus != "" || input.ObjectLockMode != "" || input.ObjectLockRetainUntilDate != nil) && input.ChecksumAlgorithm == "" {
		// "Content-MD5 OR x-amz-checksum- HTTP header is required for Put Object requests with Object Lock parameters".
		// AWS SDK for Go v1 transparently added a Content-MD4 header.
		input.ChecksumAlgorithm = types.ChecksumAlgorithmCrc32
	}

	uploader := manager.NewUploader(conn, manager.WithUploaderRequestOptions(optFns...))

	if _, err := uploader.Upload(ctx, input); err != nil {
		return sdkdiag.AppendErrorf(diags, "uploading S3 Object (%s) to Bucket (%s): %s", aws.ToString(input.Key), aws.ToString(input.Bucket), err)
	}

	if d.IsNewResource() {
		d.SetId(createObjectImportID(d))
	}

	return append(diags, resourceObjectRead(ctx, d, meta)...)
}

func validateMetadataIsLowerCase(v any, k string) (ws []string, errors []error) {
	value := v.(map[string]any)

	for k := range value {
		if k != strings.ToLower(k) {
			errors = append(errors, fmt.Errorf(
				"Metadata must be lowercase only. Offending key: %q", k))
		}
	}
	return
}

func resourceObjectCustomizeDiff(_ context.Context, d *schema.ResourceDiff, meta any) error {
	if hasObjectContentChanges(d) {
		return d.SetNewComputed("version_id")
	}

	if d.HasChange("source_hash") {
		d.SetNewComputed("version_id")
		d.SetNewComputed("etag")
	}

	return nil
}

func hasObjectContentChanges(d sdkv2.ResourceDiffer) bool {
	return slices.ContainsFunc([]string{
		"bucket_key_enabled",
		"cache_control",
		"checksum_algorithm",
		"content_base64",
		"content_disposition",
		"content_encoding",
		"content_language",
		names.AttrContentType,
		names.AttrContent,
		"etag",
		names.AttrKMSKeyID,
		"metadata",
		"server_side_encryption",
		names.AttrSource,
		"source_hash",
		names.AttrStorageClass,
		"website_redirect",
	}, d.HasChange)
}

func findObjectByBucketAndKey(ctx context.Context, conn *s3.Client, bucket, key, etag, checksumAlgorithm string, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	if checksumAlgorithm != "" {
		input.ChecksumMode = types.ChecksumModeEnabled
	}
	if etag != "" {
		input.IfMatch = aws.String(etag)
	}

	return findObject(ctx, conn, input, optFns...)
}

func findObject(ctx context.Context, conn *s3.Client, input *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	output, err := conn.HeadObject(ctx, input, optFns...)

	if tfawserr.ErrHTTPStatusCodeEquals(err, http.StatusNotFound) {
		return nil, &retry.NotFoundError{
			LastError:   err,
			LastRequest: input,
		}
	}

	if err != nil {
		return nil, err
	}

	if output == nil {
		return nil, tfresource.NewEmptyResultError(input)
	}

	return output, nil
}

func expandObjectDate(v string) *time.Time {
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return nil
	}

	return aws.Time(t)
}

func flattenObjectDate(t *time.Time) string {
	if t == nil {
		return ""
	}

	return t.Format(time.RFC3339)
}

// sdkv1CompatibleCleanKey returns an AWS SDK for Go v1 compatible clean key.
// DisableRestProtocolURICleaning was false on the standard S3Conn, so to ensure backwards
// compatibility we must "clean" the configured key before passing to AWS SDK for Go v2 APIs.
// See https://docs.aws.amazon.com/sdk-for-go/api/service/s3/#hdr-Automatic_URI_cleaning.
// See https://github.com/aws/aws-sdk-go/blob/cf903c8c543034654bb8f53b5f9d6454fdb2117f/private/protocol/rest/build.go#L247-L258.
func sdkv1CompatibleCleanKey(key string) string {
	// Remove leading './'.
	key = strings.TrimPrefix(key, "./")
	// We are effectively ignoring all leading '/'s and treating multiple '/'s as a single '/'.
	key = strings.TrimLeft(key, "/")
	key = regexache.MustCompile(`/+`).ReplaceAllString(key, "/")
	return key
}

func ignoreProviderDefaultTags(ctx context.Context, d sdkv2.ResourceDiffer) bool {
	if v, ok := d.GetOk("override_provider"); ok && len(v.([]any)) > 0 && v.([]any)[0] != nil {
		if data := expandOverrideProviderModel(ctx, v.([]any)[0].(map[string]any)); data != nil && data.DefaultTagsConfig != nil {
			return len(data.DefaultTagsConfig.Tags) == 0
		}
	}

	return false
}

type overrideProviderModel struct {
	DefaultTagsConfig *tftags.DefaultConfig
}

func expandOverrideProviderModel(ctx context.Context, tfMap map[string]any) *overrideProviderModel {
	if tfMap == nil {
		return nil
	}

	data := &overrideProviderModel{}

	if v, ok := tfMap["default_tags"].([]any); ok && len(v) > 0 {
		if v[0] != nil {
			data.DefaultTagsConfig = expandDefaultTags(ctx, v[0].(map[string]any))
		} else {
			// Ensure that DefaultTagsConfig is not nil as it's checked in ignoreProviderDefaultTags.
			data.DefaultTagsConfig = expandDefaultTags(ctx, map[string]any{})
		}
	}

	return data
}

func expandDefaultTags(ctx context.Context, tfMap map[string]any) *tftags.DefaultConfig {
	if tfMap == nil {
		return nil
	}

	data := &tftags.DefaultConfig{}

	if v, ok := tfMap[names.AttrTags].(map[string]any); ok {
		data.Tags = tftags.New(ctx, v)
	}

	return data
}

func createObjectImportID(d *schema.ResourceData) string {
	return fmt.Sprintf("%s/%s", d.Get(names.AttrBucket).(string), d.Get(names.AttrKey).(string))
}

type objectImportID struct{}

func (objectImportID) Create(d *schema.ResourceData) string {
	return createObjectImportID(d)
}

func (objectImportID) Parse(id string) (string, map[string]string, error) {
	id = strings.TrimPrefix(id, "s3://")

	bucket, key, found := strings.Cut(id, "/")
	if !found {
		return "", nil, fmt.Errorf("id \"%s\" should be in the format <bucket>/<key> or s3://<bucket>/<key>", id)
	}

	result := map[string]string{
		names.AttrBucket: bucket,
		names.AttrKey:    key,
	}
	return id, result, nil
}
