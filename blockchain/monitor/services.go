package monitor

import (
	"context"
	"time"

	"github.com/MadBase/MadNet/application/deposit"
	"github.com/MadBase/MadNet/blockchain/dkg/dkgevents"
	"github.com/MadBase/MadNet/blockchain/interfaces"
	"github.com/MadBase/MadNet/blockchain/objects"
	"github.com/MadBase/MadNet/blockchain/tasks"
	"github.com/MadBase/MadNet/config"
	"github.com/MadBase/MadNet/consensus/admin"
	"github.com/MadBase/MadNet/consensus/db"
	"github.com/MadBase/MadNet/consensus/objs"
	"github.com/MadBase/MadNet/logging"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/sirupsen/logrus"
)

//
type eventProcessor struct {
	name      string
	processor func(*objects.MonitorState, types.Log) error
}

// Services just a bundle of requirements common for monitoring functionality
type Services struct {
	logger            *logrus.Logger
	eth               interfaces.Ethereum
	consensusDb       *db.Database
	dph               *deposit.Handler
	ah                *admin.Handlers
	contractAddresses []common.Address
	batchSize         int
	eventMap          *objects.EventMap
	events            map[string]*eventProcessor
	taskMan           tasks.Manager
}

// NewServices creates a new Services struct
func NewServices(eth interfaces.Ethereum, db *db.Database, dph *deposit.Handler, ah *admin.Handlers, batchSize int) *Services {

	c := eth.Contracts()

	contractAddresses := []common.Address{
		c.DepositAddress(), c.EthdkgAddress(), c.RegistryAddress(),
		c.StakingTokenAddress(), c.UtilityTokenAddress(), c.ValidatorsAddress()}

	serviceLogger := logging.GetLogger("services")

	svcs := &Services{
		ah:                ah,
		batchSize:         batchSize,
		consensusDb:       db,
		contractAddresses: contractAddresses,
		dph:               dph,
		eth:               eth,
		eventMap:          objects.NewEventMap(),
		events:            make(map[string]*eventProcessor),
		logger:            serviceLogger,
		taskMan:           tasks.NewManager()}

	// Register handlers for known events, if this failed we really can't continue
	if err := SetupEventMap(svcs.eventMap); err != nil {
		panic(err)
	}

	// Below are the RegisterEvent()'s with nil fn's to improve logging by correlating a name with the topic
	if err := svcs.RegisterEvent("0x3529eeacda732ca25cee203cc6382b6d0688ee079ec8e53fd2dcbf259bdd3fa1", "DepositReceived-Obsolete", nil); err != nil {
		panic(err)
	}
	if err := svcs.RegisterEvent("0x6bae01a1b82866e1dfe8d98c42383fc58df9b4adeb47d7ac24ee4b53d409da6c", "DepositReceived-Obsolete", nil); err != nil {
		panic(err)
	}
	if err := svcs.RegisterEvent("0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925", "DSTokenApproval", nil); err != nil {
		panic(err)
	}
	if err := svcs.RegisterEvent("0xce241d7ca1f669fee44b6fc00b8eba2df3bb514eed0f6f668f8f89096e81ed94", "LogSetOwner", nil); err != nil {
		panic(err)
	}
	if err := svcs.RegisterEvent("0x0f6798a560793a54c3bcfe86a93cde1e73087d944c0ea20544137d4121396885", "Mint", nil); err != nil {
		panic(err)
	}
	if err := svcs.RegisterEvent("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef", "Transfer", nil); err != nil {
		panic(err)
	}
	if err := svcs.RegisterEvent("0x8c25e214c5693ebaf8008875bacedeb9e0aafd393864a314ed1801b2a4e13dd9", "ValidatorJoined", nil); err != nil {
		panic(err)
	}
	if err := svcs.RegisterEvent("0x319bbadb03b94aedc69babb34a28675536a9cb30f4bbde343e1d0018c44ebd94", "ValidatorLeft", nil); err != nil {
		panic(err)
	}
	if err := svcs.RegisterEvent("0x1de2f07b0a1c69916a8b25b889051644192307ea08444a2e11f8654d1db3ab0c", "LockedStake", nil); err != nil {
		panic(err)
	}

	// Real event processors are below
	if err := svcs.RegisterEvent("0x5b063c6569a91e8133fc6cd71d31a4ca5c65c652fd53ae093f46107754f08541", "DepositReceived", svcs.ProcessDepositReceived); err != nil {
		panic(err)
	}
	if err := svcs.RegisterEvent("0x113b129fac2dde341b9fbbec2bb79a95b9945b0e80fda711fc8ae5c7b0ea83b0", "ValidatorMember", svcs.ProcessValidatorMember); err != nil {
		panic(err)
	}
	if err := svcs.RegisterEvent("0x1c85ff1efe0a905f8feca811e617102cb7ec896aded693eb96366c8ef22bb09f", "ValidatorSet", svcs.ProcessValidatorSet); err != nil {
		panic(err)
	}
	if err := svcs.RegisterEvent("0x6d438b6b835d16cdae6efdc0259fdfba17e6aa32dae81863a2467866f85f724a", "SnapshotTaken", svcs.ProcessSnapshotTaken); err != nil {
		panic(err)
	}
	if err := svcs.RegisterEvent("0xa84d294194d6169652a99150fd2ef10e18b0d2caa10beeea237bbddcc6e22b10", "ShareDistribution", svcs.ProcessShareDistribution); err != nil {
		panic(err)
	}
	if err := svcs.RegisterEvent("0xb0ee36c3780de716eb6c83687f433ae2558a6923e090fd238b657fb6c896badc", "KeyShareSubmission", svcs.ProcessKeyShareSubmission); err != nil {
		panic(err)
	}
	if err := svcs.RegisterEvent("0x9c6f8368fe7e77e8cb9438744581403bcb3f53298e517f04c1b8475487402e97", "RegistrationOpen", svcs.ProcessRegistrationOpen); err != nil {
		panic(err)
	}

	ah.RegisterSnapshotCallback(svcs.PersistSnapshot) // HUNTER: moved out of main func and into constructor

	return svcs
}

// SetupEventMap populates map with known log topics
func SetupEventMap(em *objects.EventMap) error {

	// if err := em.RegisterLocked("0x3529eeacda732ca25cee203cc6382b6d0688ee079ec8e53fd2dcbf259bdd3fa1", "DepositReceived-Obsolete", nil); err != nil {
	// 	return err
	// }
	// if err := em.RegisterLocked("0x6bae01a1b82866e1dfe8d98c42383fc58df9b4adeb47d7ac24ee4b53d409da6c", "DepositReceived-Obsolete", nil); err != nil {
	// 	return err
	// }
	// if err := em.RegisterLocked("0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925", "DSTokenApproval", nil); err != nil {
	// 	return err
	// }
	// if err := em.RegisterLocked("0xce241d7ca1f669fee44b6fc00b8eba2df3bb514eed0f6f668f8f89096e81ed94", "LogSetOwner", nil); err != nil {
	// 	return err
	// }
	// if err := em.RegisterLocked("0x0f6798a560793a54c3bcfe86a93cde1e73087d944c0ea20544137d4121396885", "Mint", nil); err != nil {
	// 	return err
	// }
	// if err := em.RegisterLocked("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef", "Transfer", nil); err != nil {
	// 	return err
	// }
	// if err := em.RegisterLocked("0x8c25e214c5693ebaf8008875bacedeb9e0aafd393864a314ed1801b2a4e13dd9", "ValidatorJoined", nil); err != nil {
	// 	return err
	// }
	// if err := em.RegisterLocked("0x319bbadb03b94aedc69babb34a28675536a9cb30f4bbde343e1d0018c44ebd94", "ValidatorLeft", nil); err != nil {
	// 	return err
	// }
	// if err := em.RegisterLocked("0x1de2f07b0a1c69916a8b25b889051644192307ea08444a2e11f8654d1db3ab0c", "LockedStake", nil); err != nil {
	// 	return err
	// }

	// Real event processors are below
	// if err := em.RegisterLocked("0x5b063c6569a91e8133fc6cd71d31a4ca5c65c652fd53ae093f46107754f08541", "DepositReceived", svcs.ProcessDepositReceived); err != nil {
	// 	return err
	// }
	// if err := em.RegisterLocked("0x113b129fac2dde341b9fbbec2bb79a95b9945b0e80fda711fc8ae5c7b0ea83b0", "ValidatorMember", svcs.ProcessValidatorMember); err != nil {
	// 	return err
	// }
	// if err := em.RegisterLocked("0x1c85ff1efe0a905f8feca811e617102cb7ec896aded693eb96366c8ef22bb09f", "ValidatorSet", svcs.ProcessValidatorSet); err != nil {
	// 	return err
	// }
	// if err := em.RegisterLocked("0x6d438b6b835d16cdae6efdc0259fdfba17e6aa32dae81863a2467866f85f724a", "SnapshotTaken", svcs.ProcessSnapshotTaken); err != nil {
	// 	return err
	// }
	if err := em.RegisterLocked("0xa84d294194d6169652a99150fd2ef10e18b0d2caa10beeea237bbddcc6e22b10", "ShareDistribution", dkgevents.ProcessShareDistribution); err != nil {
		return err
	}
	if err := em.RegisterLocked("0xb0ee36c3780de716eb6c83687f433ae2558a6923e090fd238b657fb6c896badc", "KeyShareSubmission", dkgevents.ProcessKeyShareSubmission); err != nil {
		return err
	}
	if err := em.RegisterLocked("0x9c6f8368fe7e77e8cb9438744581403bcb3f53298e517f04c1b8475487402e97", "RegistrationOpen", dkgevents.ProcessOpenRegistration); err != nil {
		return err
	}

	return nil
}

// WatchEthereum checks state of Ethereum and processes interesting conditions
func (svcs *Services) WatchEthereum(state *objects.MonitorState) error {
	logger := svcs.logger
	eth := svcs.eth

	ctx, cancelFunc := eth.GetTimeoutContext()
	defer cancelFunc()

	// This is making sure Ethereum endpoint has synced and has peers
	// -- This doesn't care if _we_ are insync with Ethereum
	err := svcs.EndpointInSync(ctx, state)
	if err != nil {
		logger.Warnf("Failed checking if endpoint is synchronized: %v", err)
		state.CommunicationFailures++
		if state.CommunicationFailures >= uint32(svcs.eth.RetryCount()) {
			state.InSync = false
			svcs.ah.SetSynchronized(false)
		}
		return nil
	}
	state.CommunicationFailures = 0

	// If Ethereum is not in synced, it isn't an error but we can't go on
	if !state.EthereumInSync {
		s := state.Diff(state)
		if len(s) > 0 {
			logger.Warnf("...Ethereum endpoint not ready %s", s)
		}
		return nil
	}

	err = svcs.UpdateProgress(ctx, state)
	if err != nil {
		return err
	}

	// Decide what events to look for
	firstBlock := state.HighestBlockProcessed + 1
	lastBlock := state.HighestBlockProcessed + uint64(svcs.batchSize) // Be optimistic

	// Make sure we weren't too optimistic...
	finalizedHeight, err := eth.GetFinalizedHeight(ctx)
	if err != nil {
		return err
	}

	// This could happen if finality delay is too small
	if state.HighestBlockProcessed > finalizedHeight {
		logger.Warnf("Chain height shrank. Processed %v blocks but only %v are finalized.", state.HighestBlockProcessed, finalizedHeight)
		return nil
	}

	// Don't process anything past the finalized height
	if lastBlock > finalizedHeight {
		lastBlock = finalizedHeight
	}

	// No need to look for events if we're caught up
	if lastBlock >= firstBlock {

		logsByBlock := make(map[uint64][]types.Log)

		// Grab all the events in range
		logs, err := svcs.eth.GetEvents(ctx, firstBlock, lastBlock, svcs.contractAddresses)
		if err != nil {
			return err
		}

		// Find the blocks with events
		for _, log := range logs {
			bn := log.BlockNumber
			if la, ok := logsByBlock[bn]; ok {
				logsByBlock[bn] = append(la, log)
			} else {
				logsByBlock[bn] = []types.Log{log}
			}
		}

		// Interesting blocks can change based on an event, so we need to look at all blocks in range in order
		for block := firstBlock; block <= lastBlock; block++ {

			// If current block has any events, we process all of them
			if logs, present := logsByBlock[block]; present {
				for _, log := range logs {
					eventSelector := log.Topics[0].String()

					ep, ok := svcs.events[eventSelector]
					if ok {
						logger.Debugf("... block:%v event:%v name:%v", block, eventSelector, ep.name)
						if ep.processor != nil {
							err := ep.processor(state, log)
							if err != nil {
								logger.Errorf("Event handler for %v failed: %v", eventSelector, err)
							}
						}
					} else {
						logger.Debugf("... block:%v event:%v", block, eventSelector)
					}
				}
			}

			// Get the blocks currently interesting
			// if processor, present := state.interestingBlocks[block]; present {
			// 	logger.Debugf("... block:%v processor:%p", block, processor)
			// 	if present && processor != nil {
			// 		err := processor(state, block)
			// 		if err != nil {
			// 			logger.Warnf("Block handler for %v failed: %v", block, err)
			// 			// if err == dkg.ErrCanNotContinue {
			// 			// 	state.EthDKG = NewEthDKGState()
			// 			// 	state.interestingBlocks = make(map[uint64]func(*State, uint64) error)
			// 			// }
			// 		}
			// 	}
			// }

			state.HighestBlockProcessed = lastBlock
		}

		if lastBlock < finalizedHeight {
			state.InSync = false
			svcs.ah.SetSynchronized(false)
		} else {
			state.InSync = true
			svcs.ah.SetSynchronized(true)
		}

	}

	return nil
}

// RegisterEvent registers a handler for when an interesting event shows up
func (svcs *Services) RegisterEvent(selector string, name string, fn func(*objects.MonitorState, types.Log) error) error {

	svcs.events[selector] = &eventProcessor{processor: fn, name: name}
	return nil
}

// EndpointInSync Checks if our endpoint is good to use
// -- This function is different. Because we need to be aware of errors, state is always updated
func (svcs *Services) EndpointInSync(ctx context.Context, state *objects.MonitorState) error {

	// Default to assuming everything is awful
	state.EthereumInSync = false
	state.PeerCount = 0

	// Check if the endpoint is itself still syncing
	syncing, progress, err := svcs.eth.GetSyncProgress()
	if err != nil {
		svcs.logger.Warnf("Could not check if Ethereum endpoint it still syncing: %v", err)
		return err
	}

	if syncing && progress != nil {
		svcs.logger.Debugf("Ethereum endpoint syncing... at block %v of %v.",
			progress.CurrentBlock, progress.HighestBlock)
	}

	state.EthereumInSync = !syncing

	peerCount, err := svcs.eth.GetPeerCount(ctx)
	if err != nil {
		return err
	}
	state.PeerCount = uint32(peerCount)

	// TODO Remove direct reference to config. Specific values should be passed in.
	if state.EthereumInSync && state.PeerCount >= uint32(config.Configuration.Ethereum.EndpointMinimumPeers) {
		state.EthereumInSync = true
	}

	return nil
}

// UpdateProgress updates what we know of Ethereum chain height
func (svcs *Services) UpdateProgress(ctx context.Context, state *objects.MonitorState) error {
	height, err := svcs.eth.GetFinalizedHeight(ctx)
	if err != nil {
		return err
	}

	// Only updated single attribute so no need to copy -- Not sure if check is required
	state.HighestBlockFinalized = height
	return nil
}

// PersistSnapshot records the given block header on Ethereum and increments epoch
// TODO Returning an error kills the main loop, retry forever instead
func (svcs *Services) PersistSnapshot(blockHeader *objs.BlockHeader) error {

	eth := svcs.eth
	c := eth.Contracts()
	logger := svcs.logger

	// pull out the block claims
	bclaims := blockHeader.BClaims
	rawBclaims, err := bclaims.MarshalBinary()
	if err != nil {
		logger.Errorf("Could not extract BClaims from BlockHeader: %v", err)
		return nil //CAN NOT RETURN ERROR OR SUBSCRIPTION IS LOST!
	}

	// pull out the sig
	rawSigGroup := blockHeader.SigGroup

	// Do the mechanics
	txnOpts, err := svcs.eth.GetTransactionOpts(context.Background(), svcs.eth.GetDefaultAccount())
	if err != nil {
		logger.Errorf("Could not create transaction for snapshot: %v", err)
		return nil //CAN NOT RETURN ERROR OR SUBSCRIPTION IS LOST!
	}

	txn, err := c.Validators().Snapshot(txnOpts, rawSigGroup, rawBclaims)
	if err != nil {
		logger.Errorf("Failed to take snapshot: %v", err)
		return nil //CAN NOT RETURN ERROR OR SUBSCRIPTION IS LOST!
	}

	toCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	rcpt, err := eth.Queue().QueueAndWait(toCtx, txn)
	if err != nil {
		logger.Errorf("Failed to retrieve snapshot receipt: %v", err)
		return nil //CAN NOT RETURN ERROR OR SUBSCRIPTION IS LOST!
	}

	if rcpt == nil {
		logger.Warnf("No receipt from snapshot")
	} else {
		if rcpt.Status != uint64(1) {
			logger.Errorf("Snapshot receipt shows failure.")
			return nil //CAN NOT RETURN ERROR OR SUBSCRIPTION IS LOST!
		}
	}

	return nil
}

// SetBN256PrivateKey informs the admin bus of the BN256 private key
func (svcs *Services) SetBN256PrivateKey(pk []byte) error {
	return svcs.ah.AddPrivateKey(pk, 2)
}

// SetSECP256K1PrivateKey informs the admin bus of the SECP256K1 private key
func (svcs *Services) SetSECP256K1PrivateKey(pk []byte) error {
	return svcs.ah.AddPrivateKey(pk, 1)
}

// AbortETHDKG does the required cleanup to stop a round of ETHDKG
// func AbortETHDKG(ethdkg *EthDKGState) {
// 	handlers := []tasks.TaskHandler{
// 		ethdkg.RegistrationTH,
// 		ethdkg.ShareDistributionTH,
// 		ethdkg.DisputeTH,
// 		ethdkg.KeyShareSubmissionTH,
// 		ethdkg.MPKSubmissionTH,
// 		ethdkg.GPKJSubmissionTH,
// 		ethdkg.GPKJGroupAccusationTH,
// 		ethdkg.CompleteTH}

// 	// We need to cancel any handler that might be running
// 	for _, handler := range handlers {
// 		if handler != nil {
// 			handler.Cancel()
// 		}
// 	}

// 	// Erase the schedule
// 	ethdkg.Schedule = &EthDKGSchedule{}
// }

// ETHDKGInProgress indicates if ETHDKG is currently running
// func ETHDKGInProgress(ethdkg *EthDKGState, currentBlock uint64) bool {
// 	if ethdkg == nil {
// 		return false
// 	}

// 	return currentBlock >= ethdkg.Schedule.RegistrationStart &&
// 		currentBlock <= ethdkg.Schedule.CompleteEnd
// }
