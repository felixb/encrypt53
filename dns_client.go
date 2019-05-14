package main

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/route53/route53iface"
)

type DnsClient struct {
	client route53iface.Route53API
	zones  map[string]string
}

func NewDnsClient(sess *session.Session) *DnsClient {
	return &DnsClient{
		client: route53.New(sess),
		zones:  make(map[string]string),
	}
}

func (dc *DnsClient) AddChallengeRecord(fqdn, value string) error {
	challengeFqdn := challengeFqdn(fqdn)
	domain := fqdn2domain(fqdn)

	log.Infof("adding TXT record %s", challengeFqdn)
	log.Debugf("adding TXT record %s with value %s", challengeFqdn, value)
	if err := dc.changeRecord(domain, &route53.Change{
		Action:            aws.String("UPSERT"),
		ResourceRecordSet: resourceRecordSet(challengeFqdn, value),
	}); err != nil {
		return err
	} else {
		return dc.waitForDns(challengeFqdn, value)
	}
}

func (dc *DnsClient) DeleteChallengeRecord(fqdn, value string) error {
	challengeFqdn := challengeFqdn(fqdn)
	domain := fqdn2domain(fqdn)

	log.Infof("deleting TXT record %s", challengeFqdn)
	return dc.changeRecord(domain, &route53.Change{
		Action:            aws.String("DELETE"),
		ResourceRecordSet: resourceRecordSet(challengeFqdn, value),
	})
}

func (dc *DnsClient) changeRecord(domain string, change *route53.Change) error {
	if hostedZoneId, err := dc.getHostedZoneId(domain); err != nil {
		return err
	} else {
		_, err := dc.client.ChangeResourceRecordSets(&route53.ChangeResourceRecordSetsInput{
			ChangeBatch:  &route53.ChangeBatch{Changes: []*route53.Change{change}},
			HostedZoneId: aws.String(hostedZoneId),
		})
		return err
	}
}

func (dc *DnsClient) getHostedZoneId(domain string) (string, error) {
	if id, ok := dc.zones[domain]; ok {
		return id, nil
	}

	input := &route53.ListHostedZonesByNameInput{DNSName: aws.String(domain)}
	if result, err := dc.client.ListHostedZonesByName(input); err != nil {
		return "", err
	} else {
		for _, zone := range result.HostedZones {
			id := *zone.Id
			dc.zones[domain] = id
			return id, nil
		}
	}

	return "", fmt.Errorf("there is no hosted zone for domain %s", domain)
}

func (dc *DnsClient) waitForDns(fqdn, value string) error {
	for {
		log.Debugf("waiting for dns to settle for %s", fqdn)
		time.Sleep(5 * time.Second)

		if result, err := net.LookupTXT(fqdn); err != nil {
			log.Debugf("got error while waiting for dns for %s: %s", fqdn, err.Error())
			continue
		} else if len(result) == 1 && result[0] == value {
			break
		}
	}

	log.Infof("waiting for dns to settle even more for %s", fqdn)
	time.Sleep(10 * time.Second)
	return nil
}

func resourceRecordSet(fqdn, value string) *route53.ResourceRecordSet {
	return &route53.ResourceRecordSet{
		Name:            aws.String(fqdn),
		ResourceRecords: []*route53.ResourceRecord{{Value: aws.String(fmt.Sprintf(`"%s"`, value))}},
		TTL:             aws.Int64(1),
		Type:            aws.String("TXT"),
	}
}

func challengeFqdn(domain string) string {
	return fmt.Sprintf("_acme-challenge.%s.", domain)
}

func fqdn2domain(fqdn string) string {
	return strings.SplitN(fqdn, ".", 2)[1]
}
