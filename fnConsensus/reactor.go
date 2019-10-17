package fnConsensus

import (
	"bytes"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/go-kit/kit/metrics"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/state"
	"github.com/tendermint/tendermint/types"
)

type SigningThreshold string

const (
	Maj23SigningThreshold SigningThreshold = "Maj23"
	AllSigningThreshold   SigningThreshold = "All"
)

// MethodIDs for tracing purpose
const (
	initValidatorSetMethodID  = "initValidatorSet"
	voteMethodID              = "vote"
	commitMethodID            = "commit"
	maj23MsgHandlerMethodID   = "handleMaj23Msg"
	voteSetMsgHandlerMethodID = "handleVoteSetMsg"
)

const (
	// Channel IDs need to be unique across all the reactors.
	// To avoid conflict with other reactors and to give Tendermint some wiggle room when they
	// add more channels we're using channel IDs from 0x50 onwards for this reactor.

	// FnVoteSetChannel is used to gossip votesets
	FnVoteSetChannel = byte(0x50)
	// FnMajChannel is used to gossip votesets that have reached 2/3+ majority
	FnMajChannel = byte(0x51)

	// MaxMsgSize is the max number of bytes that can sent on a P2P channel
	MaxMsgSize = 2 * 1000 * 1024 // 2MB

	// Denotes interval (synced across nodes) between two proposals
	proposeIntervalInSeconds int64 = 10
	commitIntervalInSeconds  int64 = 5

	// Delay between propogating votesets to update other peers
	voteSetPropogationDelay = 1 * time.Second

	// Time to wait between attempts to load TM state from state.db on startup
	progressLoopStartDelay = 2 * time.Second
)

type FnConsensusReactor struct {
	p2p.BaseReactor

	connectedPeers map[p2p.ID]p2p.Peer
	peerMapMtx     sync.RWMutex

	state    *ReactorState
	stateMtx sync.Mutex

	db        dbm.DB // fnConsensus.db
	tmStateDB dbm.DB // TM state.db to load current validator set from
	chainID   string

	fnRegistry FnRegistry

	privValidator    types.PrivValidator // used to sign votes
	staticValidators *types.ValidatorSet // overrides the TM validator set if not nil

	cfg *ReactorConfig
}

var (
	submittedMessageCount metrics.Counter
	nonceGauge            metrics.Gauge
)

func init() {
	submittedMessageCount = kitprometheus.NewCounterFrom(
		stdprometheus.CounterOpts{
			Namespace: "loomchain",
			Subsystem: "fnConsensus",
			Name:      "submitted_message_count",
			Help:      "Number of messages successfully submitted by the validator (per fnID)",
		}, []string{"fnID"},
	)
	nonceGauge = kitprometheus.NewGaugeFrom(
		stdprometheus.GaugeOpts{
			Namespace: "loomchain",
			Subsystem: "fnConsensus",
			Name:      "current_nonce",
			Help:      "Current nonce (per fnID)",
		}, []string{"fnID"},
	)
}

func NewFnConsensusReactor(
	chainID string, privValidator types.PrivValidator, fnRegistry FnRegistry, db dbm.DB, tmStateDB dbm.DB,
	parsableConfig *ReactorConfigParsable,
) (*FnConsensusReactor, error) {
	parsedConfig, err := parsableConfig.Parse()
	if err != nil {
		return nil, err
	}

	reactor := &FnConsensusReactor{
		connectedPeers: make(map[p2p.ID]p2p.Peer),
		db:             db,
		chainID:        chainID,
		tmStateDB:      tmStateDB,
		fnRegistry:     fnRegistry,
		privValidator:  privValidator,
		cfg:            parsedConfig,
	}

	reactor.BaseReactor = *p2p.NewBaseReactor("FnConsensusReactor", reactor)
	return reactor, nil
}

func (f *FnConsensusReactor) safeSubmitMultiSignedMessage(fnID string, fn Fn, message []byte, signatures [][]byte) {
	defer func() {
		err := recover()
		if err != nil {
			f.Logger.Error("panicked while invoking SubmitMultiSignedMessage", "error", err)
		}
	}()
	fn.SubmitMultiSignedMessage(nil, message, signatures)
	submittedMessageCount.With("fnID", fnID).Add(1)
}

// Returns a message and associated signature (which can be anything really).
func (f *FnConsensusReactor) safeGetMessageAndSignature(fn Fn) ([]byte, []byte, error) {
	defer func() {
		err := recover()
		if err != nil {
			f.Logger.Error("panicked while invoking GetMessageAndSignature", "error", err)
		}
	}()
	return fn.GetMessageAndSignature(nil)
}

func (f *FnConsensusReactor) String() string {
	return "FnConsensusReactor"
}

// OnStart implements BaseReactor by loading the previously persisted reactor state from fnConsensus.db,
// loading the current validator set, and starting the vote & commit go-routines.
func (f *FnConsensusReactor) OnStart() error {
	reactorState, err := loadReactorState(f.db)
	if err != nil {
		return err
	}

	f.state = reactorState

	go f.initRoutine()

	return nil
}

// GetChannels implements BaseReactor by returning a list of channel descriptors.
func (f *FnConsensusReactor) GetChannels() []*p2p.ChannelDescriptor {
	// Priorities are deliberately set to low, to prevent interfering with core TM
	return []*p2p.ChannelDescriptor{
		{
			ID:                  FnMajChannel,
			Priority:            20,
			SendQueueCapacity:   100,
			RecvMessageCapacity: MaxMsgSize,
		},
		{
			ID:                  FnVoteSetChannel,
			Priority:            25,
			SendQueueCapacity:   100,
			RecvMessageCapacity: MaxMsgSize,
		},
	}
}

// AddPeer implements BaseReactor, it will be called by the switch when a new peer is added.
func (f *FnConsensusReactor) AddPeer(peer p2p.Peer) {
	f.peerMapMtx.Lock()
	f.connectedPeers[peer.ID()] = peer
	f.peerMapMtx.Unlock()
}

// RemovePeer implements BaseReactor, it will be called by the switch when a peer is stopped
// (due to error or other reason).
func (f *FnConsensusReactor) RemovePeer(peer p2p.Peer, reason interface{}) {
	f.peerMapMtx.Lock()
	defer f.peerMapMtx.Unlock()
	delete(f.connectedPeers, peer.ID())
}

// Sends the given msgBytes on the given channel to all peers, with one possible exception.
func (f *FnConsensusReactor) broadcastMsgSync(chID byte, exception *p2p.ID, msgBytes []byte) {
	f.peerMapMtx.RLock()
	defer f.peerMapMtx.RUnlock()

	for _, peer := range f.connectedPeers {
		if exception != nil && (*exception) == peer.ID() {
			continue
		}
		peer.Send(chID, msgBytes)
	}
}

func (f *FnConsensusReactor) myAddress() []byte {
	return f.privValidator.GetPubKey().Address()
}

func (f *FnConsensusReactor) areWeValidator(currentValidatorSet *types.ValidatorSet) (bool, int) {
	validatorIndex, _ := currentValidatorSet.GetByAddress(f.myAddress())
	return validatorIndex != -1, validatorIndex
}

func calculateMessageHash(message []byte) ([]byte, error) {
	hash := sha512.New()
	_, err := hash.Write(message)
	if err != nil {
		return nil, err
	}
	return hash.Sum(nil), nil
}

func calculateSleepTimeForCommit(areWeValidator bool) time.Duration {
	currentEpochTime := time.Now().Unix()
	baseTimeToSleep := commitIntervalInSeconds - currentEpochTime%commitIntervalInSeconds

	const maxBoundForVariableComponent = 2 * time.Second
	const baseCommitDelay = 100 * time.Millisecond

	if !areWeValidator {
		return (time.Duration(baseTimeToSleep) * time.Second) + baseCommitDelay
	}

	return (time.Duration(baseTimeToSleep) * time.Second) +
		time.Duration(rand.Int63n(int64(maxBoundForVariableComponent))) +
		baseCommitDelay
}

func calculateSleepTimeForPropose(areWeValidator bool) time.Duration {
	currentEpochTime := time.Now().Unix()
	baseTimeToSleep := proposeIntervalInSeconds - currentEpochTime%proposeIntervalInSeconds

	const baseProposalDelay = 500 * time.Millisecond
	const maxBoundForVariableComponent = 2 * time.Second

	if !areWeValidator {
		return (time.Duration(baseTimeToSleep) * time.Second) + baseProposalDelay
	}

	return (time.Duration(baseTimeToSleep) * time.Second) +
		time.Duration(rand.Int63n(int64(maxBoundForVariableComponent))) +
		baseProposalDelay
}

// Loads staticValidators if OverrideValidators setting is specified in the config.
func (f *FnConsensusReactor) initValidatorSet(tmState state.State) error {
	if len(f.cfg.OverrideValidators) == 0 {
		f.Logger.Info("FnConsensusReactor: using DPoS validator set for consensus", "method", initValidatorSetMethodID)
		return nil
	}

	validatorArray := make([]*types.Validator, 0, len(f.cfg.OverrideValidators))

	for _, overrideValidator := range f.cfg.OverrideValidators {
		// tmState.Validators is the tendermint address, not the loom address.
		validatorIndex, validator := tmState.Validators.GetByAddress(overrideValidator.Address)
		if validatorIndex == -1 {
			return fmt.Errorf("validator specified in override config, doesnt exist in TM validator set")
		}
		// We need to overwrite DPoS voting power with static one
		// otherwise there is possibility of validator hash disagreement
		// among nodes, if one or more nodes restarts. This happens due to
		// recalculation of validator set on every election.
		validator.VotingPower = overrideValidator.VotingPower

		f.Logger.Info("FnConsensusReactor: adding validator to static validator set", "validator", validator.String(),
			"method", initValidatorSetMethodID)

		validatorArray = append(validatorArray, validator)
	}

	f.staticValidators = types.NewValidatorSet(validatorArray)

	f.Logger.Info("FnConsensusReactor: using static validator set for consensus", "validatorSetHash",
		hex.EncodeToString(f.staticValidators.Hash()),
		"method", initValidatorSetMethodID)

	return nil
}

func (f *FnConsensusReactor) getValidatorSet() *types.ValidatorSet {
	if f.staticValidators == nil {
		tmState := state.LoadState(f.tmStateDB)
		return tmState.Validators
	}

	return f.staticValidators
}

func (f *FnConsensusReactor) initRoutine() {
	var currentState state.State

	// Wait till state is populated
	for currentState = state.LoadState(f.tmStateDB); currentState.IsEmpty(); currentState = state.LoadState(f.tmStateDB) {
		f.Logger.Error("TM state is empty. Cant start progress loop, retrying in some time...")
		time.Sleep(progressLoopStartDelay)
	}

	if err := f.initValidatorSet(currentState); err != nil {
		f.Logger.Error("error while initializing reactor", "err", err)
		f.Stop()
		return
	}

	go f.voteRoutine()
	go f.commitRoutine()
}

func (f *FnConsensusReactor) commitRoutine() {
	defer func() {
		if r := recover(); r != nil {
			f.Logger.Error("Recovered in FnConsensusReactor.commitRoutine", "r", r)
		}
	}()
	currentValidators := f.getValidatorSet()

	// Initializing these vars with sane value to calculate initial time
	areWeValidator, _ := f.areWeValidator(currentValidators)

OUTER_LOOP:
	for {
		commitSleepTime := calculateSleepTimeForCommit(areWeValidator)
		commitTimer := time.NewTimer(commitSleepTime)

		select {
		case <-f.Quit():
			commitTimer.Stop()
			break OUTER_LOOP
		case <-commitTimer.C:
			fnIDs := f.fnRegistry.GetAll()
			sort.Strings(fnIDs)

			fnsEligibleForCommit := make([]string, 0, len(fnIDs))

			f.stateMtx.Lock()
			for _, fnID := range fnIDs {
				currentVoteState := f.state.CurrentVoteSets[fnID]
				if currentVoteState == nil {
					continue
				}
				fnsEligibleForCommit = append(fnsEligibleForCommit, fnID)
			}
			f.stateMtx.Unlock()

			for _, fnID := range fnsEligibleForCommit {
				f.commit(fnID)
			}
		}
	}
}

func (f *FnConsensusReactor) voteRoutine() {
	defer func() {
		if r := recover(); r != nil {
			f.Logger.Error("Recovered in FnConsensusReactor.voteRoutine", "r", r)
		}
	}()

	currentValidators := f.getValidatorSet()

	// Initializing these vars with sane value to calculate initial time
	areWeValidator, _ := f.areWeValidator(currentValidators)

OUTER_LOOP:
	for {
		// Align to minutes, to make sure this routine runs at almost same time across all nodes
		// Not strictly required
		// state and other variables will be same as the one initialized in second case statement
		proposeSleepTime := calculateSleepTimeForPropose(areWeValidator)
		proposeTimer := time.NewTimer(proposeSleepTime)

		select {
		case <-f.Quit():
			proposeTimer.Stop()
			break OUTER_LOOP
		case <-proposeTimer.C:
			currentValidators := f.getValidatorSet()
			areWeValidator, ownValidatorIndex := f.areWeValidator(currentValidators)

			if !areWeValidator {
				break
			}

			fnIDs := f.fnRegistry.GetAll()
			sort.Strings(fnIDs)

			fnsEligibleForVoting := make([]string, 0, len(fnIDs))

			f.stateMtx.Lock()
			for _, fnID := range fnIDs {
				currentVoteState := f.state.CurrentVoteSets[fnID]
				if currentVoteState != nil {
					f.Logger.Info("FnConsensusReactor: unable to vote, execution is in progress", "FnID", fnID)
					continue
				}
				fnsEligibleForVoting = append(fnsEligibleForVoting, fnID)
			}
			f.stateMtx.Unlock()

			for _, fnID := range fnsEligibleForVoting {
				fn := f.fnRegistry.Get(fnID)
				f.vote(fnID, fn, currentValidators, ownValidatorIndex)
			}
		}
	}
}

// Creates a vote signed by the validator corresponding to the given index and broadcasts it to all peers.
func (f *FnConsensusReactor) vote(fnID string, fn Fn, currentValidators *types.ValidatorSet, validatorIndex int) {
	message, signature, err := f.safeGetMessageAndSignature(fn)
	if err != nil {
		f.Logger.Error(
			"FnConsensusReactor: received error while executing fn.GetMessageAndSignature",
			"fnID", fnID, "err", err, "method", voteMethodID,
		)
		return
	}

	hash, err := calculateMessageHash(message)
	if err != nil {
		f.Logger.Error(
			"FnConsensusReactor: unable to calculate message hash",
			"fnID", fnID, "err", err, "method", voteMethodID,
		)
		return
	}

	executionRequest, err := NewFnExecutionRequest(fnID, f.fnRegistry)
	if err != nil {
		f.Logger.Error(
			"FnConsensusReactor: unable to create Fn execution request",
			"fnID", fnID, "err", err, "method", voteMethodID,
		)
		return
	}

	executionResponse := NewFnExecutionResponse(&FnIndividualExecutionResponse{
		Hash:            hash,
		OracleSignature: signature, // TODO: reactor shouldn't know anything about oracles
	}, validatorIndex, currentValidators)

	f.stateMtx.Lock()
	defer f.stateMtx.Unlock()

	f.state.Messages[fnID] = Message{
		Payload: message,
		Hash:    hash,
	}

	currentNonce, ok := f.state.CurrentNonces[fnID]
	if !ok {
		currentNonce = 1
	}

	voteSet, err := NewVoteSet(
		currentNonce,
		f.chainID,
		validatorIndex,
		NewFnVotePayload(executionRequest, executionResponse),
		f.privValidator,
		currentValidators,
	)
	if err != nil {
		f.Logger.Error(
			"FnConsensusReactor: unable to create new voteset",
			"fnID", fnID, "err", err, "method", voteMethodID,
		)
		return
	}

	// Have we achieved Maj23 already?
	aggregateExecutionResponse := voteSet.MajResponse(f.cfg.FnVoteSigningThreshold, currentValidators)
	if aggregateExecutionResponse != nil {
		if !bytes.Equal(f.state.Messages[fnID].Hash, aggregateExecutionResponse.Hash) {
			f.Logger.Error(
				"FnConsensusReactor: message hash mismatch",
				"fnID", fnID, "method", voteMethodID, "nonce", currentNonce, "validator", validatorIndex,
			)
			return
		}
		f.safeSubmitMultiSignedMessage(
			fnID,
			fn,
			safeCopyBytes(f.state.Messages[fnID].Payload),
			safeCopyDoubleArray(aggregateExecutionResponse.OracleSignatures),
		)
		return
	}

	f.state.CurrentVoteSets[fnID] = voteSet

	if err := saveReactorState(f.db, f.state, true); err != nil {
		f.Logger.Error(
			"FnConsensusReactor: unable to save state",
			"fnID", fnID, "err", err, "method", voteMethodID,
		)
		return
	}

	marshalledBytes, err := voteSet.Marshal()
	if err != nil {
		f.Logger.Error(
			"FnConsensusReactor: Unable to marshal currentVoteSet",
			"fnID", fnID, "err", err, "method", voteMethodID,
		)
		return
	}

	// NOTE: f.state is still locked at this point, so until the broadcast is complete we won't be able
	// to receive any votesets from anyone else because both handleVoteSetChannelMessage and
	// handleMaj23VoteSetChannel must acquire the f.state lock before they can do anything of substance.
	f.broadcastMsgSync(FnVoteSetChannel, nil, marshalledBytes)
}

// Checks if the signing threshold has been reached (2/3+ majority usually) in the current voteset,
// if it has been SubmitMultiSignedMessage will be invoked for the given fnID. If the threshold hasn't
// been reached both the previous voteset that reached the threshold and the current voteset
// will be broadcast to all peers to help them move towards consensus.
// TODO: Double-check the Ethereum Gateway uses a similar algo to calculate the threshold, otherwise
//       we could end up in a situation where 2/3+ majority is reached here but the threshold calculated
//       by the Ethereum Gateway is slightly more than that.
func (f *FnConsensusReactor) commit(fnID string) {
	fn := f.fnRegistry.Get(fnID)
	if fn == nil {
		f.Logger.Error(
			"FnConsensusReactor: fn is nil while trying to access it in commit routine, ignoring...",
			"method", commitMethodID,
		)
		return
	}

	currentValidators := f.getValidatorSet()
	areWeValidator, ownValidatorIndex := f.areWeValidator(currentValidators)

	f.stateMtx.Lock()
	defer f.stateMtx.Unlock()

	currentVoteSet := f.state.CurrentVoteSets[fnID]
	currentNonce := f.state.CurrentNonces[fnID]

	if err := currentVoteSet.IsValid(f.chainID, currentValidators, f.fnRegistry); err != nil {
		f.Logger.Error(
			"FnConsensusReactor: Invalid VoteSet found",
			"VoteSet", currentVoteSet, "err", err, "method", commitMethodID)

		delete(f.state.CurrentVoteSets, fnID)

		if err := saveReactorState(f.db, f.state, true); err != nil {
			f.Logger.Error(
				"FnConsensusReactor: unable to save state",
				"fnID", fnID, "err", err, "method", commitMethodID,
			)
			return
		}
		return
	}

	if !currentVoteSet.HasConverged(f.cfg.FnVoteSigningThreshold, currentValidators) {
		f.Logger.Info(
			"No consensus achieved",
			"fnID", fnID, "VoteSet", currentVoteSet, "Payload", currentVoteSet.Payload,
			"Response", currentVoteSet.Payload.Response, "method", commitMethodID,
		)

		previousConvergedVoteSet := f.state.PreviousMajVoteSets[fnID]
		if previousConvergedVoteSet != nil {
			marshalledBytesOfPreviousVoteSet, err := previousConvergedVoteSet.Marshal()
			if err != nil {
				f.Logger.Error(
					"unable to marshal PreviousMajVoteSet",
					"err", err, "fnID", fnID, "method", commitMethodID,
				)
				return
			}

			marshalledBytesOfCurrentVoteSet, err := currentVoteSet.Marshal()
			if err != nil {
				f.Logger.Error(
					"unable to marshal Current Vote set",
					"err", err, "fnID", fnID, "method", commitMethodID,
				)
				return
			}

			// Propagate your last Maj23, to remedy any issue
			f.broadcastMsgSync(FnMajChannel, nil, marshalledBytesOfPreviousVoteSet)

			time.Sleep(voteSetPropogationDelay)

			// Propagate your current voteSet, to get newly joined node to sign it
			f.broadcastMsgSync(FnVoteSetChannel, nil, marshalledBytesOfCurrentVoteSet)
		}
	} else {
		if areWeValidator {
			majExecutionResponse := currentVoteSet.MajResponse(f.cfg.FnVoteSigningThreshold, currentValidators)
			if majExecutionResponse != nil {
				f.Logger.Info(
					"Maj-consensus achieved",
					"fnID", fnID, "VoteSet", currentVoteSet, "Payload", currentVoteSet.Payload,
					"Response", currentVoteSet.Payload.Response, "method", commitMethodID,
				)
				numberOfAgreeVotes := majExecutionResponse.NumberOfAgreeVotes()
				agreeVoteIndex := majExecutionResponse.AgreeIndex(ownValidatorIndex)
				// The consensus result only needs to be sent to the cluster by a single validator,
				// that validator is chosen in a round-robin fashion every voting round.
				if agreeVoteIndex != -1 && (currentNonce%int64(numberOfAgreeVotes)) == int64(agreeVoteIndex) {
					if !bytes.Equal(f.state.Messages[fnID].Hash, majExecutionResponse.Hash) {
						f.Logger.Error(
							"FnConsensusReactor: message hash mismatch",
							"fnID", fnID, "method", commitMethodID, "nonce", currentNonce,
							"validator", ownValidatorIndex,
						)
						return
					}
					f.Logger.Info("FnConsensusReactor: Submitting Multisigned message")
					f.safeSubmitMultiSignedMessage(
						fnID,
						fn,
						safeCopyBytes(f.state.Messages[fnID].Payload),
						safeCopyDoubleArray(majExecutionResponse.OracleSignatures),
					)
				}
			}
		}

		f.state.CurrentNonces[fnID]++
		nonceGauge.With("fnID", fnID).Set(float64(f.state.CurrentNonces[fnID]))
		f.state.PreviousValidatorSet = currentValidators
		f.state.PreviousMajVoteSets[fnID] = currentVoteSet
		delete(f.state.CurrentVoteSets, fnID)
	}

	if err := saveReactorState(f.db, f.state, true); err != nil {
		f.Logger.Error("FnConsensusReactor: unable to save state", "fnID", fnID, "err", err, "method", commitMethodID)
		return
	}
}

// Compares the trustworthiness of a voteset received from a peer to the current local voteset.
// Returns zero if both votesets have the same trustworthiness, 1 if the remote voteset is more trustworthy,
// or -1 if the local voteset is more trustworthy.
func (f *FnConsensusReactor) compareFnVoteSets(
	remoteVoteSet *FnVoteSet,
	currentVoteSet *FnVoteSet,
	currentNonce int64,
	currentValidators *types.ValidatorSet,
) int {
	if currentVoteSet == nil {
		if currentNonce == remoteVoteSet.Nonce {
			return 1
		}

		if remoteVoteSet.HasConverged(f.cfg.FnVoteSigningThreshold, currentValidators) {
			return 1
		}

		return -1
	}

	if currentVoteSet.Nonce == remoteVoteSet.Nonce {
		return 0
	}

	currentVoteSetConverged := currentVoteSet.HasConverged(f.cfg.FnVoteSigningThreshold, currentValidators)
	remoteVoteSetConverged := remoteVoteSet.HasConverged(f.cfg.FnVoteSigningThreshold, currentValidators)

	if currentVoteSetConverged && !remoteVoteSetConverged {
		return -1
	} else if !currentVoteSetConverged && remoteVoteSetConverged {
		return 1
	} else if !currentVoteSetConverged && !remoteVoteSetConverged {
		return -1
	}

	currentNumberOfVotes := currentVoteSet.NumberOfVotes()
	remoteNumberOfVotes := remoteVoteSet.NumberOfVotes()

	if remoteNumberOfVotes < currentNumberOfVotes {
		return -1
	} else if remoteNumberOfVotes > currentNumberOfVotes {
		return 1
	}

	currentMajResponse := currentVoteSet.MajResponse(f.cfg.FnVoteSigningThreshold, currentValidators)
	remoteMajResponse := remoteVoteSet.MajResponse(f.cfg.FnVoteSigningThreshold, currentValidators)

	currentMajAgreed := currentMajResponse != nil
	remoteMajAgreed := remoteMajResponse != nil

	if currentMajAgreed && !remoteMajAgreed {
		return -1
	} else if !currentMajAgreed && remoteMajAgreed {
		return 1
	} else if !currentMajAgreed && !remoteMajAgreed {
		return -1
	}

	currentMajResponseAgreedVotes := currentMajResponse.NumberOfAgreeVotes()
	remoteMajResponseAgreedVotes := remoteMajResponse.NumberOfAgreeVotes()

	if remoteMajResponseAgreedVotes < currentMajResponseAgreedVotes {
		return -1
	} else if remoteMajResponseAgreedVotes > currentMajResponseAgreedVotes {
		return 1
	}

	// If everything is same, we will trust current vote set
	return -1
}

func (f *FnConsensusReactor) handleMaj23VoteSetChannel(sender p2p.Peer, msgBytes []byte) {
	f.stateMtx.Lock()
	defer f.stateMtx.Unlock()

	currentValidatorSet := f.getValidatorSet()
	previousValidatorSet := f.state.PreviousValidatorSet

	validatorSetWhichSignedRemoteVoteSet := currentValidatorSet

	remoteMajVoteSet := &FnVoteSet{}
	if err := remoteMajVoteSet.Unmarshal(msgBytes); err != nil {
		f.Logger.Error(
			"FnConsensusReactor: Invalid Data passed, ignoring...",
			"err", err, "method", maj23MsgHandlerMethodID,
		)
		return
	}

	if f.cfg.FnConcensusSignerCfg.Enabled {
		// check if this node part of validator, if not broadcast the msg
		if f.cfg.FnConcensusSignerCfg.Validator == false {
			f.broadcastMsgSync(FnVoteSetChannel, nil, msgBytes)
		}
	}

	// We might have recently changed validator set, so maybe this voteset is valid with
	// previousValidatorSet and not current. We dont need to validate the proposer, as it might be
	// outdated in our case.
	if err := remoteMajVoteSet.IsValid(f.chainID, currentValidatorSet, f.fnRegistry); err != nil {
		if previousValidatorSet == nil {
			f.Logger.Error(
				"FnConsensusReactor: Invalid VoteSet specified, ignoring...",
				"err", err, "method", maj23MsgHandlerMethodID,
			)
			return
		}
		if err := remoteMajVoteSet.IsValid(f.chainID, previousValidatorSet, f.fnRegistry); err != nil {
			f.Logger.Error(
				"FnConsensusReactor: Invalid VoteSet specified, ignoring...",
				"err", err, "method", maj23MsgHandlerMethodID,
			)
			return
		}
		validatorSetWhichSignedRemoteVoteSet = previousValidatorSet
	}

	remoteFnID := remoteMajVoteSet.GetFnID()
	currentNonce, ok := f.state.CurrentNonces[remoteFnID]
	if !ok {
		currentNonce = 1
	}

	previousMaj23VoteSet := f.state.PreviousMajVoteSets[remoteFnID]
	needToBroadcast := true

	if !remoteMajVoteSet.HasConverged(f.cfg.FnVoteSigningThreshold, validatorSetWhichSignedRemoteVoteSet) {
		f.Logger.Error("FnConsensusReactor: got non maj23 voteset, Ignoring...", "method", maj23MsgHandlerMethodID)
		return
	}

	if remoteMajVoteSet.Nonce < currentNonce {
		needToBroadcast = false
		if remoteMajVoteSet.Nonce == currentNonce-1 {
			if previousMaj23VoteSet == nil {
				previousMaj23VoteSet = remoteMajVoteSet
				f.state.PreviousMajVoteSets[remoteFnID] = remoteMajVoteSet
				f.state.PreviousValidatorSet = validatorSetWhichSignedRemoteVoteSet
			}
		}
	} else {
		// Remote Maj23 is at nonce `x`. So, current nonce must be `x` + 1.
		previousMaj23VoteSet = remoteMajVoteSet
		f.state.PreviousMajVoteSets[remoteFnID] = remoteMajVoteSet
		f.state.PreviousValidatorSet = validatorSetWhichSignedRemoteVoteSet
		f.state.CurrentNonces[remoteFnID] = remoteMajVoteSet.Nonce + 1
		nonceGauge.With("fnID", remoteFnID).Set(float64(f.state.CurrentNonces[remoteFnID]))

		// If we have found maj23 voteset with a nonce equal or greater than our current nonce,
		// our current vote set is clearly outdated, and should be removed.
		delete(f.state.CurrentVoteSets, remoteFnID)

		// NOTE: f.safeSubmitMultiSignedMessage is not invoked here presumably because it was already
		// invoked by the peers that we got the remote voteset from.
	}

	if err := saveReactorState(f.db, f.state, true); err != nil {
		f.Logger.Error(
			"FnConsensusReactor: unable to save reactor state",
			"err", err, "method", maj23MsgHandlerMethodID,
		)
		return
	}

	if !needToBroadcast {
		return
	}

	marshalledBytes, err := previousMaj23VoteSet.Marshal()
	if err != nil {
		f.Logger.Error(
			"FnConsensusReactor: unable to marshal bytes",
			"err", err, "method", maj23MsgHandlerMethodID,
		)
		return
	}

	// TODO: If remote nonce is greater or equal to ours then we end up sending the remote voteset back
	//       to the peer that sent it to us, we should we exclude that peer from the broadcast instead.
	f.broadcastMsgSync(FnMajChannel, nil, marshalledBytes)
}

func (f *FnConsensusReactor) handleVoteSetChannelMessage(sender p2p.Peer, msgBytes []byte) {
	currentValidators := f.getValidatorSet()
	areWeValidator, ownValidatorIndex := f.areWeValidator(currentValidators)

	remoteVoteSet := &FnVoteSet{}
	if err := remoteVoteSet.Unmarshal(msgBytes); err != nil {
		f.Logger.Error(
			"FnConsensusReactor: Invalid Data passed, ignoring...",
			"err", err, "method", voteSetMsgHandlerMethodID,
		)
		return
	}

	if f.cfg.FnConcensusSignerCfg.Enabled {
		// check if this node part of validator, if not broadcast the msg
		if f.cfg.FnConcensusSignerCfg.Validator == false {
			f.broadcastMsgSync(FnVoteSetChannel, nil, msgBytes)
		}
	}

	fnID := remoteVoteSet.GetFnID()

	if err := remoteVoteSet.IsValid(f.chainID, currentValidators, f.fnRegistry); err != nil {
		f.Logger.Error(
			"FnConsensusReactor: Invalid VoteSet specified, ignoring...",
			"err", err, "method", voteSetMsgHandlerMethodID,
		)
		return
	}

	f.stateMtx.Lock()
	defer f.stateMtx.Unlock()

	currentNonce, ok := f.state.CurrentNonces[fnID]
	if !ok {
		currentNonce = 1
		f.state.CurrentNonces[fnID] = currentNonce
		nonceGauge.With("fnID", fnID).Set(float64(currentNonce))
	}
	currentVoteSet := f.state.CurrentVoteSets[fnID]

	if currentNonce > remoteVoteSet.Nonce {
		f.Logger.Info(
			"FnConsensusReactor: Already seen this nonce, ignoring",
			"currentNonce", currentNonce,
			"remoteNonce", remoteVoteSet.Nonce,
		)
		return
	}

	var didWeContribute, hasOurVoteSetChanged bool
	var err error

	switch f.compareFnVoteSets(remoteVoteSet, currentVoteSet, currentNonce, currentValidators) {
	// Both votesets have same trustworthiness, so merge
	case 0:
		if didWeContribute, err = currentVoteSet.Merge(currentValidators, remoteVoteSet); err != nil {
			f.Logger.Error(
				"FnConsensusReactor: Unable to merge remote vote set into our own.",
				"err", err, "method", voteSetMsgHandlerMethodID,
			)
			return
		}
		hasOurVoteSetChanged = didWeContribute

	// Remote voteset is more trustworthy, so replace
	case 1:
		f.state.CurrentVoteSets[fnID] = remoteVoteSet
		f.state.CurrentNonces[fnID] = remoteVoteSet.Nonce

		currentVoteSet = remoteVoteSet
		currentNonce = remoteVoteSet.Nonce
		nonceGauge.With("fnID", fnID).Set(float64(currentNonce))

		hasOurVoteSetChanged = true
		didWeContribute = false

	// Current voteset is more trustworthy
	case -1:
		if currentVoteSet == nil {
			return
		}
	}

	if areWeValidator && !currentVoteSet.HaveWeAlreadySigned(ownValidatorIndex) {
		fn := f.fnRegistry.Get(fnID)

		message, signature, err := f.safeGetMessageAndSignature(fn)
		if err != nil {
			f.Logger.Error(
				"FnConsensusReactor: received error while executing fn.GetMessageAndSignature",
				"fnID", fnID, "err", err, "method", voteSetMsgHandlerMethodID,
			)
			return
		}

		hash, err := calculateMessageHash(message)
		if err != nil {
			f.Logger.Error(
				"FnConsensusReactor: unable to calculate message hash",
				"fnID", fnID, "err", err, "method", voteSetMsgHandlerMethodID,
			)
			return
		}

		err = currentVoteSet.AddVote(currentNonce, &FnIndividualExecutionResponse{
			Hash:            hash,
			OracleSignature: signature,
		}, currentValidators, ownValidatorIndex, f.privValidator)
		if err != nil {
			f.Logger.Error(
				"FnConsensusError: unable to add agree vote to current voteset, ignoring...",
				"err", err, "method", voteSetMsgHandlerMethodID,
			)
			return
		}

		didWeContribute = true
		hasOurVoteSetChanged = true
	}

	// If our voteset hasn't changed, no need to announce it, as we would have already annonunced it
	// the last time it changed. This could mean no new additions happened on our existing voteset,
	// and by logic other flags also will be false.
	if !hasOurVoteSetChanged {
		return
	}

	marshalledBytes, err := currentVoteSet.Marshal()
	if err != nil {
		f.Logger.Error(
			"FnConsensusReactor: Unable to marshal currentVoteSet",
			"fnID", fnID, "err", err, "method", voteSetMsgHandlerMethodID,
		)
		return
	}

	// If we didnt contribute to remote vote, no need to pass it to sender
	// If this is false, then we must not have achieved Maj23
	broadCastException := sender.ID()
	if !didWeContribute {
		f.broadcastMsgSync(FnVoteSetChannel, &broadCastException, marshalledBytes)
	} else {
		f.broadcastMsgSync(FnVoteSetChannel, nil, marshalledBytes)
	}
}

// Receive implements BaseReactor, it's called when msgBytes is received from a peer.
//
// NOTE reactor can't keep msgBytes around after Receive completes without copying.
//
// CONTRACT: msgBytes are not nil.
func (f *FnConsensusReactor) Receive(chID byte, sender p2p.Peer, msgBytes []byte) {
	switch chID {
	case FnVoteSetChannel:
		f.handleVoteSetChannelMessage(sender, msgBytes)
	case FnMajChannel:
		f.handleMaj23VoteSetChannel(sender, msgBytes)
	default:
		f.Logger.Error("FnConsensusReactor: Unknown channel: %v", chID)
	}
}
