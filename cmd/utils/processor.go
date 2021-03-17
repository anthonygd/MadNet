package utils

import (
	"math/big"

	"github.com/MadBase/MadNet/blockchain"
	"github.com/MadBase/MadNet/consensus/db"
	cobjs "github.com/MadBase/MadNet/consensus/objs"
	"github.com/MadBase/MadNet/constants"
	"github.com/MadBase/MadNet/crypto/bn256"
	"github.com/MadBase/MadNet/crypto/bn256/cloudflare"
	"github.com/MadBase/MadNet/logging"
	"github.com/dgraph-io/badger/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/sirupsen/logrus"
)

// NewProcessor returns a configured Processor
func NewProcessor(db *db.Database, c *blockchain.Contracts, outpath string, stakeLim, reward *big.Int) (*Processor, func() error, error) {
	w, cf, err := OpenFileAsWriter(outpath)
	if err != nil {
		return nil, nil, err
	}
	p := &Processor{
		w:               &Writer{Writer: w},
		db:              db,
		c:               c,
		stakeLim:        stakeLim,
		reward:          reward,
		epoch:           1,
		logger:          logging.GetLogger("processor"),
		finalValidators: make(map[common.Address]uint32),
		validatorIDMap:  make(map[common.Address][2]*big.Int),
		validatorCmdMap: make(map[uint32][]*AddValidatorImmediate),
		snapShots:       make(map[uint32]*Snapshot),
		migrations:      make(map[uint32]*Migrate),
		deposits:        make(map[uint32][]*DirectDeposit),
	}
	return p, cf, nil
}

// Processor converts a stream of events into a set of *CommandObj for writing
// to a file
type Processor struct {
	db              *db.Database
	w               *Writer
	c               *blockchain.Contracts
	stakeLim        *big.Int
	reward          *big.Int
	epoch           uint32
	logger          *logrus.Logger
	finalValidators map[common.Address]uint32
	validatorIDMap  map[common.Address][2]*big.Int
	validatorCmdMap map[uint32][]*AddValidatorImmediate
	snapShots       map[uint32]*Snapshot
	migrations      map[uint32]*Migrate
	deposits        map[uint32][]*DirectDeposit
}

// ValidatorJoined handles KeyShareSubmission events
func (p *Processor) ValidatorJoined(ebn uint64, l types.Log) error {
	event, err := p.c.Validators.ParseValidatorJoined(l)
	if err != nil {
		p.logger.Errorf("validator joined: %v", err)
		return err
	}
	p.validatorIDMap[event.Validator] = event.MadID
	return nil
}

// ValidatorSet handles ValidatorSet events
func (p *Processor) ValidatorSet(ebn uint64, l types.Log) error {
	event, err := p.c.Ethdkg.ParseValidatorSet(l)
	if err != nil {
		p.logger.Errorf("validator set: %v", err)
		return err
	}
	epoch := event.Epoch
	height := event.MadHeight
	validatorCount := event.ValidatorCount

	mpk := [4]*big.Int{
		event.GroupKey0,
		event.GroupKey1,
		event.GroupKey2,
		event.GroupKey3,
	}

	addrList := []common.Address{}
	gpkjList := [][4]*big.Int{}
	err = p.db.View(func(txn *badger.Txn) error {
		vs, err := p.db.GetValidatorSet(txn, height)
		if err != nil {
			return err
		}
		for i := 0; i < len(vs.Validators); i++ {
			pubkpoint := new(cloudflare.G2)
			addr := [20]byte{}
			_, err := pubkpoint.Unmarshal(vs.Validators[i].GroupShare)
			if err != nil {
				return err
			}
			gpkjList = append(gpkjList, bn256.G2ToBigIntArray(pubkpoint))
			copy(addr[:], vs.Validators[i].VAddr)
			addrList = append(addrList, common.Address(addr))
		}
		return nil
	})
	if err != nil {
		return err
	}

	epoch32 := uint32(epoch.Uint64())
	for i := 0; i < len(addrList); i++ {
		if p.validatorCmdMap[epoch32] == nil {
			p.validatorCmdMap[epoch32] = make([]*AddValidatorImmediate, validatorCount)
		}
		p.validatorCmdMap[epoch32][i] = &AddValidatorImmediate{
			Validator: addrList[i],
		}
	}

	m := &Migrate{
		Epoch:     epoch,
		EthHeight: uint32(ebn),
		MadHeight: height,
		MPK:       mpk,
		Addresses: addrList,
		Gpkj:      gpkjList,
	}
	p.migrations[epoch32] = m
	return nil
}

// Deposit handles Deposit events
func (p *Processor) Deposit(ebn uint64, l types.Log) error {
	if p.deposits[p.epoch] == nil {
		p.deposits[p.epoch] = []*DirectDeposit{}
	}
	event, err := p.c.Deposit.ParseDepositReceived(l)
	if err != nil {
		p.logger.Errorf("deposit: %v", err)
		return err
	}
	d := &DirectDeposit{
		Who:    event.Depositor,
		Amount: event.Amount,
		ID:     event.DepositID,
	}
	p.deposits[p.epoch] = append(p.deposits[p.epoch], d)
	return nil
}

// Snapshot handles Snapshot events
func (p *Processor) Snapshot(ebn uint64, l types.Log) error {
	event, err := p.c.Validators.ParseSnapshotTaken(l)
	if err != nil {
		p.logger.Errorf("snapshot: %v", err)
		return err
	}
	epoch := uint32(event.Epoch.Uint64())
	height := constants.EpochLength * epoch
	var bh *cobjs.BlockHeader
	err = p.db.View(func(txn *badger.Txn) error {
		tmp, err := p.db.GetCommittedBlockHeader(txn, height)
		if err != nil {
			return err
		}
		bh = tmp
		return nil
	})
	if err != nil {
		return err
	}
	rawBclaims, err := bh.BClaims.MarshalBinary()
	if err != nil {
		return err
	}
	s := &Snapshot{
		SnapshotID:     event.Epoch,
		SignatureGroup: bh.SigGroup,
		BClaims:        rawBclaims,
	}
	p.snapShots[epoch] = s
	return p.NextEpoch(epoch)
}

// NextEpoch advances the state by one epoch
func (p *Processor) NextEpoch(epoch uint32) error {
	if p.migrations[epoch] != nil {
		avl := p.validatorCmdMap[epoch]
		for i := 0; i < len(avl); i++ {
			avl[i].MadID = p.validatorIDMap[avl[i].Validator]
			cmd := &CommandObj{}
			if err := p.w.WriteLine(cmd.WithAddValidatorImmediate(avl[i])); err != nil {
				return err
			}
			p.finalValidators[avl[i].Validator] = epoch
		}
		cmd := &CommandObj{}
		if err := p.w.WriteLine(cmd.WithMigrate(p.migrations[epoch])); err != nil {
			return err
		}
		delete(p.validatorCmdMap, epoch)
		delete(p.migrations, epoch)
	}
	if p.deposits[epoch] != nil {
		for i := 0; i < len(p.deposits); i++ {
			cmd := &CommandObj{}
			if err := p.w.WriteLine(cmd.WithDirectDeposit(p.deposits[epoch][i])); err != nil {
				return err
			}
		}
		delete(p.deposits, epoch)
	}
	p.epoch = epoch + 1
	for se := range p.snapShots {
		if se+10 < p.epoch {
			delete(p.snapShots, se)
		}
	}
	return nil
}

// Finalize sets final state once all blocks have been processed
func (p *Processor) Finalize() error {
	for addr, startEpoch := range p.finalValidators {
		ne := big.NewInt(int64(p.epoch - startEpoch))
		unlocked := ne.Mul(ne, p.reward)
		s := &SetBalancesFor{
			Who:            addr,
			LockedStake:    p.stakeLim,
			UnlockedStake:  big.NewInt(0),
			UnlockedReward: unlocked,
		}
		cmd := &CommandObj{}
		if err := p.w.WriteLine(cmd.WithSetBalancesFor(s)); err != nil {
			return err
		}
	}
	for i := uint32(5); i > 0; i-- {
		cmd := &CommandObj{}
		if err := p.w.WriteLine(cmd.WithSnapshot(p.snapShots[p.epoch-i])); err != nil {
			return err
		}
	}
	e := &SetEpoch{
		Number: big.NewInt(int64(p.epoch)),
	}
	cmd := &CommandObj{}
	if err := p.w.WriteLine(cmd.WithSetEpoch(e)); err != nil {
		return err
	}
	return nil
}
