package utils

import (
	"context"
	"io"

	"github.com/MadBase/bridge/bindings"

	"github.com/MadBase/MadNet/blockchain"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
)

// NewMigrator returns a configured migrator
func NewMigrator() (*Migrator, error) {
	panic("not implemented")
}

// Migrator performs a migration
type Migrator struct {
	staking    *bindings.MigrateStakingFacet
	validators *bindings.MigrateParticipantsFacet
	deposit    *bindings.Deposit
	snapshots  *bindings.MigrateSnapshotsFacet
	epoch      *bindings.SnapshotsFacet
	dkg        *bindings.MigrateETHDKG
	txOpts     *bind.TransactOpts
	eth        blockchain.Ethereum
	r          *Reader
	w          *Writer // should be used to write cmds as the succeed for recovery
	startLine  int     // should be determined based on written info as above
}

// Migrate will read each line of the file and perform the specified
// action
func (m *Migrator) Migrate() error {
	for i := 0; ; i++ {
		cmd, err := m.r.ReadLine()
		if err != nil {
			if err != io.EOF {
				return err
			}
			break
		}
		if i < m.startLine {
			continue
		}
		tx, err := m.Dispatch(cmd)
		if err != nil {
			return err
		}
		ctx := context.Background()
		_, err = m.eth.WaitForReceipt(ctx, tx)
		if err != nil {
			return err
		}
	}
	return nil
}

// Dispatch exectues the command specified by c
func (m *Migrator) Dispatch(c *CommandObj) (*types.Transaction, error) {
	switch {
	case c.HasSetBalancesFor():
		return m.staking.SetBalancesFor(
			m.txOpts,
			c.SetBalancesFor.Who,
			c.SetBalancesFor.LockedStake,
			c.SetBalancesFor.UnlockedStake,
			c.SetBalancesFor.UnlockedReward,
		)
	case c.HasAddValidatorImmediate():
		return m.validators.AddValidatorImmediate(
			m.txOpts,
			c.AddValidatorImmediate.Validator,
			c.AddValidatorImmediate.MadID,
		)
	case c.HasDirectDeposit():
		return m.deposit.DirectDeposit(
			m.txOpts,
			c.DirectDeposit.ID,
			c.DirectDeposit.Who,
			c.DirectDeposit.Amount,
		)
	case c.HasSnapshot():
		return m.snapshots.Snapshot(
			m.txOpts,
			c.Snapshot.SnapshotID,
			c.Snapshot.SignatureGroup,
			c.Snapshot.BClaims,
		)
	case c.HasMigrate():
		return m.dkg.Migrate(
			m.txOpts,
			c.Migrate.Epoch,
			c.Migrate.EthHeight,
			c.Migrate.MadHeight,
			c.Migrate.MPK,
			c.Migrate.Addresses,
			c.Migrate.Gpkj,
		)
	case c.HasSetEpoch():
		return m.epoch.SetEpoch(
			m.txOpts,
			c.SetEpoch.Number,
		)
	default:
		panic("no type for command")
	}
}
