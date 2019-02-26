package fnConsensus

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/state"
	"github.com/tendermint/tendermint/types"

	dbm "github.com/tendermint/tendermint/libs/db"

	"crypto/rand"
	"crypto/sha512"
)

const FnVoteSetChannel = byte(0x50)

const VoteSetIDSize = 32

const StartingNonce int64 = 1

// Max message size 1 MB
const maxMsgSize = 1000 * 1024

const ProgressIntervalInSeconds int64 = 91

const CommitRoutineExecutionBuffer = 1 * time.Second
const DefaultValidityPeriod = 71 * time.Second
const DefaultValidityPeriodForSync = 60 * time.Second

// Max context size 1 KB
const MaxContextSize = 1024

const MaxAllowedTimeDriftInFuture = 10 * time.Second

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

	commitRoutineQuitCh map[string]chan struct{}
}

func NewFnConsensusReactor(chainID string, privValidator types.PrivValidator, fnRegistry FnRegistry, db dbm.DB, tmStateDB dbm.DB) *FnConsensusReactor {
	reactor := &FnConsensusReactor{
		connectedPeers:      make(map[p2p.ID]p2p.Peer),
		db:                  db,
		chainID:             chainID,
		tmStateDB:           tmStateDB,
		fnRegistry:          fnRegistry,
		privValidator:       privValidator,
		commitRoutineQuitCh: make(map[string]chan struct{}),
	}

	reactor.BaseReactor = *p2p.NewBaseReactor("FnConsensusReactor", reactor)
	return reactor
}

func (f *FnConsensusReactor) String() string {
	return "FnConsensusReactor"
}

func (f *FnConsensusReactor) OnStart() error {
	reactorState, err := LoadReactorState(f.db)
	if err != nil {
		return err
	}

	f.stateMtx.Lock()
	defer f.stateMtx.Unlock()

	f.state = reactorState

	fnIDs := f.fnRegistry.GetAll()
	for _, fnID := range fnIDs {
		currentVoteState := f.state.CurrentVoteSets[fnID]
		if currentVoteState != nil {
			if currentVoteState.IsExpired(DefaultValidityPeriod) {
				delete(f.state.CurrentVoteSets, fnID)
			}
		}
	}

	if err := SaveReactorState(f.db, f.state, true); err != nil {
		return err
	}

	go f.progressRoutine()
	return nil
}

// GetChannels returns the list of channel descriptors.
func (f *FnConsensusReactor) GetChannels() []*p2p.ChannelDescriptor {
	// Priorities are deliberately set to low, to prevent interfering with core TM
	return []*p2p.ChannelDescriptor{
		{
			ID:                  FnVoteSetChannel,
			Priority:            25,
			SendQueueCapacity:   100,
			RecvMessageCapacity: maxMsgSize,
		},
	}
}

// AddPeer is called by the switch when a new peer is added.
func (f *FnConsensusReactor) AddPeer(peer p2p.Peer) {
	f.peerMapMtx.Lock()
	defer f.peerMapMtx.Unlock()
	f.connectedPeers[peer.ID()] = peer
}

// RemovePeer is called by the switch when the peer is stopped (due to error
// or other reason).
func (f *FnConsensusReactor) RemovePeer(peer p2p.Peer, reason interface{}) {
	f.peerMapMtx.Lock()
	defer f.peerMapMtx.Unlock()
	delete(f.connectedPeers, peer.ID())
}

func (f *FnConsensusReactor) areWeValidator(currentValidatorSet *types.ValidatorSet) (bool, int) {
	validatorIndex, _ := currentValidatorSet.GetByAddress(f.privValidator.GetPubKey().Address())
	return validatorIndex != -1, validatorIndex
}

func (f *FnConsensusReactor) generateVoteSetID() (string, error) {
	randomBytes := make([]byte, VoteSetIDSize)

	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(randomBytes), nil
}

func (f *FnConsensusReactor) calculateMessageHash(message []byte) ([]byte, error) {
	hash := sha512.New()
	_, err := hash.Write(message)
	if err != nil {
		return nil, err
	}
	return hash.Sum(nil), nil
}

func (f *FnConsensusReactor) progressRoutine() {

OUTER_LOOP:
	for {
		// Align to minutes, to make sure this routine runs at almost same time across all nodes
		// Not strictly required
		currentEpochTime := time.Now().Unix()
		timeToSleep := int64(ProgressIntervalInSeconds - currentEpochTime%ProgressIntervalInSeconds)
		timer := time.NewTimer(time.Duration(timeToSleep) * time.Second)

		select {
		case <-f.Quit():
			timer.Stop()
			break OUTER_LOOP
		case <-timer.C:
			var areWeAllowedToPropose bool

			currentState := state.LoadState(f.tmStateDB)
			areWeValidator, ownValidatorIndex := f.areWeValidator(currentState.Validators)

			proposer := currentState.Validators.GetProposer()
			if proposer == nil {
				f.Logger.Error("FnConsensusReactor: unable to get proposer from current validators")
				break
			}

			proposerIndex, _ := currentState.Validators.GetByAddress(proposer.Address)

			if areWeValidator && proposerIndex == ownValidatorIndex {
				areWeAllowedToPropose = true
			} else {
				areWeAllowedToPropose = false
			}

			f.stateMtx.Lock()

			fnIDs := f.fnRegistry.GetAll()
			sort.Strings(fnIDs)

			fnsEligibleForProposal := make([]string, 0, len(fnIDs))

			for _, fnID := range fnIDs {
				currentVoteState := f.state.CurrentVoteSets[fnID]
				if currentVoteState != nil {
					if currentVoteState.IsExpired(DefaultValidityPeriod) {
						f.state.PreviousTimedOutVoteSets[fnID] = f.state.CurrentVoteSets[fnID]
						delete(f.state.CurrentVoteSets, fnID)
						f.Logger.Info("FnConsensusReactor: archiving expired Fn execution", "FnID", fnID)
					} else {
						f.Logger.Info("FnConsensusReactor: unable to propose, previous execution is still pending", "FnID", fnID)
						continue
					}
				}
				fnsEligibleForProposal = append(fnsEligibleForProposal, fnID)
			}

			if err := SaveReactorState(f.db, f.state, true); err != nil {
				f.Logger.Error("FnConsensusReactor: unable to save reactor state")
				f.stateMtx.Unlock()
				break
			}

			f.stateMtx.Unlock()

			if !areWeAllowedToPropose {
				break
			}

			for _, fnID := range fnsEligibleForProposal {
				fn := f.fnRegistry.Get(fnID)
				f.propose(fnID, fn, currentState, ownValidatorIndex)
			}

		}
	}
}

func (f *FnConsensusReactor) propose(fnID string, fn Fn, currentState state.State, validatorIndex int) {
	shouldExecuteFn, ctx, err := fn.PrepareContext()
	if err != nil {
		f.Logger.Error("FnConsensusReactor: received error while executing fn.PrepareContext", "error", err)
		return
	}

	if len(ctx) > MaxContextSize {
		f.Logger.Error("FnConsensusReactor: context cannot be more than", "MaxContextSize", MaxContextSize)
		return
	}

	if !shouldExecuteFn {
		f.Logger.Info("FnConsensusReactor: PrepareContext indicated to not execute fn", "fnID", fnID)
		return
	}

	message, signature, err := fn.GetMessageAndSignature(safeCopyBytes(ctx))
	if err != nil {
		f.Logger.Error("FnConsensusReactor: received error while executing fn.GetMessageAndSignature", "fnID", fnID)
		return
	}

	hash, err := f.calculateMessageHash(message)
	if err != nil {
		f.Logger.Error("FnConsensusReactor: unable to calculate message hash", "fnID", fnID, "error", err)
		return
	}

	if err = fn.MapMessage(safeCopyBytes(ctx), safeCopyBytes(hash), safeCopyBytes(message)); err != nil {
		f.Logger.Error("FnConsensusReactor: received error while executing fn.MapMessage", "fnID", fnID, "error", err)
		return
	}

	executionRequest, err := NewFnExecutionRequest(fnID, f.fnRegistry)
	if err != nil {
		f.Logger.Error("FnConsensusReactor: unable to create Fn execution request as FnID is invalid", "fnID", fnID)
		return
	}

	executionResponse := NewFnExecutionResponse(&FnIndividualExecutionResponse{
		Error:           "",
		Hash:            hash,
		OracleSignature: signature,
		Status:          0,
	}, validatorIndex, currentState.Validators)

	votesetPayload := NewFnVotePayload(executionRequest, executionResponse)

	f.stateMtx.Lock()

	currentNonce, ok := f.state.CurrentNonces[fnID]
	if !ok {
		currentNonce = 1
	}

	newVoteSetID, err := f.generateVoteSetID()
	if err != nil {
		f.Logger.Error("FnConsensusReactor: unable to generate new vote set id")
		f.stateMtx.Unlock()
		return
	}

	voteSet, err := NewVoteSet(newVoteSetID, currentNonce, f.chainID, DefaultValidityPeriod, validatorIndex, ctx,
		votesetPayload, f.privValidator, currentState.Validators)
	if err != nil {
		f.Logger.Error("FnConsensusReactor: unable to create new voteset", "fnID", fnID, "error", err)
		f.stateMtx.Unlock()
		return
	}

	// It seems we are the only validator, so return the signature and close the case.
	if voteSet.IsMaj23Agree(currentState.Validators) {
		fn.SubmitMultiSignedMessage(safeCopyBytes(ctx),
			safeCopyBytes(voteSet.Payload.Response.Hash),
			safeCopyDoubleArray(voteSet.Payload.Response.OracleSignatures))
		f.stateMtx.Unlock()
		return
	}

	f.state.CurrentVoteSets[fnID] = voteSet
	quitCh := make(chan struct{})
	f.commitRoutineQuitCh[fnID] = quitCh
	go f.commitRoutine(fnID, time.Unix(voteSet.CreationTime, 0).Add(DefaultValidityPeriodForSync+CommitRoutineExecutionBuffer), quitCh)

	if err := SaveReactorState(f.db, f.state, true); err != nil {
		f.Logger.Error("FnConsensusReactor: unable to save state", "fnID", fnID, "error", err)
		f.stateMtx.Unlock()
		return
	}

	f.stateMtx.Unlock()

	marshalledBytes, err := voteSet.Marshal()
	if err != nil {
		f.Logger.Error(fmt.Sprintf("FnConsensusReactor: Unable to marshal currentVoteSet at FnID: %s", fnID))
		return
	}

	f.peerMapMtx.RLock()
	for _, peer := range f.connectedPeers {
		peer.Send(FnVoteSetChannel, marshalledBytes)
	}
	f.peerMapMtx.RUnlock()
}

func (f *FnConsensusReactor) handleCommit(fnIDToMonitor string) {
	fn := f.fnRegistry.Get(fnIDToMonitor)
	if fn == nil {
		f.Logger.Error("FnConsensusReactor: fn is nil while trying to access it in commit routine, Ignoring...")
		return
	}

	currentState := state.LoadState(f.tmStateDB)

	f.stateMtx.Lock()
	defer f.stateMtx.Unlock()

	areWeValidator, ownValidatorIndex := f.areWeValidator(currentState.Validators)

	currentVoteSet := f.state.CurrentVoteSets[fnIDToMonitor]
	currentNonce := f.state.CurrentNonces[fnIDToMonitor]

	if !currentVoteSet.IsValid(f.chainID, MaxContextSize, DefaultValidityPeriod, currentState.Validators, f.fnRegistry) {
		f.Logger.Error("Invalid VoteSet", "VoteSet", currentVoteSet)
		return
	}

	if !currentVoteSet.IsMaj23(currentState.Validators) {
		f.Logger.Info("No major 2/3 achived", "VoteSet", currentVoteSet)

		f.state.PreviousTimedOutVoteSets[fnIDToMonitor] = currentVoteSet
		delete(f.state.CurrentVoteSets, fnIDToMonitor)
	} else {
		if areWeValidator && currentVoteSet.IsMaj23Agree(currentState.Validators) {
			numberOfAgreeVotes := currentVoteSet.NumberOfAgreeVotes()

			agreeVoteIndex, err := currentVoteSet.GetAgreeVoteIndexForValidatorIndex(ownValidatorIndex)
			if err != nil {
				f.Logger.Error("FnConsensusReactor: unable to get agree vote index for validator", "validatorIndex", ownValidatorIndex)
				return
			}

			if (currentNonce % int64(numberOfAgreeVotes)) == int64(agreeVoteIndex) {
				fn.SubmitMultiSignedMessage(safeCopyBytes(currentVoteSet.ExecutionContext),
					safeCopyBytes(currentVoteSet.Payload.Response.Hash),
					safeCopyDoubleArray(currentVoteSet.Payload.Response.OracleSignatures))
			}
		}

		f.state.CurrentNonces[fnIDToMonitor]++
		f.state.PreviousMaj23VoteSets[fnIDToMonitor] = currentVoteSet
		delete(f.state.CurrentVoteSets, fnIDToMonitor)
	}

	if err := SaveReactorState(f.db, f.state, true); err != nil {
		f.Logger.Error("FnConsensusReactor: unable to save state", "fnID", fnIDToMonitor, "error", err)
		return
	}
}

func (f *FnConsensusReactor) commitRoutine(fnIDToMonitor string, monitoringTill time.Time, quitCh <-chan struct{}) {
	unlockDuration := time.Until(monitoringTill)
	timer := time.NewTimer(unlockDuration)

	select {
	case <-quitCh:
		break
	case <-timer.C:
		f.handleCommit(fnIDToMonitor)
		break
	}
}

func (f *FnConsensusReactor) compareVoteSets(remoteVoteSet *FnVoteSet, currentVoteSet *FnVoteSet, currentNonce int64, currentValidators *types.ValidatorSet) int {
	if currentVoteSet == nil {
		if currentNonce == remoteVoteSet.Nonce {
			return 1
		}

		if remoteVoteSet.IsMaj23(currentValidators) {
			return 1
		}

		return -1
	}

	// Perfect candidate to merge
	if currentVoteSet.ID == remoteVoteSet.ID {
		return 0
	}

	currentVoteSetMaj23Agree := currentVoteSet.IsMaj23Agree(currentValidators)
	currentVoteSetMaj23Disagree := currentVoteSet.IsMaj23Disagree(currentValidators)
	currentVoteSetMaj23 := currentVoteSetMaj23Agree || currentVoteSetMaj23Disagree

	remoteVoteSetMaj23Agree := remoteVoteSet.IsMaj23Agree(currentValidators)
	remoteVoteSetMaj23Disagree := remoteVoteSet.IsMaj23Disagree(currentValidators)
	remoteVoteSetMaj23 := remoteVoteSetMaj23Agree || remoteVoteSetMaj23Disagree

	if currentVoteSetMaj23 && !remoteVoteSetMaj23 {
		return -1
	} else if !currentVoteSetMaj23 && remoteVoteSetMaj23 {
		return 1
	} else if !currentVoteSetMaj23 && !remoteVoteSetMaj23 {
		return -1
	}

	if currentVoteSetMaj23Agree && !remoteVoteSetMaj23Agree {
		return -1
	} else if !currentVoteSetMaj23Agree && remoteVoteSetMaj23Agree {
		return 1
	} else if !currentVoteSetMaj23Agree && !remoteVoteSetMaj23Agree {
		return -1
	}

	currentNumberOfVotes := currentVoteSet.NumberOfVotes()
	currentNumberOfAgreeVotes := currentVoteSet.NumberOfAgreeVotes()

	remoteNumberOfVotes := remoteVoteSet.NumberOfVotes()
	remoteNumberOfAgreeVotes := remoteVoteSet.NumberOfAgreeVotes()

	if remoteNumberOfVotes < currentNumberOfVotes {
		return -1
	} else if remoteNumberOfVotes > currentNumberOfVotes {
		return 1
	}

	if remoteNumberOfAgreeVotes < currentNumberOfAgreeVotes {
		return -1
	} else if remoteNumberOfAgreeVotes > currentNumberOfAgreeVotes {
		return 1
	}

	if currentVoteSet.CreationTime > remoteVoteSet.CreationTime {
		return -1
	} else if currentVoteSet.CreationTime < remoteVoteSet.CreationTime {
		return 1
	}

	// If everything is same, we will trust current vote set
	return -1
}

func (f *FnConsensusReactor) handleVoteSetChannelMessage(sender p2p.Peer, msgBytes []byte) {
	currentState := state.LoadState(f.tmStateDB)
	areWeValidator, ownValidatorIndex := f.areWeValidator(currentState.Validators)

	f.stateMtx.Lock()
	defer f.stateMtx.Unlock()

	remoteVoteSet := &FnVoteSet{}
	if err := remoteVoteSet.Unmarshal(msgBytes); err != nil {
		f.Logger.Error("FnConsensusReactor: Invalid Data passed, ignoring...", "error", err)
		return
	}

	if !remoteVoteSet.IsValid(f.chainID, MaxContextSize, DefaultValidityPeriodForSync, currentState.Validators, f.fnRegistry) {
		f.Logger.Error("FnConsensusReactor: Invalid VoteSet specified, ignoring...")
		return
	}

	fnID := remoteVoteSet.GetFnID()
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
			f.Logger.Error("FnConsensusReactor: Already seen this nonce, ignoring", "currentNonce", currentNonce, "remoteNonce", remoteVoteSet.Nonce)
			return
		}
	}

	switch f.compareVoteSets(remoteVoteSet, currentVoteSet, currentNonce, currentState.Validators) {
	// Both vote set have same trustworthy ness, so merge
	case 0:
		if didWeContribute, err = f.state.CurrentVoteSets[fnID].Merge(currentState.Validators, remoteVoteSet); err != nil {
			f.Logger.Error("FnConsensusReactor: Unable to merge remote vote set into our own.", "error:", err)
			return
		}
		currentVoteSet = f.state.CurrentVoteSets[fnID]
		currentNonce = f.state.CurrentNonces[fnID]

		hasOurVoteSetChanged = didWeContribute
		break
	// Remote voteset is more trustworthy, so replace
	case 1:
		if currentVoteSet != nil {
			quitCh := f.commitRoutineQuitCh[fnID]
			close(quitCh)
		}

		f.state.CurrentVoteSets[fnID] = remoteVoteSet
		f.state.CurrentNonces[fnID] = remoteVoteSet.Nonce

		currentVoteSet = f.state.CurrentVoteSets[fnID]
		currentNonce = f.state.CurrentNonces[fnID]

		hasOurVoteSetChanged = true
		didWeContribute = false

		quitCh := make(chan struct{})
		f.commitRoutineQuitCh[fnID] = quitCh
		go f.commitRoutine(fnID, time.Unix(currentVoteSet.CreationTime, 0).Add(DefaultValidityPeriodForSync+CommitRoutineExecutionBuffer), quitCh)
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

		message, signature, err := fn.GetMessageAndSignature(safeCopyBytes(currentVoteSet.ExecutionContext))
		if err != nil {
			f.Logger.Error("FnConsensusReactor: fn.GetMessageAndSignature returned an error, ignoring..")
			return
		}

		hash, err := f.calculateMessageHash(message)
		if err != nil {
			f.Logger.Error("FnConsensusReactor: unable to calculate message hash", "fnID", fnID, "error", err)
			return
		}

		areWeAgreed := (bytes.Compare(currentVoteSet.GetMessageHash(), hash) == 0)

		if err = fn.MapMessage(safeCopyBytes(currentVoteSet.ExecutionContext), safeCopyBytes(hash), safeCopyBytes(message)); err != nil {
			f.Logger.Error("FnConsensusReactor: received error while executing fn.MapMessage", "fnID", fnID, "error", err)
			return
		}

		if areWeAgreed {
			err = currentVoteSet.AddVote(currentNonce, &FnIndividualExecutionResponse{
				Status:          0,
				Error:           "",
				Hash:            hash,
				OracleSignature: signature,
			}, currentState.Validators, ownValidatorIndex, VoteTypeAgree, f.privValidator)
			if err != nil {
				f.Logger.Error("FnConsensusError: unable to add agree vote to current voteset, ignoring...", "error", err)
				return
			}
		} else {
			err = currentVoteSet.AddVote(currentNonce, &FnIndividualExecutionResponse{
				Status:          0,
				Error:           "",
				Hash:            hash,
				OracleSignature: nil,
			}, currentState.Validators, ownValidatorIndex, VoteTypeDisAgree, f.privValidator)
			if err != nil {
				f.Logger.Error("FnConsensusError: unable to add disagree vote to current voteset, ignoring...", "error", err)
				return
			}
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
		f.Logger.Error(fmt.Sprintf("FnConsensusReactor: Unable to marshal currentVoteSet at FnID: %s", fnID))
		return
	}

	f.peerMapMtx.RLock()
	for peerID, peer := range f.connectedPeers {

		// If we didnt contribute to remote vote, no need to pass it to sender
		// If this is false, then we must not have achieved Maj23
		if !didWeContribute {
			if peerID == sender.ID() {
				continue
			}
		}

		// TODO: Handle timeout
		peer.Send(FnVoteSetChannel, marshalledBytes)
	}
	f.peerMapMtx.RUnlock()

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
	default:
		f.Logger.Error("FnConsensusReactor: Unknown channel: %v", chID)
	}
}
