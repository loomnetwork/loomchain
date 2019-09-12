package trustwallet

type JsonGetValidators struct {
	Validators []Validator `json:"validators,omitempty"`
}

type Validator struct {
	Address         string `json:"address,omitempty"`
	Jailed          bool   `json:"jailed"`
	Name            string `json:"name,omitempty"`
	Description     string `json:"description,omitempty"`
	Image           string `json:"image,omitempty"`
	Website         string `json:"website,omitempty"`
	DelegationTotal string `json:"delegationTotal,omitempty"`
	Fee             string `json:"fee,omitempty"`
}

type JsonListDelegation struct {
	Delegations     []Delegation `json:"delegations,omitempty"`
	DelegationTotal string       `json:"delegation_total,omitempty"`
}

type Delegation struct {
	ValidatorAddress   string `json:"validator,omitempty"`
	DelegatorAddress   string `json:"delegator,omitempty"`
	Index              string `json:"index,omitempty"`
	Amount             string `json:"amount,omitempty"`
	UpdatedValidator   string `json:"updated_validator,omitempty"`
	UpdatedAmount      string `json:"updeted_amount,omitempty"`
	LockTimeTier       string `json:"lock_time_tier,omitempty"`
	UpdateLockTimeTier string `json:"updeted_lock_time_tier,omitempty"`
	LockTime           string `json:"lock_time,omitempty"`
	State              string `json:"state,omitempty"`
	Referrer           string `json:"referrer,omitempty"`
}
