package utils

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// SetBalancesFor is the argument storage object for the SetBalancesFor command
type SetBalancesFor struct {
	Who            common.Address `json:"who"`
	LockedStake    *big.Int       `json:"lockedStake"`
	UnlockedStake  *big.Int       `json:"unlockedStake"`
	UnlockedReward *big.Int       `json:"unlockedReward"`
}

// AddValidatorImmediate is the argument storage object for the AddValidatorImmediate command
type AddValidatorImmediate struct {
	Validator common.Address `json:"validator"`
	MadID     [2]*big.Int    `json:"madID"`
}

// DirectDeposit is the argument storage object for the DirectDeposit command
type DirectDeposit struct {
	ID     *big.Int       `json:"id"`
	Who    common.Address `json:"who"`
	Amount *big.Int       `json:"amount"`
}

// Migrate is the argument storage object for the Migrate command
type Migrate struct {
	Epoch     *big.Int         `json:"epoch"`
	EthHeight uint32           `json:"eth_height"`
	MadHeight uint32           `json:"mad_height"`
	MPK       [4]*big.Int      `json:"mpk"`
	Addresses []common.Address `json:"addresses"`
	Gpkj      [][4]*big.Int    `json:"gpkj"`
}

// Snapshot is the argument storage object for the Snapshot command
type Snapshot struct {
	SnapshotID     *big.Int `json:"snapshotID"`
	SignatureGroup []byte   `json:"signatureGroup"`
	BClaims        []byte   `json:"bClaims"`
}

// SetEpoch is the argument storage object for the SetEpoch command
type SetEpoch struct {
	Number *big.Int `json:"number"`
}
