package aws

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/google/uuid"
)

// CreateRandomQueue creates a new SQS queue with a random name that starts with the given prefix and return the queue URL.
func CreateRandomQueue(t *testing.T, awsRegion string, prefix string, sessExists ...*session.Session) string {
	url, err := CreateRandomQueueE(t, awsRegion, prefix, sessExists[0])
	if err != nil {
		t.Fatal(err)
	}
	return url
}

// CreateRandomQueueE creates a new SQS queue with a random name that starts with the given prefix and return the queue URL.
func CreateRandomQueueE(t *testing.T, awsRegion string, prefix string, sessExists ...*session.Session) (string, error) {
	logger.Logf(t, "Creating randomly named SQS queue with prefix %s", prefix)

	sqsClient, err := NewSqsClientE(t, awsRegion, sessExists[0])
	if err != nil {
		return "", err
	}

	channel, err := uuid.NewUUID()
	if err != nil {
		return "", err
	}

	channelName := fmt.Sprintf("%s-%s", prefix, channel.String())

	queue, err := sqsClient.CreateQueue(&sqs.CreateQueueInput{
		QueueName: aws.String(channelName),
	})

	if err != nil {
		return "", err
	}

	return aws.StringValue(queue.QueueUrl), nil
}

// DeleteQueue deletes the SQS queue with the given URL.
func DeleteQueue(t *testing.T, awsRegion string, queueURL string, sessExists ...*session.Session) {
	err := DeleteQueueE(t, awsRegion, queueURL, sessExists[0])
	if err != nil {
		t.Fatal(err)
	}
}

// DeleteQueueE deletes the SQS queue with the given URL.
func DeleteQueueE(t *testing.T, awsRegion string, queueURL string, sessExists ...*session.Session) error {
	logger.Logf(t, "Deleting SQS Queue %s", queueURL)

	sqsClient, err := NewSqsClientE(t, awsRegion, sessExists[0])
	if err != nil {
		return err
	}

	_, err = sqsClient.DeleteQueue(&sqs.DeleteQueueInput{
		QueueUrl: aws.String(queueURL),
	})

	return err
}

// DeleteMessageFromQueue deletes the message with the given receipt from the SQS queue with the given URL.
func DeleteMessageFromQueue(t *testing.T, awsRegion string, queueURL string, receipt string, sessExists ...*session.Session) {
	err := DeleteMessageFromQueueE(t, awsRegion, queueURL, receipt, sessExists[0])
	if err != nil {
		t.Fatal(err)
	}
}

// DeleteMessageFromQueueE deletes the message with the given receipt from the SQS queue with the given URL.
func DeleteMessageFromQueueE(t *testing.T, awsRegion string, queueURL string, receipt string, sessExists ...*session.Session) error {
	logger.Logf(t, "Deleting message from queue %s (%s)", queueURL, receipt)

	sqsClient, err := NewSqsClientE(t, awsRegion, sessExists[0])
	if err != nil {
		return err
	}

	_, err = sqsClient.DeleteMessage(&sqs.DeleteMessageInput{
		ReceiptHandle: &receipt,
		QueueUrl:      &queueURL,
	})

	return err
}

// SendMessageToQueue sends the given message to the SQS queue with the given URL.
func SendMessageToQueue(t *testing.T, awsRegion string, queueURL string, message string, sessExists ...*session.Session) {
	err := SendMessageToQueueE(t, awsRegion, queueURL, message, sessExists[0])
	if err != nil {
		t.Fatal(err)
	}
}

// SendMessageToQueueE sends the given message to the SQS queue with the given URL.
func SendMessageToQueueE(t *testing.T, awsRegion string, queueURL string, message string, sessExists ...*session.Session) error {
	logger.Logf(t, "Sending message %s to queue %s", message, queueURL)

	sqsClient, err := NewSqsClientE(t, awsRegion, sessExists[0])
	if err != nil {
		return err
	}

	res, err := sqsClient.SendMessage(&sqs.SendMessageInput{
		MessageBody: &message,
		QueueUrl:    &queueURL,
	})

	if err != nil {
		if strings.Contains(err.Error(), "AWS.SimpleQueueService.NonExistentQueue") {
			logger.Logf(t, fmt.Sprintf("WARN: Client has stopped listening on queue %s", queueURL))
			return nil
		}
		return err
	}

	logger.Logf(t, "Message id %s sent to queue %s", aws.StringValue(res.MessageId), queueURL)

	return nil
}

// QueueMessageResponse contains a queue message.
type QueueMessageResponse struct {
	ReceiptHandle string
	MessageBody   string
	Error         error
}

// WaitForQueueMessage waits to receive a message from on the queueURL. Since the API only allows us to wait a max 20 seconds for a new
// message to arrive, we must loop TIMEOUT/20 number of times to be able to wait for a total of TIMEOUT seconds
func WaitForQueueMessage(t *testing.T, awsRegion string, queueURL string, timeout int, sessExists ...*session.Session) QueueMessageResponse {
	sqsClient, err := NewSqsClientE(t, awsRegion, sessExists[0])
	if err != nil {
		return QueueMessageResponse{Error: err}
	}

	cycles := timeout
	cycleLength := 1
	if timeout >= 20 {
		cycleLength = 20
		cycles = timeout / cycleLength
	}

	for i := 0; i < cycles; i++ {
		logger.Logf(t, "Waiting for message on %s (%ss)", queueURL, strconv.Itoa(i*cycleLength))
		result, err := sqsClient.ReceiveMessage(&sqs.ReceiveMessageInput{
			QueueUrl:              aws.String(queueURL),
			AttributeNames:        aws.StringSlice([]string{"SentTimestamp"}),
			MaxNumberOfMessages:   aws.Int64(1),
			MessageAttributeNames: aws.StringSlice([]string{"All"}),
			WaitTimeSeconds:       aws.Int64(int64(cycleLength)),
		})

		if err != nil {
			return QueueMessageResponse{Error: err}
		}

		if len(result.Messages) > 0 {
			logger.Logf(t, "Message %s received on %s", *result.Messages[0].MessageId, queueURL)
			return QueueMessageResponse{ReceiptHandle: *result.Messages[0].ReceiptHandle, MessageBody: *result.Messages[0].Body}
		}
	}

	return QueueMessageResponse{Error: ReceiveMessageTimeout{QueueUrl: queueURL, TimeoutSec: timeout}}
}

// NewSqsClient creates a new SQS client.
func NewSqsClient(t *testing.T, region string, sessExists ...*session.Session) *sqs.SQS {
	client, err := NewSqsClientE(t, region, sessExists[0])
	if err != nil {
		t.Fatal(err)
	}
	return client
}

// NewSqsClientE creates a new SQS client.
func NewSqsClientE(t *testing.T, region string, sessExists ...*session.Session) (*sqs.SQS, error) {
	sess, err := NewAuthenticatedSession(region, sessExists[0])
	if err != nil {
		return nil, err
	}

	return sqs.New(sess), nil
}

// ReceiveMessageTimeout is an error that occurs if receiving a message times out.
type ReceiveMessageTimeout struct {
	QueueUrl   string
	TimeoutSec int
}

func (err ReceiveMessageTimeout) Error() string {
	return fmt.Sprintf("Failed to receive messages on %s within %s seconds", err.QueueUrl, strconv.Itoa(err.TimeoutSec))
}
