package main

import (
	"bytes"
	"encoding/json"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
)

type Notifyer struct {
	client   snsiface.SNSAPI
	topicArn string
}

func NewNotifyer(sess *session.Session, topicArn string) *Notifyer {
	return &Notifyer{
		client:   sns.New(sess),
		topicArn: topicArn,
	}
}

func (n *Notifyer) Notify(subject string, msg interface{}) error {
	if n.topicArn == "" {
		return nil
	}

	if b, err := json.Marshal(msg); err != nil {
		return err
	} else {
		var out bytes.Buffer
		if err := json.Indent(&out, b, "", "  "); err != nil {
			return err
		}

		_, err := n.client.Publish(&sns.PublishInput{
			TopicArn: aws.String(n.topicArn),
			Subject:  aws.String(subject),
			Message:  aws.String(out.String()),
		})
		return err
	}
}

func (n *Notifyer) Error(subject string, err error) error {
	if err == nil {
		return nil
	}

	msg := make(map[string]string)
	msg["error-message"] = err.Error()

	_ = n.Notify(subject, msg)
	return err
}
