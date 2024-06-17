package huaweicloud

import (
	"fmt"
	"slices"

	"github.com/StackExchange/dnscontrol/v4/models"
	"github.com/StackExchange/dnscontrol/v4/pkg/printer"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dns/v2/model"
)

func getRRSetIDFromRecords(rcs models.Records) []string {
	ids := []string{}
	for _, r := range rcs {
		if r.Original == nil {
			continue
		}
		if r.Original.(*model.ListRecordSets).Id == nil {
			printer.Warnf("RecordSet ID is nil for record %+v\n", r)
			continue
		}
		ids = append(ids, *r.Original.(*model.ListRecordSets).Id)
	}
	return slices.Compact(ids)
}

func nativeToRecords(n *model.ListRecordSets, zoneName string) (models.Records, error) {
	if n.Name == nil || n.Type == nil || n.Records == nil || n.Ttl == nil {
		return nil, fmt.Errorf("missing required fields in Huaweicloud's RRset: %+v", n)
	}
	var rcs models.Records
	recName := *n.Name
	recType := *n.Type

	// Split into records
	for _, value := range *n.Records {
		rc := &models.RecordConfig{
			TTL:      uint32(*n.Ttl),
			Original: n,
		}
		rc.SetLabelFromFQDN(recName, zoneName)
		if err := rc.PopulateFromString(recType, value, zoneName); err != nil {
			return nil, fmt.Errorf("unparsable record received from Huaweicloud: %w", err)
		}
		rcs = append(rcs, rc)
	}

	return rcs, nil
}

func recordsToNative(rcs models.Records, expectedKey models.RecordKey) *model.ListRecordSets {
	resultTTL := int32(0)
	resultVal := []string{}
	name := expectedKey.NameFQDN + "."
	result := &model.ListRecordSets{
		Name:    &name,
		Type:    &expectedKey.Type,
		Ttl:     &resultTTL,
		Records: &resultVal,
	}

	for _, r := range rcs {
		key := r.Key()
		if key != expectedKey {
			continue
		}
		val := r.GetTargetCombined()
		// special case for empty TXT records
		if key.Type == "TXT" && len(val) == 0 {
			val = "\"\""
		}

		resultVal = append(resultVal, val)
		if resultTTL == 0 {
			resultTTL = int32(r.TTL)
		}

		// Check if all TTLs are the same
		if int32(r.TTL) != resultTTL {
			printer.Warnf("All TTLs for a rrset (%v) must be the same. Using smaller of %v and %v.\n", key, r.TTL, resultTTL)
			if int32(r.TTL) < resultTTL {
				resultTTL = int32(r.TTL)
			}
		}
	}

	return result
}
