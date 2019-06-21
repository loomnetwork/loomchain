package fnConsensus

import (
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/state"
	"github.com/tendermint/tendermint/types"

	dbm "github.com/tendermint/tendermint/libs/db"

	"crypto/sha512"
)

type SigningThreshold string

// MethodIDs for tracing purpose
const (
	voteMethodID              = "vote"
	commitMethodID            = "commit"
	maj23MsgHandlerMethodID   = "handleMaj23Msg"
	voteSetMsgHandlerMethodID = "handleVoteSetMsg"
)

const (
	// ChannelIDs need to be unique across all the reactors.
	// so to avoid conflict with other reactor's channel id and
	// Give TM some wiggle room when they add more channel, we are starting
	// channel ids from 0x50 for this reactor.
	FnVoteSetChannel = byte(0x50)
	FnMajChannel     = byte(0x51)

	// Max message size 2 MB
	MaxMsgSize = 2 * 1000 * 1024

	// Adding the Commit execution buffer to both ProgressInterval and ExpiresIn
	// so that 10 seconds interval
	// is maintained between sync expiration, overall expiration and new proposal

	// ProgressIntervalInSeconds denotes interval (synced across node) between two progress/propose
	ProposeIntervalInSeconds int64 = 10
	CommitIntervalInSeconds  int64 = 5

	// Delay between propogating votesets to make other peers up to date.
	VoteSetPropogationDelay = 1 * time.Second

	// Max context size 1 KB
	MaxContextSize = 1024

	ProgressLoopStartDelay = 2 * time.Second

	Maj23SigningThreshold SigningThreshold = "Maj23"
	AllSigningThreshold   SigningThreshold = "All"

	ProposalInfoSigningThreshold = Maj23SigningThreshold
)

var ErrInvalidReactorConfiguration = errors.New("invalid reactor configuration")

type FnConsensusReactor struct {
	p2p.BaseReactor

	connectedPeers map[p2p.ID]p2p.Peer
	state          *ReactorState
	db             dbm.DB
	tmStateDB      dbm.DB
	chainID        string

	fnRegistry FnRegistry

	privValidator types.PrivValidator

	peerMapMtx sync.RWMutex

	stateMtx sync.Mutex

	staticValidators *types.ValidatorSet

	cfg *ReactorConfig
}

func NewFnConsensusReactor(chainID string, privValidator types.PrivValidator, fnRegistry FnRegistry, db dbm.DB, tmStateDB dbm.DB, parsableConfig *ReactorConfigParsable) (*FnConsensusReactor, error) {
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

func (f *FnConsensusReactor) safeSubmitMultiSignedMessage(fn Fn, ctx []byte, message []byte, signatures [][]byte) {
	defer func() {
		err := recover()
		if err != nil {
			f.Logger.Error("panicked while invoking SubmitMultiSignedMessage", "error", err)
		}
	}()
	fn.SubmitMultiSignedMessage(ctx, message, signatures)
}

func (f *FnConsensusReactor) safeGetMessageAndSignature(fn Fn, ctx []byte) ([]byte, []byte, error) {
	defer func() {
		err := recover()
		if err != nil {
			f.Logger.Error("panicked while invoking GetMessageAndSignature", "error", err)
		}
	}()
	return fn.GetMessageAndSignature(nil)
}

func (f *FnConsensusReactor) safeMapMessage(fn Fn, ctx []byte, hash []byte, message []byte) error {
	defer func() {
		err := recover()
		if err != nil {
			f.Logger.Error("panicked while invoking MapMessage", "error", err)
		}
	}()
	return fn.MapMessage(ctx, hash, message)
}

func (f *FnConsensusReactor) String() string {
	return "FnConsensusReactor"
}

func (f *FnConsensusReactor) OnStart() error {
	reactorState, err := LoadReactorState(f.db)
	if err != nil {
		return err
	}

	f.state = reactorState

	go f.initRoutine()

	return nil
}

// GetChannels returns the list of channel descriptors.
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

// AddPeer is called by the switch when a new peer is added.
func (f *FnConsensusReactor) AddPeer(peer p2p.Peer) {
	f.peerMapMtx.Lock()
	f.connectedPeers[peer.ID()] = peer
	f.peerMapMtx.Unlock()
}

// RemovePeer is called by the switch when the peer is stopped (due to error
// or other reason).
func (f *FnConsensusReactor) RemovePeer(peer p2p.Peer, reason interface{}) {
	f.peerMapMtx.Lock()
	defer f.peerMapMtx.Unlock()
	delete(f.connectedPeers, peer.ID())
}

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

func (f *FnConsensusReactor) calculateMessageHash(message []byte) ([]byte, error) {
	hash := sha512.New()
	_, err := hash.Write(message)
	if err != nil {
		return nil, err
	}
	return hash.Sum(nil), nil
}

func (f *FnConsensusReactor) calculateSleepTimeForCommit(areWeValidator bool) time.Duration {
	currentEpochTime := time.Now().Unix()
	baseTimeToSleep := CommitIntervalInSeconds - currentEpochTime%CommitIntervalInSeconds

	const maxBoundForVariableComponent = 2 * time.Second
	const baseCommitDelay = 100 * time.Millisecond

	if !areWeValidator {
		return (time.Duration(baseTimeToSleep) * time.Second) + baseCommitDelay
	}

	return (time.Duration(baseTimeToSleep) * time.Second) +
		time.Duration(rand.Int63n(int64(maxBoundForVariableComponent))) +
		baseCommitDelay
}

func (f *FnConsensusReactor) calculateSleepTimeForPropose(areWeValidator bool) time.Duration {
	currentEpochTime := time.Now().Unix()
	baseTimeToSleep := ProposeIntervalInSeconds - currentEpochTime%ProposeIntervalInSeconds

	const baseProposalDelay = 500 * time.Millisecond
	const maxBoundForVariableComponent = 2 * time.Second

	if !areWeValidator {
		return (time.Duration(baseTimeToSleep) * time.Second) + baseProposalDelay
	}

	return (time.Duration(baseTimeToSleep) * time.Second) +
		time.Duration(rand.Int63n(int64(maxBoundForVariableComponent))) +
		baseProposalDelay
}

func (f *FnConsensusReactor) initValidatorSet(tmState state.State) error {
	if len(f.cfg.OverrideValidators) == 0 {
		f.Logger.Info("FnConsensusReactor: using DPoS validators for consensus")
		return nil
	}

	validatorArray := make([]*types.Validator, 0, len(f.cfg.OverrideValidators))

	for _, overrideValidator := range f.cfg.OverrideValidators {
		// tmState.Validators is the tendermint address, not the loom address.
		validatorIndex, validator := tmState.Validators.GetByAddress(overrideValidator.Address)
		if validatorIndex == -1 {
			return fmt.Errorf("validator specified in override config, doesnt exist in TM validator set")
		}
		f.Logger.Info("FnConsensusReactor: adding static validator to set", "validator", validator.String())
		validatorArray = append(validatorArray, validator.Copy())
	}

	f.staticValidators = types.NewValidatorSet(validatorArray)

	f.Logger.Info("FnConsensusReactor: using static validators for consensus", "validatorSetHash", f.staticValidators.Hash())

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
		time.Sleep(ProgressLoopStartDelay)
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
	currentValidators := f.getValidatorSet()

	// Initializing these vars with sane value to calculate initial time
	areWeValidator, _ := f.areWeValidator(currentValidators)

OUTER_LOOP:
	for {
		commitSleepTime := f.calculateSleepTimeForCommit(areWeValidator)
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
	currentValidators := f.getValidatorSet()

	// Initializing these vars with sane value to calculate initial time
	areWeValidator, ownValidatorIndex := f.areWeValidator(currentValidators)

OUTER_LOOP:
	for {
		// Align to minutes, to make sure this routine runs at almost same time across all nodes
		// Not strictly required
		// state and other variables will be same as the one initialized in second case statement
		proposeSleepTime := f.calculateSleepTimeForPropose(areWeValidator)
		proposeTimer := time.NewTimer(proposeSleepTime)

		select {
		case <-f.Quit():
			proposeTimer.Stop()
			break OUTER_LOOP
		case <-proposeTimer.C:
			currentValidators := f.getValidatorSet()
			areWeValidator, ownValidatorIndex = f.areWeValidator(currentValidators)

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

func (f *FnConsensusReactor) vote(fnID string, fn Fn, currentValidators *types.ValidatorSet, validatorIndex int) {

	message, signature, err := f.safeGetMessageAndSignature(fn, nil)
	if err != nil {
		f.Logger.Error("FnConsensusReactor: received error while executing fn.GetMessageAndSignature", "fnID", fnID, "error", err, "method", voteMethodID)
		return
	}

	hash, err := f.calculateMessageHash(message)
	if err != nil {
		f.Logger.Error("FnConsensusReactor: unable to calculate message hash", "fnID", fnID, "error", err, "method", voteMethodID)
		return
	}

	if err = f.safeMapMessage(fn, nil, safeCopyBytes(hash), safeCopyBytes(message)); err != nil {
		f.Logger.Error("FnConsensusReactor: received error while executing fn.MapMessage", "fnID", fnID, "error", err, "method", voteMethodID)
		return
	}

	executionRequest, err := NewFnExecutionRequest(fnID, f.fnRegistry)
	if err != nil {
		f.Logger.Error("FnConsensusReactor: unable to create Fn execution request as FnID is invalid", "fnID", fnID, "error", err, "method", voteMethodID)
		return
	}

	executionResponse := NewFnExecutionResponse(&FnIndividualExecutionResponse{
		Hash:            hash,
		OracleSignature: signature,
	}, validatorIndex, currentValidators)

	votesetPayload := NewFnVotePayload(executionRequest, executionResponse)

	f.stateMtx.Lock()
	defer f.stateMtx.Unlock()

	currentNonce, ok := f.state.CurrentNonces[fnID]
	if !ok {
		currentNonce = 1
	}

	voteSet, err := NewVoteSet(currentNonce, f.chainID, validatorIndex,
		votesetPayload, f.privValidator, currentValidators)
	if err != nil {
		f.Logger.Error("FnConsensusReactor: unable to create new voteset", "fnID", fnID, "error", err, "method", voteMethodID)
		return
	}

	// Have we achieved Maj23 already?
	aggregateExecutionResponse := voteSet.MajResponse(f.cfg.FnVoteSigningThreshold, currentValidators)
	if aggregateExecutionResponse != nil {
		f.safeSubmitMultiSignedMessage(fn, nil,
			safeCopyBytes(aggregateExecutionResponse.Hash),
			safeCopyDoubleArray(aggregateExecutionResponse.OracleSignatures))
		return
	}

	f.state.CurrentVoteSets[fnID] = voteSet

	if err := SaveReactorState(f.db, f.state, true); err != nil {
		f.Logger.Error("FnConsensusReactor: unable to save state", "fnID", fnID, "error", err, "method", voteMethodID)
		return
	}

	marshalledBytes, err := voteSet.Marshal()
	if err != nil {
		f.Logger.Error(fmt.Sprintf("FnConsensusReactor: Unable to marshal currentVoteSet at FnID: %s", fnID), "error", err, "method", voteMethodID)
		return
	}

	f.broadcastMsgSync(FnVoteSetChannel, nil, marshalledBytes)
}

func (f *FnConsensusReactor) commit(fnID string) {

	fn := f.fnRegistry.Get(fnID)
	if fn == nil {
		f.Logger.Error("FnConsensusReactor: fn is nil while trying to access it in commit routine, Ignoring...", "method", commitMethodID)
		return
	}

	currentValidators := f.getValidatorSet()
	areWeValidator, ownValidatorIndex := f.areWeValidator(currentValidators)

	f.stateMtx.Lock()
	defer f.stateMtx.Unlock()

	currentVoteSet := f.state.CurrentVoteSets[fnID]
	currentNonce := f.state.CurrentNonces[fnID]

	if err := currentVoteSet.IsValid(f.chainID, currentValidators, f.fnRegistry); err != nil {
		f.Logger.Error("FnConsensusReactor: Invalid VoteSet found", "VoteSet", currentVoteSet, "error", err, "method", commitMethodID)
		delete(f.state.CurrentVoteSets, fnID)
		if err := SaveReactorState(f.db, f.state, true); err != nil {
			f.Logger.Error("FnConsensusReactor: unable to save state", "fnID", fnID, "error", err, "method", commitMethodID)
			return
		}
		return
	}

	if !currentVoteSet.HasConverged(f.cfg.FnVoteSigningThreshold, currentValidators) {
		f.Logger.Info("No consensus achieved", "fnID", fnID, "VoteSet", currentVoteSet, "Payload",
			currentVoteSet.Payload, "Response", currentVoteSet.Payload.Response, "method", commitMethodID)

		previousConvergedVoteSet := f.state.PreviousMajVoteSets[fnID]
		if previousConvergedVoteSet != nil {
			marshalledBytesOfPreviousVoteSet, err := previousConvergedVoteSet.Marshal()
			if err != nil {
				f.Logger.Error("unable to marshal PreviousMajVoteSet", "error", err, "fnID", fnID, "method", commitMethodID)
				return
			}

			marshalledBytesOfCurrentVoteSet, err := currentVoteSet.Marshal()
			if err != nil {
				f.Logger.Error("unable to marshal Current Vote set", "error", err, "fnID", fnID, "method", commitMethodID)
				return
			}

			// Propagate your last Maj23, to remedy any issue
			f.broadcastMsgSync(FnMajChannel, nil, marshalledBytesOfPreviousVoteSet)

			time.Sleep(VoteSetPropogationDelay)

			// Propagate your current voteSet, to get newly joined node to sign it
			f.broadcastMsgSync(FnVoteSetChannel, nil, marshalledBytesOfCurrentVoteSet)
		}
	} else {
		if areWeValidator {
			majExecutionResponse := currentVoteSet.MajResponse(f.cfg.FnVoteSigningThreshold, currentValidators)
			if majExecutionResponse != nil {
				f.Logger.Info("Maj-consensus achieved", "fnID", fnID, "VoteSet", currentVoteSet, "Payload", currentVoteSet.Payload, "Response", currentVoteSet.Payload.Response, "method", commitMethodID)
				numberOfAgreeVotes := majExecutionResponse.NumberOfAgreeVotes()
				agreeVoteIndex := majExecutionResponse.AgreeIndex(ownValidatorIndex)
				if agreeVoteIndex != -1 && (currentNonce%int64(numberOfAgreeVotes)) == int64(agreeVoteIndex) {
					f.Logger.Info("FnConsensusReactor: Submitting Multisigned message")
					f.safeSubmitMultiSignedMessage(fn, nil, safeCopyBytes(majExecutionResponse.Hash),
						safeCopyDoubleArray(majExecutionResponse.OracleSignatures))
				}
			}
		}

		f.state.CurrentNonces[fnID]++
		f.state.PreviousValidatorSet = currentValidators
		f.state.PreviousMajVoteSets[fnID] = currentVoteSet
		delete(f.state.CurrentVoteSets, fnID)
	}

	if err := SaveReactorState(f.db, f.state, true); err != nil {
		f.Logger.Error("FnConsensusReactor: unable to save state", "fnID", fnID, "error", err, "method", commitMethodID)
		return
	}
}

func (f *FnConsensusReactor) compareFnVoteSets(
	remoteVoteSet *FnVoteSet,
	currentVoteSet *FnVoteSet,
	currentNonce int64,
	currentValidators *types.ValidatorSet) int {
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
		f.Logger.Error("FnConsensusReactor: Invalid Data passed, ignoring...", "error", err, "method", maj23MsgHandlerMethodID)
		return
	}

	// We might have recently changed validator set, so Maybe this voteset is valid with previousValidatorSet and not current
	// We dont need to validate the proposer, as it might be outdated in our case
	if err := remoteMajVoteSet.IsValid(f.chainID, currentValidatorSet, f.fnRegistry); err != nil {
		if previousValidatorSet == nil {
			f.Logger.Error("FnConsensusReactor: Invalid VoteSet specified, ignoring...", "error", err, "method", maj23MsgHandlerMethodID)
			return
		}
		if err := remoteMajVoteSet.IsValid(f.chainID, previousValidatorSet, f.fnRegistry); err != nil {
			f.Logger.Error("FnConsensusReactor: Invalid VoteSet specified, ignoring...", "error", err, "method", maj23MsgHandlerMethodID)
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

		// If we have found maj23 voteset with Nonce equal or greater than our current nonce,
		// our current vote set is clearly outdated, and should be removed.
		delete(f.state.CurrentVoteSets, remoteFnID)
	}

	if err := SaveReactorState(f.db, f.state, true); err != nil {
		f.Logger.Error("FnConsensusReactor: unable to save reactor state", "error", err, "method", maj23MsgHandlerMethodID)
		return
	}

	if !needToBroadcast {
		return
	}

	marshalledBytes, err := previousMaj23VoteSet.Marshal()
	if err != nil {
		f.Logger.Error("FnConsensusReactor: unable to marshal bytes", "error", err, "method", maj23MsgHandlerMethodID)
		return
	}

	f.broadcastMsgSync(FnMajChannel, nil, marshalledBytes)
}

func (f *FnConsensusReactor) handleVoteSetChannelMessage(sender p2p.Peer, msgBytes []byte) {
	currentValidators := f.getValidatorSet()
	areWeValidator, ownValidatorIndex := f.areWeValidator(currentValidators)

	f.stateMtx.Lock()
	defer f.stateMtx.Unlock()

	remoteVoteSet := &FnVoteSet{}
	if err := remoteVoteSet.Unmarshal(msgBytes); err != nil {
		f.Logger.Error("FnConsensusReactor: Invalid Data passed, ignoring...", "error", err, "method", voteSetMsgHandlerMethodID)
		return
	}

	fnID := remoteVoteSet.GetFnID()

	if err := remoteVoteSet.IsValid(f.chainID, currentValidators, f.fnRegistry); err != nil {
		f.Logger.Error("FnConsensusReactor: Invalid VoteSet specified, ignoring...", "error", err, "method", voteSetMsgHandlerMethodID)
		return
	}

	var didWeContribute, hasOurVoteSetChanged bool
	var err error

	currentNonce, ok := f.state.CurrentNonces[remoteVoteSet.GetFnID()]
	if !ok {
		currentNonce = 1
		f.state.CurrentNonces[remoteVoteSet.GetFnID()] = currentNonce
	}
	currentVoteSet := f.state.CurrentVoteSets[remoteVoteSet.GetFnID()]

	if currentNonce != remoteVoteSet.Nonce {
		if currentNonce > remoteVoteSet.Nonce {
			f.Logger.Info(
				"FnConsensusReactor: Already seen this nonce, ignoring",
				"currentNonce", currentNonce,
				"remoteNonce", remoteVoteSet.Nonce,
			)
			return
		}
	}

	switch f.compareFnVoteSets(remoteVoteSet, currentVoteSet, currentNonce, currentValidators) {
	// Both vote set have same trustworthy ness, so merge
	case 0:
		if didWeContribute, err = f.state.CurrentVoteSets[fnID].Merge(currentValidators, remoteVoteSet); err != nil {
			f.Logger.Error("FnConsensusReactor: Unable to merge remote vote set into our own.", "error", err, "method", voteSetMsgHandlerMethodID)
			return
		}
		currentVoteSet = f.state.CurrentVoteSets[fnID]
		currentNonce = f.state.CurrentNonces[fnID]

		hasOurVoteSetChanged = didWeContribute
		break
	// Remote voteset is more trustworthy, so replace
	case 1:
		f.state.CurrentVoteSets[fnID] = remoteVoteSet
		f.state.CurrentNonces[fnID] = remoteVoteSet.Nonce

		currentVoteSet = f.state.CurrentVoteSets[fnID]
		currentNonce = f.state.CurrentNonces[fnID]

		hasOurVoteSetChanged = true
		didWeContribute = false
		break
	// Current voteset is more trustworthy
	case -1:
		if currentVoteSet == nil {
			return
		}
		break
	}

	if areWeValidator && !currentVoteSet.HaveWeAlreadySigned(ownValidatorIndex) {
		fn := f.fnRegistry.Get(fnID)

		message, signature, err := f.safeGetMessageAndSignature(fn, nil)
		if err != nil {
			f.Logger.Error("FnConsensusReactor: received error while executing fn.GetMessageAndSignature", "fnID", fnID, "error", err, "method", voteSetMsgHandlerMethodID)
			return
		}

		hash, err := f.calculateMessageHash(message)
		if err != nil {
			f.Logger.Error("FnConsensusReactor: unable to calculate message hash", "fnID", fnID, "error", err, "method", voteSetMsgHandlerMethodID)
			return
		}

		if err = f.safeMapMessage(fn, nil, safeCopyBytes(hash), safeCopyBytes(message)); err != nil {
			f.Logger.Error("FnConsensusReactor: received error while executing fn.MapMessage", "fnID", fnID, "error", err, "method", voteSetMsgHandlerMethodID)
			return
		}

		err = currentVoteSet.AddVote(currentNonce, &FnIndividualExecutionResponse{
			Hash:            hash,
			OracleSignature: signature,
		}, currentValidators, ownValidatorIndex, f.privValidator)
		if err != nil {
			f.Logger.Error("FnConsensusError: unable to add agree vote to current voteset, ignoring...", "error", err, "method", voteSetMsgHandlerMethodID)
			return
		}

		didWeContribute = true
		hasOurVoteSetChanged = true
	}

	// If our vote havent't changed, no need to annonce it, as
	// we would have already annonunced it last time it changed
	// This could mean no new additions happened on our existing voteset, and
	// by logic other flags also will be false
	if !hasOurVoteSetChanged {
		return
	}

	marshalledBytes, err := currentVoteSet.Marshal()
	if err != nil {
		f.Logger.Error(fmt.Sprintf("FnConsensusReactor: Unable to marshal currentVoteSet at FnID: %s", fnID), "error", err, "method", voteSetMsgHandlerMethodID)
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

// Receive is called when msgBytes is received from peer.
//
// NOTE reactor can not keep msgBytes around after Receive completes without
// copying.
//
// CONTRACT: msgBytes are not nil.
func (f *FnConsensusReactor) Receive(chID byte, sender p2p.Peer, msgBytes []byte) {

	switch chID {
	case FnVoteSetChannel:
		f.handleVoteSetChannelMessage(sender, msgBytes)
		break
	case FnMajChannel:
		f.handleMaj23VoteSetChannel(sender, msgBytes)
		break
	default:
		f.Logger.Error("FnConsensusReactor: Unknown channel: %v", chID)
	}
}
