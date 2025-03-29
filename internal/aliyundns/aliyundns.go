package aliyundns

import (
	"fmt"
	"strings"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/alidns"
	"github.com/go-acme/lego/v4/challenge/dns01"
	"github.com/go-acme/lego/v4/platform/config/env"
)

// DNSProvider implements the challenge.Provider interface for Aliyun DNS
type DNSProvider struct {
	client   *alidns.Client
	waitTime time.Duration
	zoneName string
}

// NewDNSProvider returns a new Aliyun DNS provider
func NewDNSProvider(accessKey, secretKey, regionID string) (*DNSProvider, error) {
	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("Aliyun DNS: access key and secret key are required")
	}

	if regionID == "" {
		regionID = "cn-hangzhou" // Default region
	}

	// Create a new Aliyun DNS client
	client, err := alidns.NewClientWithAccessKey(regionID, accessKey, secretKey)
	if err != nil {
		return nil, fmt.Errorf("Aliyun DNS: %v", err)
	}

	waitTime := env.GetOrDefaultSecond("ALIYUN_POLLING_INTERVAL", 60)

	return &DNSProvider{
		client:   client,
		waitTime: time.Duration(waitTime) * time.Second,
	}, nil
}

// Present creates a TXT record to fulfill the DNS-01 challenge
func (d *DNSProvider) Present(domain, token, keyAuth string) error {
	fqdn, value := dns01.GetRecord(domain, keyAuth)

	// Get the domain zone name
	zoneName, err := d.getHostedZone(domain)
	if err != nil {
		return fmt.Errorf("Aliyun DNS: %v", err)
	}
	d.zoneName = zoneName

	// Create a new DNS record
	recordName := d.getRecordName(fqdn, zoneName)
	request := alidns.CreateAddDomainRecordRequest()
	request.DomainName = zoneName
	request.Type = "TXT"
	request.RR = recordName
	request.Value = value
	request.TTL = "600"

	_, err = d.client.AddDomainRecord(request)
	if err != nil {
		return fmt.Errorf("Aliyun DNS: %v", err)
	}

	return nil
}

// CleanUp removes the TXT record matching the specified parameters
func (d *DNSProvider) CleanUp(domain, token, keyAuth string) error {
	fqdn, _ := dns01.GetRecord(domain, keyAuth)

	// Get the record IDs
	recordName := d.getRecordName(fqdn, d.zoneName)
	request := alidns.CreateDescribeDomainRecordsRequest()
	request.DomainName = d.zoneName
	request.RRKeyWord = recordName
	request.Type = "TXT"

	response, err := d.client.DescribeDomainRecords(request)
	if err != nil {
		return fmt.Errorf("Aliyun DNS: %v", err)
	}

	// Delete the record(s)
	for _, record := range response.DomainRecords.Record {
		request := alidns.CreateDeleteDomainRecordRequest()
		request.RecordId = record.RecordId
		_, err = d.client.DeleteDomainRecord(request)
		if err != nil {
			return fmt.Errorf("Aliyun DNS: %v", err)
		}
	}

	return nil
}

// Timeout returns the timeout for the DNS provider
func (d *DNSProvider) Timeout() time.Duration {
	return d.waitTime
}

// getHostedZone returns the hosted zone name for a domain
func (d *DNSProvider) getHostedZone(domain string) (string, error) {
	request := alidns.CreateDescribeDomainsRequest()
	response, err := d.client.DescribeDomains(request)
	if err != nil {
		return "", err
	}

	var hostedZone string
	for _, zone := range response.Domains.Domain {
		if isZoneMatch(zone.DomainName, domain) {
			if len(zone.DomainName) > len(hostedZone) {
				hostedZone = zone.DomainName
			}
		}
	}

	if hostedZone == "" {
		return "", fmt.Errorf("zone not found for domain %s", domain)
	}

	return hostedZone, nil
}

// isZoneMatch checks if a domain is a subdomain of a zone
func isZoneMatch(zone, domain string) bool {
	return strings.HasSuffix(domain, zone) || domain == zone
}

// getRecordName returns the record name for a domain
func (d *DNSProvider) getRecordName(fqdn, domain string) string {
	name := dns01.UnFqdn(fqdn)
	if domain != name && name != "" {
		name = name[:len(name)-len(domain)-1]
	} else {
		name = "@"
	}
	return name
}
