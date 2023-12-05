package app

type Headers struct {
	Global     HeaderKV `json:"global"`
	BaseDomain HeaderKV `json:"baseDomain"`
	NewDomain  HeaderKV `json:"newDomain"`
}

type HeaderKV map[string]string
