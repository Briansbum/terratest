package aws

import (
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
)

// GetIamCurrentUserName gets the username for the current IAM user.
func GetIamCurrentUserName(t *testing.T, sessExists ...*session.Session) string {
	out, err := GetIamCurrentUserNameE(t, sessExists[0])
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// GetIamCurrentUserNameE gets the username for the current IAM user.
func GetIamCurrentUserNameE(t *testing.T, sessExists ...*session.Session) (string, error) {
	iamClient, err := NewIamClientE(t, defaultRegion, sessExists[0])
	if err != nil {
		return "", err
	}

	resp, err := iamClient.GetUser(&iam.GetUserInput{})
	if err != nil {
		return "", err
	}

	return *resp.User.UserName, nil
}

// GetIamCurrentUserArn gets the ARN for the current IAM user.
func GetIamCurrentUserArn(t *testing.T, sessExists ...*session.Session) string {
	out, err := GetIamCurrentUserArnE(t, sessExists[0])
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// GetIamCurrentUserArnE gets the ARN for the current IAM user.
func GetIamCurrentUserArnE(t *testing.T, sessExists ...*session.Session) (string, error) {
	iamClient, err := NewIamClientE(t, defaultRegion, sessExists[0])
	if err != nil {
		return "", err
	}

	resp, err := iamClient.GetUser(&iam.GetUserInput{})
	if err != nil {
		return "", err
	}

	return *resp.User.Arn, nil
}

// CreateMfaDevice creates an MFA device using the given IAM client.
func CreateMfaDevice(t *testing.T, iamClient *iam.IAM, deviceName string, sessExists ...*session.Session) *iam.VirtualMFADevice {
	mfaDevice, err := CreateMfaDeviceE(t, iamClient, deviceName, sessExists[0])
	if err != nil {
		t.Fatal(err)
	}
	return mfaDevice
}

// CreateMfaDeviceE creates an MFA device using the given IAM client.
func CreateMfaDeviceE(t *testing.T, iamClient *iam.IAM, deviceName string, sessExists ...*session.Session) (*iam.VirtualMFADevice, error) {
	logger.Logf(t, "Creating an MFA device called %s", deviceName)

	output, err := iamClient.CreateVirtualMFADevice(&iam.CreateVirtualMFADeviceInput{
		VirtualMFADeviceName: aws.String(deviceName),
	})
	if err != nil {
		return nil, err
	}

	if err := EnableMfaDeviceE(t, iamClient, output.VirtualMFADevice, sessExists[0]); err != nil {
		return nil, err
	}

	return output.VirtualMFADevice, nil
}

// EnableMfaDevice enables a newly created MFA Device by supplying the first two one-time passwords, so that it can be used for future
// logins by the given IAM User.
func EnableMfaDevice(t *testing.T, iamClient *iam.IAM, mfaDevice *iam.VirtualMFADevice, sessExists ...*session.Session) {
	err := EnableMfaDeviceE(t, iamClient, mfaDevice, sessExists[0])
	if err != nil {
		t.Fatal(err)
	}
}

// EnableMfaDeviceE enables a newly created MFA Device by supplying the first two one-time passwords, so that it can be used for future
// logins by the given IAM User.
func EnableMfaDeviceE(t *testing.T, iamClient *iam.IAM, mfaDevice *iam.VirtualMFADevice, sessExists ...*session.Session) error {
	logger.Logf(t, "Enabling MFA device %s", aws.StringValue(mfaDevice.SerialNumber))

	iamUserName, err := GetIamCurrentUserArnE(t, sessExists[0])
	if err != nil {
		return err
	}

	authCode1, err := GetTimeBasedOneTimePassword(mfaDevice)
	if err != nil {
		return err
	}

	logger.Logf(t, "Waiting 30 seconds for a new MFA Token to be generated...")
	time.Sleep(30 * time.Second)

	authCode2, err := GetTimeBasedOneTimePassword(mfaDevice)
	if err != nil {
		return err
	}

	_, err = iamClient.EnableMFADevice(&iam.EnableMFADeviceInput{
		AuthenticationCode1: aws.String(authCode1),
		AuthenticationCode2: aws.String(authCode2),
		SerialNumber:        mfaDevice.SerialNumber,
		UserName:            aws.String(iamUserName),
	})

	if err != nil {
		return err
	}

	logger.Log(t, "Waiting for MFA Device enablement to propagate.")
	time.Sleep(10 * time.Second)

	return nil
}

// NewIamClient creates a new IAM client.
func NewIamClient(t *testing.T, region string, sessExists ...*session.Session) *iam.IAM {
	client, err := NewIamClientE(t, region, sessExists[0])
	if err != nil {
		t.Fatal(err)
	}
	return client
}

// NewIamClientE creates a new IAM client.
func NewIamClientE(t *testing.T, region string, sessExists ...*session.Session) (*iam.IAM, error) {
	sess, err := NewAuthenticatedSession(region, sessExists[0])
	if err != nil {
		return nil, err
	}
	return iam.New(sess), nil
}
