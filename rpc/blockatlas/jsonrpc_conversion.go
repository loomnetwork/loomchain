package blockatlas

type JsonGetValidators struct {
	Validators []Validator `json:"validators,omitempty"`
}

type Validator struct {
	Address         string `json:"address,omitempty"`
	Jailed          bool   `json:"jailed,omitempty"`
	Name            string `json:"name,omitempty"`
	Description     string `json:"description,omitempty"`
	Image           string `json:"image,omitempty"`
	Website         string `json:"website,omitempty"`
	DelegationTotal string `json:"delegationTotal,omitempty"`
}
