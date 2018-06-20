// Integration tests that validate S3-related code in AWS.
package aws

import (
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/gruntwork-io/terratest/modules/random"
)

func TestCreateAndDestroyS3Bucket(t *testing.T) {
	t.Parallel()

	region := GetRandomRegion(t, nil, nil)
	id := random.UniqueId()
	logger.Logf(t, "Random values selected. Region = %s, Id = %s\n", region, id)

	s3BucketName := "gruntwork-terratest-" + strings.ToLower(id)

	CreateS3Bucket(t, region, s3BucketName)
	DeleteS3Bucket(t, region, s3BucketName)
}

func TestAssertS3BucketExistsNoFalseNegative(t *testing.T) {
	t.Parallel()

	region := GetRandomRegion(t, nil, nil)
	s3BucketName := "gruntwork-terratest-" + strings.ToLower(random.UniqueId())
	logger.Logf(t, "Random values selected. Region = %s, s3BucketName = %s\n", region, s3BucketName)

	CreateS3Bucket(t, region, s3BucketName)
	defer DeleteS3Bucket(t, region, s3BucketName)

	AssertS3BucketExists(t, region, s3BucketName)
}

func TestAssertS3BucketExistsNoFalsePositive(t *testing.T) {
	t.Parallel()

	region := GetRandomRegion(t, nil, nil)
	s3BucketName := "gruntwork-terratest-" + strings.ToLower(random.UniqueId())
	logger.Logf(t, "Random values selected. Region = %s, s3BucketName = %s\n", region, s3BucketName)

	// We elect not to create the S3 bucket to confirm that our function correctly reports it doesn't exist.
	//aws.CreateS3Bucket(region, s3BucketName)

	err := AssertS3BucketExistsE(t, region, s3BucketName)
	if err == nil {
		t.Fatalf("Function claimed that S3 Bucket '%s' exists, but in fact it does not.", s3BucketName)
	}
}
