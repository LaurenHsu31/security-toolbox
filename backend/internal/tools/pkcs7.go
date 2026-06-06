package tools

import (
	"encoding/asn1"
	"encoding/json"
)

type cmsContentInfo struct {
	ContentType asn1.ObjectIdentifier
	Content     asn1.RawValue `asn1:"explicit,optional,tag:0"`
}

func handlePKCS7(raw json.RawMessage) (any, error) {
	var in pemInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	der, format, err := DecodeToDER(in.Input, "PKCS7")
	if err != nil {
		return nil, err
	}

	out := map[string]any{"detectedFormat": format}

	var ci cmsContentInfo
	if _, err := asn1.Unmarshal(der, &ci); err == nil {
		ct := ci.ContentType.String()
		entry := map[string]any{"oid": ct}
		if name := oidLookup(ct); name != "" {
			entry["name"] = name
		}
		out["contentType"] = entry
	}

	tree, err := walkASN1(der, 0)
	if err != nil {
		return nil, err
	}
	out["asn1"] = tree
	return out, nil
}
