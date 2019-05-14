package main

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/felixb/encrypt53/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDnsClient_DeleteChallengeRecord(t *testing.T) {
	client, dc := givenDnsClient()
	dc.zones["some.domain"] = "some-id"

	client.On("ChangeResourceRecordSets", mock.MatchedBy(func(input *route53.ChangeResourceRecordSetsInput) bool {
		return *input.HostedZoneId == "some-id" &&
			*input.ChangeBatch.Changes[0].Action == "DELETE" &&
			*input.ChangeBatch.Changes[0].ResourceRecordSet.Name == "_acme-challenge.server.some.domain." &&
			*input.ChangeBatch.Changes[0].ResourceRecordSet.Type == "TXT" &&
			*input.ChangeBatch.Changes[0].ResourceRecordSet.ResourceRecords[0].Value == `"some-token-value"`
	})).Return(nil, nil)

	assert.NoError(t, dc.DeleteChallengeRecord("server.some.domain", "some-token-value"))
	client.AssertExpectations(t)
}

func TestDnsClient_getHostedZoneId(t *testing.T) {
	client, dc := givenDnsClient()

	client.On("ListHostedZonesByName", mock.MatchedBy(func(input *route53.ListHostedZonesByNameInput) bool {
		return input.HostedZoneId == nil && *input.DNSName == "some.domain"
	})).Return(&route53.ListHostedZonesByNameOutput{
		HostedZones: []*route53.HostedZone{
			{Id: aws.String("some-id"), Name: aws.String("some.domain")}}}, nil)

	id, err := dc.getHostedZoneId("some.domain")

	assert.NoError(t, err)
	assert.Equal(t, "some-id", id)
	assert.Equal(t, "some-id", dc.zones["some.domain"])
	client.AssertExpectations(t)
}

func TestDnsClient_getHostedZoneId_from_cache(t *testing.T) {
	client, dc := givenDnsClient()
	dc.zones["some.domain"] = "some-id"

	id, err := dc.getHostedZoneId("some.domain")

	assert.NoError(t, err)
	assert.Equal(t, "some-id", id)
	client.AssertExpectations(t)
}

func givenDnsClient() (*mocks.Route53API, *DnsClient) {
	client := new(mocks.Route53API)
	dc := &DnsClient{
		client: client,
		zones:  make(map[string]string),
	}

	return client, dc
}
