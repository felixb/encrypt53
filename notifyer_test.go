package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/felixb/encrypt53/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	notifyer_test_subject  = "some-subject"
	notifyer_test_topicArn = "some-topic-arn"
)

var (
	notifyer_test_msg = map[string]string{"some-key": "some-value"}
)

func TestNotifyer_Notify_emptyTopic(t *testing.T) {
	client, n := givenNotifyer("")

	assert.NoError(t, n.Notify(notifyer_test_subject, notifyer_test_msg))
	client.AssertExpectations(t)
}

func TestNotifyer_Notify_withTopic(t *testing.T) {
	client, n := givenNotifyer(notifyer_test_topicArn)

	client.On("Publish", mock.MatchedBy(func(input *sns.PublishInput) bool {
		return *input.Subject == notifyer_test_subject &&
			*input.TopicArn == notifyer_test_topicArn &&
			strings.Contains(*input.Message, `"some-key":`)
	})).Return(nil, nil)

	assert.NoError(t, n.Notify(notifyer_test_subject, notifyer_test_msg))
	client.AssertExpectations(t)
}

func TestNotifyer_Error(t *testing.T) {
	client, n := givenNotifyer(notifyer_test_topicArn)
	err := errors.New("some error")

	client.On("Publish", mock.MatchedBy(func(input *sns.PublishInput) bool {
		return *input.Subject == notifyer_test_subject &&
			*input.TopicArn == notifyer_test_topicArn &&
			strings.Contains(*input.Message, `"error-message": "some error"`)
	})).Return(nil, nil)

	assert.Equal(t, err, n.Error(notifyer_test_subject, err))
	client.AssertExpectations(t)
}

func TestNotifyer_Error_nil(t *testing.T) {
	client, n := givenNotifyer(notifyer_test_topicArn)

	assert.NoError(t, n.Error(notifyer_test_subject, nil))
	client.AssertExpectations(t)
}

func givenNotifyer(topicArn string) (*mocks.SNSAPI, *Notifyer) {
	client := new(mocks.SNSAPI)
	n := &Notifyer{
		client:   client,
		topicArn: topicArn,
	}
	return client, n
}
