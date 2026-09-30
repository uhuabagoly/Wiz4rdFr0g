package releaseproof

import "regexp"

type SeafileRetentionProof struct {
	Key       string `json:"key"`
	View      string `json:"view"`
	Name      string `json:"name"`
	Value     string `json:"value"`
	Preserved bool   `json:"preserved"`
}

func (p *SeafileRetentionProof) valid() bool {
	return p != nil && p.Key == `HKEY_CURRENT_USER\SOFTWARE\Seafile` && p.View == "Registry64" && p.Preserved && regexp.MustCompile(`^Wiz4rdFr0g-retention-[a-f0-9]{32}$`).MatchString(p.Name) && regexp.MustCompile(`^[a-f0-9]{64}$`).MatchString(p.Value)
}
