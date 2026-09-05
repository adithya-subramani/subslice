package transform

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"subslice/pkg/config"
	"subslice/pkg/model"
)

type Transformer struct {
	rules map[string][]config.Rule
}

func NewTransformer(rules map[string][]config.Rule) *Transformer {
	return &Transformer{rules: rules}
}

// TransformRecord modifies sensitivity fields on generic Record buffers in-place
func (t *Transformer) TransformRecord(record *model.Record) {
	entityRules, exists := t.rules[record.EntityName]
	if !exists {
		return
	}

	for _, rule := range entityRules {
		val, found := record.Data[rule.Column]
		if !found || val == nil {
			continue
		}

		strVal := fmt.Sprintf("%v", val)

		switch rule.Rule {
		case "hash":
			record.Data[rule.Column] = hashHMAC(strVal, rule.Salt)
		case "nullify":
			record.Data[rule.Column] = nil
		case "fake_name":
			record.Data[rule.Column] = "Anonymized User"
		case "redact_pan":
			if len(strVal) >= 4 {
				record.Data[rule.Column] = "**** **** **** " + strVal[len(strVal)-4:]
			} else {
				record.Data[rule.Column] = "**** **** **** ****"
			}
		}
	}
}

func hashHMAC(input, salt string) string {
	h := hmac.New(sha256.New, []byte(salt))
	h.Write([]byte(input))
	return hex.EncodeToString(h.Sum(nil))[:16]
}
