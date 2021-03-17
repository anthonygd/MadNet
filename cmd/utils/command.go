package utils

import "encoding/json"

// CommandObj is the generic argument storage object for function invocation
// this object is a mutually exclusive object that will delete any other
// data if more than one field is set.
type CommandObj struct {
	SetBalancesFor        *SetBalancesFor        `json:"setBalancesFor,omitempty"`
	AddValidatorImmediate *AddValidatorImmediate `json:"addValidatorImmediate,omitempty"`
	DirectDeposit         *DirectDeposit         `json:"directDeposit,omitempty"`
	Migrate               *Migrate               `json:"migrate,omitempty"`
	Snapshot              *Snapshot              `json:"snapshot,omitempty"`
	SetEpoch              *SetEpoch              `json:"setEpoch,omitempty"`
}

// Marshal outputs the object as a string
func (c *CommandObj) Marshal() (string, error) {
	b, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return string(b), err
}

// Unmarshal populates object from a string
func (c *CommandObj) Unmarshal(s string) error {
	return json.Unmarshal([]byte(s), c)
}

// WithSetBalancesFor sets the SetBalancesFor of the object
func (c *CommandObj) WithSetBalancesFor(obj *SetBalancesFor) *CommandObj {
	c.SetBalancesFor = nil
	c.AddValidatorImmediate = nil
	c.DirectDeposit = nil
	c.Migrate = nil
	c.Snapshot = nil
	c.SetEpoch = nil
	c.SetBalancesFor = obj
	return c
}

// WithAddValidatorImmediate sets the AddValidatorImmediate of the object
func (c *CommandObj) WithAddValidatorImmediate(obj *AddValidatorImmediate) *CommandObj {
	c.SetBalancesFor = nil
	c.AddValidatorImmediate = nil
	c.DirectDeposit = nil
	c.Migrate = nil
	c.Snapshot = nil
	c.SetEpoch = nil
	c.AddValidatorImmediate = obj
	return c
}

// WithDirectDeposit sets the DirectDeposit of the object
func (c *CommandObj) WithDirectDeposit(obj *DirectDeposit) *CommandObj {
	c.SetBalancesFor = nil
	c.AddValidatorImmediate = nil
	c.DirectDeposit = nil
	c.Migrate = nil
	c.Snapshot = nil
	c.SetEpoch = nil
	c.DirectDeposit = obj
	return c
}

// WithMigrate sets the Migrate of the object
func (c *CommandObj) WithMigrate(obj *Migrate) *CommandObj {
	c.SetBalancesFor = nil
	c.AddValidatorImmediate = nil
	c.DirectDeposit = nil
	c.Migrate = nil
	c.Snapshot = nil
	c.SetEpoch = nil
	c.Migrate = obj
	return c
}

// WithSnapshot sets the Snapshot of the object
func (c *CommandObj) WithSnapshot(obj *Snapshot) *CommandObj {
	c.SetBalancesFor = nil
	c.AddValidatorImmediate = nil
	c.DirectDeposit = nil
	c.Migrate = nil
	c.Snapshot = nil
	c.SetEpoch = nil
	c.Snapshot = obj
	return c
}

// WithSetEpoch sets the SetEpoch of the object
func (c *CommandObj) WithSetEpoch(obj *SetEpoch) *CommandObj {
	c.SetBalancesFor = nil
	c.AddValidatorImmediate = nil
	c.DirectDeposit = nil
	c.Migrate = nil
	c.Snapshot = nil
	c.SetEpoch = nil
	c.SetEpoch = obj
	return c
}

// HasSetBalancesFor returns true if the object contains a SetBalancesFor
func (c *CommandObj) HasSetBalancesFor() bool {
	if c.SetBalancesFor != nil {
		return true
	}
	return false
}

// HasAddValidatorImmediate returns true if the object contains a AddValidatorImmediate
func (c *CommandObj) HasAddValidatorImmediate() bool {
	if c.AddValidatorImmediate != nil {
		return true
	}
	return false
}

// HasDirectDeposit returns true if the object contains a DirectDeposit
func (c *CommandObj) HasDirectDeposit() bool {
	if c.DirectDeposit != nil {
		return true
	}
	return false
}

// HasMigrate returns true if the object contains a Migrate
func (c *CommandObj) HasMigrate() bool {
	if c.Migrate != nil {
		return true
	}
	return false
}

// HasSnapshot returns true if the object contains a Snapshot
func (c *CommandObj) HasSnapshot() bool {
	if c.Snapshot != nil {
		return true
	}
	return false
}

// HasSetEpoch returns true if the object contains a SetEpoch
func (c *CommandObj) HasSetEpoch() bool {
	if c.SetEpoch != nil {
		return true
	}
	return false
}
