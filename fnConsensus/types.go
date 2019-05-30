package fnConsensus

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/pkg/errors"

	"github.com/tendermint/tendermint/crypto"
	cmn "github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/types"
)

var ErrFnVoteInvalidValidatorAddress = errors.New("invalid validator address for FnVote")
var ErrFnVoteInvalidSignature = errors.New("invalid validator signature")
var ErrFnVoteInvalidProposerSignature = errors.New("invalid proposer signature")
var ErrFnVoteNotPresent = errors.New("Fn vote is not present for validator")
var ErrFnVoteAlreadyCast = errors.New("Fn vote is already cast")
var ErrFnResponseSignatureAlreadyPresent = errors.New("Fn Response signature is already present")

var ErrFnVoteMergeDiffPayload = errors.New("merging is not allowed, as fn votes have different payload")
var ErrPetitionVoteMergeDiffPayload = errors.New("merging is not allowed, as petition votes have different payload")

type fnIDToNonce struct {
	Nonce int64
	FnID  string
}

type FnIndividualExecutionResponse struct {
	Hash            []byte
	OracleSignature []byte
}

func (f *FnIndividualExecutionResponse) Marshal() ([]byte, error) {
	return cdc.MarshalBinaryLengthPrefixed(f)
}

type reactorStateMarshallable struct {
	CurrentVoteSets          []*FnVoteSet
	CurrentNonces            []*fnIDToNonce
	PreviousTimedOutVoteSets []*FnVoteSet
	PreviousMajVoteSets      []*FnVoteSet
	PreviousValidatorSet     *types.ValidatorSet
}

type ReactorState struct {
	CurrentVoteSets          map[string]*FnVoteSet
	CurrentNonces            map[string]int64
	PreviousTimedOutVoteSets map[string]*FnVoteSet
	PreviousMajVoteSets      map[string]*FnVoteSet
	PreviousValidatorSet     *types.ValidatorSet
}

func (p *ReactorState) Marshal() ([]byte, error) {
	reactorStateMarshallable := &reactorStateMarshallable{
		CurrentVoteSets:          make([]*FnVoteSet, len(p.CurrentVoteSets)),
		CurrentNonces:            make([]*fnIDToNonce, len(p.CurrentNonces)),
		PreviousTimedOutVoteSets: make([]*FnVoteSet, len(p.PreviousTimedOutVoteSets)),
		PreviousMajVoteSets:      make([]*FnVoteSet, len(p.PreviousMajVoteSets)),
		PreviousValidatorSet:     p.PreviousValidatorSet,
	}

	i := 0
	for _, voteSet := range p.CurrentVoteSets {
		reactorStateMarshallable.CurrentVoteSets[i] = voteSet
		i++
	}

	i = 0
	for fnID, nonce := range p.CurrentNonces {
		reactorStateMarshallable.CurrentNonces[i] = &fnIDToNonce{
			FnID:  fnID,
			Nonce: nonce,
		}
		i++
	}

	i = 0
	for _, timedOutVoteSet := range p.PreviousTimedOutVoteSets {
		reactorStateMarshallable.PreviousTimedOutVoteSets[i] = timedOutVoteSet
		i++
	}

	i = 0
	for _, maj23VoteSet := range p.PreviousMajVoteSets {
		reactorStateMarshallable.PreviousMajVoteSets[i] = maj23VoteSet
		i++
	}

	return cdc.MarshalBinaryLengthPrefixed(reactorStateMarshallable)
}

func (p *ReactorState) Unmarshal(bz []byte) error {
	reactorStateMarshallable := &reactorStateMarshallable{}
	if err := cdc.UnmarshalBinaryLengthPrefixed(bz, reactorStateMarshallable); err != nil {
		return err
	}

	p.CurrentVoteSets = make(map[string]*FnVoteSet)
	p.CurrentNonces = make(map[string]int64)
	p.PreviousTimedOutVoteSets = make(map[string]*FnVoteSet)
	p.PreviousMajVoteSets = make(map[string]*FnVoteSet)
	p.PreviousValidatorSet = reactorStateMarshallable.PreviousValidatorSet

	for _, voteSet := range reactorStateMarshallable.CurrentVoteSets {
		p.CurrentVoteSets[voteSet.Payload.Request.FnID] = voteSet
	}

	for _, fnIDToNonce := range reactorStateMarshallable.CurrentNonces {
		p.CurrentNonces[fnIDToNonce.FnID] = fnIDToNonce.Nonce
	}

	for _, timeOutVoteSet := range reactorStateMarshallable.PreviousTimedOutVoteSets {
		p.PreviousTimedOutVoteSets[timeOutVoteSet.Payload.Request.FnID] = timeOutVoteSet
	}

	for _, maj23VoteSet := range reactorStateMarshallable.PreviousMajVoteSets {
		p.PreviousMajVoteSets[maj23VoteSet.Payload.Request.FnID] = maj23VoteSet
	}

	return nil
}

func NewReactorState() *ReactorState {
	return &ReactorState{
		CurrentVoteSets:          make(map[string]*FnVoteSet),
		CurrentNonces:            make(map[string]int64),
		PreviousTimedOutVoteSets: make(map[string]*FnVoteSet),
		PreviousMajVoteSets:      make(map[string]*FnVoteSet),
	}
}

type FnExecutionRequest struct {
	FnID string
}

func (f *FnExecutionRequest) Marshal() ([]byte, error) {
	return cdc.MarshalBinaryLengthPrefixed(f)
}

func (f *FnExecutionRequest) Unmarshal(bz []byte) error {
	return cdc.UnmarshalBinaryLengthPrefixed(bz, f)
}

func (f *FnExecutionRequest) CannonicalCompare(remoteRequest *FnExecutionRequest) bool {
	return f.FnID == remoteRequest.FnID
}

func (f *FnExecutionRequest) Compare(remoteRequest *FnExecutionRequest) bool {
	return f.CannonicalCompare(remoteRequest)
}

func (f *FnExecutionRequest) SignBytes() ([]byte, error) {
	return f.Marshal()
}

func NewFnExecutionRequest(fnID string, registry FnRegistry) (*FnExecutionRequest, error) {
	if registry.Get(fnID) == nil {
		return nil, fmt.Errorf("fnConsensusReactor: unable to create FnExecutionRequest as id is invalid")
	}

	return &FnExecutionRequest{
		FnID: fnID,
	}, nil
}

type FnAggregateExecutionResponse struct {
	Hash              []byte
	SignatureBitArray *cmn.BitArray
	OracleSignatures  [][]byte
}

func (f *FnAggregateExecutionResponse) AgreeIndex(ownValidatorIndex int) int {
	agreeVoteIndex := -1

	if !f.SignatureBitArray.GetIndex(ownValidatorIndex) {
		return agreeVoteIndex
	}

	for i := 0; i <= ownValidatorIndex; i++ {
		if f.SignatureBitArray.GetIndex(i) {
			agreeVoteIndex++
		}
	}

	return agreeVoteIndex
}

func (f *FnAggregateExecutionResponse) NumberOfAgreeVotes() int {
	agreeVoteIndex := 0
	for i := 0; i < f.SignatureBitArray.Size(); i++ {
		if f.SignatureBitArray.GetIndex(i) {
			agreeVoteIndex++
		}
	}
	return agreeVoteIndex
}

type FnExecutionResponse struct {
	Hashes            [][]byte
	SignatureBitArray *cmn.BitArray
	OracleSignatures  [][]byte
}

func (f *FnExecutionResponse) Marshal() ([]byte, error) {
	return cdc.MarshalBinaryLengthPrefixed(f)
}

func (f *FnExecutionResponse) Unmarshal(bz []byte) error {
	return cdc.UnmarshalBinaryLengthPrefixed(bz, f)
}

func (f *FnExecutionResponse) Merge(anotherExecutionResponse *FnExecutionResponse) (bool, error) {
	if anotherExecutionResponse == nil {
		return false, fmt.Errorf("cant merge as another execution response is nil")
	}

	if !f.CannonicalCompare(anotherExecutionResponse) {
		return false, fmt.Errorf("cant merge as another execution response is different")
	}

	hasResponseChanged := false

	for i := 0; i < len(f.OracleSignatures); i++ {
		if f.SignatureBitArray.GetIndex(i) || !anotherExecutionResponse.SignatureBitArray.GetIndex(i) {
			continue
		}

		hasResponseChanged = true

		f.OracleSignatures[i] = anotherExecutionResponse.OracleSignatures[i]
		f.Hashes[i] = anotherExecutionResponse.Hashes[i]

		f.SignatureBitArray.SetIndex(i, true)
	}

	return hasResponseChanged, nil
}

func (f *FnExecutionResponse) IsValid(currentValidatorSet *types.ValidatorSet) error {
	if f.Hashes == nil {
		return fmt.Errorf("executionResponse's hashes field cant be nil")
	}

	if f.SignatureBitArray == nil {
		return fmt.Errorf("executionResponse's SignatureBitArray cant be nil")
	}

	if f.OracleSignatures == nil {
		return fmt.Errorf("executionResponse's OracleSignatures field cant be nil")
	}

	if currentValidatorSet.Size() != len(f.OracleSignatures) {
		return fmt.Errorf("executionResponse's oracle signature's length does not match current validator set's length")
	}

	if currentValidatorSet.Size() != len(f.Hashes) {
		return fmt.Errorf("executionResponse's hashes' length does not match current validator set's length")
	}

	if currentValidatorSet.Size() != f.SignatureBitArray.Size() {
		return fmt.Errorf("executionResponse's signature bit array's size does not mach current validator set's length")
	}

	for i := 0; i < currentValidatorSet.Size(); i++ {
		oracleSignatureBytesPresent := f.OracleSignatures[i] != nil
		oracleSignatureFlagPresent := f.SignatureBitArray.GetIndex(i)
		hashPresent := f.Hashes[i] != nil

		if oracleSignatureBytesPresent != oracleSignatureFlagPresent {
			return fmt.Errorf("mismatch between oracle signature array and signature flag")
		}

		if oracleSignatureBytesPresent != hashPresent {
			return fmt.Errorf("mismatch between oracle signature array and hashes array")
		}
	}

	return nil
}

func (f *FnExecutionResponse) CannonicalCompare(remoteResponse *FnExecutionResponse) bool {
	if len(f.Hashes) != len(remoteResponse.Hashes) {
		return false
	}

	if f.SignatureBitArray.Size() != remoteResponse.SignatureBitArray.Size() {
		return false
	}

	if len(f.OracleSignatures) != len(remoteResponse.OracleSignatures) {
		return false
	}

	return true
}

func (f *FnExecutionResponse) SignBytes(validatorIndex int) ([]byte, error) {
	individualResponse := &FnIndividualExecutionResponse{
		Hash:            f.Hashes[validatorIndex],
		OracleSignature: f.OracleSignatures[validatorIndex],
	}

	return individualResponse.Marshal()
}

func (f *FnExecutionResponse) Compare(remoteResponse *FnExecutionResponse) bool {
	if !f.CannonicalCompare(remoteResponse) {
		return false
	}

	for i := 0; i < len(f.OracleSignatures); i++ {
		if f.SignatureBitArray.GetIndex(i) != remoteResponse.SignatureBitArray.GetIndex(i) {
			return false
		}
		if !bytes.Equal(f.OracleSignatures[i], remoteResponse.OracleSignatures[i]) {
			return false
		}
	}

	for i := 0; i < len(f.Hashes); i++ {
		if !bytes.Equal(f.Hashes[i], remoteResponse.Hashes[i]) {
			return false
		}
	}

	return true
}

func (f *FnExecutionResponse) AddSignature(
	individualResponse *FnIndividualExecutionResponse,
	validatorIndex int) error {
	if f.SignatureBitArray.GetIndex(validatorIndex) {
		return ErrFnResponseSignatureAlreadyPresent
	}

	f.OracleSignatures[validatorIndex] = individualResponse.OracleSignature
	f.Hashes[validatorIndex] = individualResponse.Hash

	f.SignatureBitArray.SetIndex(validatorIndex, true)
	return nil
}

func (f *FnExecutionResponse) ToMajResponse(
	signingThreshold SigningThreshold,
	currentValidatorSet *types.ValidatorSet) *FnAggregateExecutionResponse {
	hashMap := make(map[string]int64)
	var highestVotedHash []byte
	var highestVotingPowerObserved int64 = -1

	agreeVotesBitArray := cmn.NewBitArray(currentValidatorSet.Size())
	agreeOracleSignatures := make([][]byte, currentValidatorSet.Size())

	for i := 0; i < len(f.Hashes); i++ {
		if f.Hashes[i] == nil {
			continue
		}

		hashKey := hex.EncodeToString(f.Hashes[i])

		_, val := currentValidatorSet.GetByIndex(i)

		hashMap[hashKey] += val.VotingPower

		if highestVotingPowerObserved < hashMap[hashKey] {
			highestVotingPowerObserved = hashMap[hashKey]
			highestVotedHash = f.Hashes[i]
		}
	}

	for i := 0; i < len(f.Hashes); i++ {
		if bytes.Equal(highestVotedHash, f.Hashes[i]) {
			agreeVotesBitArray.SetIndex(i, true)
			agreeOracleSignatures[i] = f.OracleSignatures[i]
		}
	}

	fnAggregateResponse := &FnAggregateExecutionResponse{
		Hash:              highestVotedHash,
		SignatureBitArray: agreeVotesBitArray,
		OracleSignatures:  agreeOracleSignatures,
	}

	switch signingThreshold {
	case Maj23SigningThreshold:
		if highestVotingPowerObserved >= (currentValidatorSet.TotalVotingPower()*2/3 + 1) {
			return fnAggregateResponse
		}
		return nil
	case AllSigningThreshold:
		if highestVotingPowerObserved == currentValidatorSet.TotalVotingPower() {
			return fnAggregateResponse
		}
		return nil
	default:
		panic("unknown signing threshold type")
	}
}

func NewFnExecutionResponse(individualResponse *FnIndividualExecutionResponse, validatorIndex int, valSet *types.ValidatorSet) *FnExecutionResponse {
	newFnExecutionResponse := &FnExecutionResponse{
		Hashes: make([][]byte, valSet.Size()),
	}

	newFnExecutionResponse.OracleSignatures = make([][]byte, valSet.Size())
	newFnExecutionResponse.SignatureBitArray = cmn.NewBitArray(valSet.Size())

	newFnExecutionResponse.SignatureBitArray.SetIndex(validatorIndex, true)
	newFnExecutionResponse.Hashes[validatorIndex] = individualResponse.Hash
	newFnExecutionResponse.OracleSignatures[validatorIndex] = individualResponse.OracleSignature

	return newFnExecutionResponse
}

type FnVotePayload struct {
	Request  *FnExecutionRequest  `json:"fn_execution_request"`
	Response *FnExecutionResponse `json:"fn_execution_response"`
}

func (f *FnVotePayload) Marshal() ([]byte, error) {
	return cdc.MarshalBinaryLengthPrefixed(f)
}

func (f *FnVotePayload) Unmarshal(bz []byte) error {
	return cdc.UnmarshalBinaryLengthPrefixed(bz, f)
}

func (f *FnVotePayload) IsValid(currentValidatorSet *types.ValidatorSet) error {
	if f.Request == nil {
		return errors.New("fnVotePayload's request can't be nil")
	}

	if f.Response == nil {
		return errors.New("fnVotePayload's response can't be nil")
	}

	if err := f.Response.IsValid(currentValidatorSet); err != nil {
		return errors.Wrapf(err, "error encountered while validating Response")
	}

	return nil
}

func (f *FnVotePayload) Merge(anotherPayload *FnVotePayload) (bool, error) {
	if anotherPayload == nil {
		return false, fmt.Errorf("can't merge nil payload")
	}

	if !f.CannonicalCompare(anotherPayload) {
		return false, fmt.Errorf("can't merge as payload contents are different")
	}

	return f.Response.Merge(anotherPayload.Response)
}

func (f *FnVotePayload) CannonicalCompare(remotePayload *FnVotePayload) bool {
	if remotePayload == nil || remotePayload.Request == nil || remotePayload.Response == nil {
		return false
	}

	if !f.Request.CannonicalCompare(remotePayload.Request) {
		return false
	}

	if !f.Response.CannonicalCompare(remotePayload.Response) {
		return false
	}

	return true
}

func (f *FnVotePayload) Compare(remotePayload *FnVotePayload) bool {
	if remotePayload == nil || remotePayload.Request == nil || remotePayload.Response == nil {
		return false
	}

	if !f.Request.Compare(remotePayload.Request) {
		return false
	}

	if !f.Response.Compare(remotePayload.Response) {
		return false
	}

	return true
}

func (f *FnVotePayload) SignBytes(validatorIndex int) ([]byte, error) {
	requestSignBytes, err := f.Request.SignBytes()
	if err != nil {
		return nil, err
	}

	responseSignBytes, err := f.Response.SignBytes(validatorIndex)
	if err != nil {
		return nil, err
	}

	sepearator := []byte{0x50}

	signBytes := make([]byte, len(requestSignBytes)+len(responseSignBytes)+len(sepearator))

	copy(signBytes, requestSignBytes)
	copy(signBytes[len(requestSignBytes):], sepearator)
	copy(signBytes[len(requestSignBytes)+len(sepearator):], responseSignBytes)

	return signBytes, nil
}

func NewFnVotePayload(fnRequest *FnExecutionRequest, fnResponse *FnExecutionResponse) *FnVotePayload {
	return &FnVotePayload{
		Request:  fnRequest,
		Response: fnResponse,
	}
}

type FnVoteSet struct {
	Nonce               int64          `json:"nonce"`
	ValidatorsHash      []byte         `json:"validator_hash"`
	ChainID             string         `json:"chain_id"`
	TotalVotingPower    int64          `json:"total_voting_power"`
	VoteBitArray        *cmn.BitArray  `json:"vote_bitarray"`
	Payload             *FnVotePayload `json:"vote_payload"`
	ValidatorSignatures [][]byte       `json:"signature"`
	ValidatorAddresses  [][]byte       `json:"validator_address"`
}

func NewVoteSet(nonce int64, chainID string, validatorIndex int, initialPayload *FnVotePayload, privValidator types.PrivValidator, valSet *types.ValidatorSet) (*FnVoteSet, error) {
	voteBitArray := cmn.NewBitArray(valSet.Size())
	signatures := make([][]byte, valSet.Size())
	validatorAddresses := make([][]byte, valSet.Size())

	var totalVotingPower int64

	if err := initialPayload.IsValid(valSet); err != nil {
		return nil, errors.Wrap(err, "fnConsensusReactor: unable to create new voteSet as initialPayload passed is invalid")
	}

	valSet.Iterate(func(index int, validator *types.Validator) bool {
		if index == validatorIndex {
			totalVotingPower = validator.VotingPower
		}
		validatorAddresses[index] = validator.Address
		return false
	})

	voteBitArray.SetIndex(validatorIndex, true)

	if totalVotingPower == 0 {
		return nil, errors.New("fnConsensusReactor: unable to create new voteset as validatorIndex is invalid")
	}

	newVoteSet := &FnVoteSet{
		Nonce:               nonce,
		ValidatorsHash:      valSet.Hash(),
		ChainID:             chainID,
		TotalVotingPower:    totalVotingPower,
		Payload:             initialPayload,
		VoteBitArray:        voteBitArray,
		ValidatorSignatures: signatures,
		ValidatorAddresses:  validatorAddresses,
	}

	signBytes, err := newVoteSet.SignBytes(validatorIndex)
	if err != nil {
		return nil, errors.New("fnConsesnusReactor: unable to create new voteset as not able to get signbytes")
	}

	signature, err := privValidator.Sign(signBytes)
	if err != nil {
		return nil, errors.New("fnConsensusReactor: unable to create new voteset as not able to sign initial payload")
	}

	newVoteSet.ValidatorSignatures[validatorIndex] = signature

	return newVoteSet, nil
}

func (voteSet *FnVoteSet) Marshal() ([]byte, error) {
	return cdc.MarshalBinaryLengthPrefixed(voteSet)
}

func (voteSet *FnVoteSet) Unmarshal(bz []byte) error {
	return cdc.UnmarshalBinaryLengthPrefixed(bz, voteSet)
}

func (voteSet *FnVoteSet) CannonicalCompare(remoteVoteSet *FnVoteSet) bool {
	if voteSet.Nonce != remoteVoteSet.Nonce {
		return false
	}

	if voteSet.ChainID != remoteVoteSet.ChainID {
		return false
	}

	if !bytes.Equal(voteSet.ValidatorsHash, remoteVoteSet.ValidatorsHash) {
		return false
	}

	if remoteVoteSet.Payload == nil {
		return false
	}

	if !voteSet.Payload.CannonicalCompare(remoteVoteSet.Payload) {
		return false
	}

	if len(voteSet.ValidatorSignatures) != len(remoteVoteSet.ValidatorSignatures) {
		return false
	}

	// For misbehaving nodes
	if len(voteSet.ValidatorAddresses) != len(remoteVoteSet.ValidatorAddresses) {
		return false
	}

	for i := 0; i < len(voteSet.ValidatorAddresses); i++ {
		if !bytes.Equal(voteSet.ValidatorAddresses[i], remoteVoteSet.ValidatorAddresses[i]) {
			return false
		}
	}

	return true
}

func (voteset *FnVoteSet) ActiveValidators() [][]byte {
	activeValidators := make([][]byte, voteset.VoteBitArray.Size())
	j := 0
	for i := 0; i < voteset.VoteBitArray.Size(); i++ {
		if !voteset.VoteBitArray.GetIndex(i) {
			continue
		}

		activeValidators[j] = voteset.ValidatorAddresses[i]
		j++
	}

	return activeValidators[:j]
}

func (voteSet *FnVoteSet) SignBytes(validatorIndex int) ([]byte, error) {
	payloadBytes, err := voteSet.Payload.SignBytes(validatorIndex)
	if err != nil {
		return nil, err
	}

	var seperator = []byte{17, 19, 23, 29}

	prefix := []byte(fmt.Sprintf("NONCE:%d|CD:%s|VA:%s|PL:", voteSet.Nonce, voteSet.ChainID, voteSet.ValidatorAddresses[validatorIndex]))

	signBytes := make([]byte, len(prefix)+len(seperator)+len(voteSet.ValidatorsHash)+len(seperator)+len(payloadBytes))

	numCopied := 0

	copy(signBytes[numCopied:], prefix)
	numCopied += len(prefix)

	copy(signBytes[numCopied:], seperator)
	numCopied += len(seperator)

	copy(signBytes[numCopied:], voteSet.ValidatorsHash)
	numCopied += len(voteSet.ValidatorsHash)

	copy(signBytes[numCopied:], seperator)
	numCopied += len(seperator)

	copy(signBytes[numCopied:], payloadBytes)
	numCopied += len(payloadBytes)

	return signBytes, nil
}

func (voteSet *FnVoteSet) VerifyValidatorSign(validatorIndex int, pubKey crypto.PubKey) error {
	if !voteSet.VoteBitArray.GetIndex(validatorIndex) {
		return ErrFnVoteNotPresent
	}

	return voteSet.verifyInternal(voteSet.ValidatorSignatures[validatorIndex], validatorIndex,
		voteSet.ValidatorAddresses[validatorIndex], pubKey)
}

func (voteSet *FnVoteSet) verifyInternal(signature []byte, validatorIndex int, validatorAddress []byte, pubKey crypto.PubKey) error {
	if !bytes.Equal(pubKey.Address(), validatorAddress) {
		return ErrFnVoteInvalidValidatorAddress
	}

	signBytes, err := voteSet.SignBytes(validatorIndex)
	if err != nil {
		return err
	}

	if !pubKey.VerifyBytes(signBytes, signature) {
		return ErrFnVoteInvalidSignature
	}
	return nil
}

func (voteSet *FnVoteSet) GetFnID() string {
	return voteSet.Payload.Request.FnID
}

func (voteSet *FnVoteSet) NumberOfVotes() int {
	numberOfVotes := 0
	for i := 0; i < voteSet.VoteBitArray.Size(); i++ {
		if voteSet.VoteBitArray.GetIndex(i) {
			numberOfVotes++
		}
	}
	return numberOfVotes
}

func (voteSet *FnVoteSet) HasConverged(signingThreshold SigningThreshold, currentValidatorSet *types.ValidatorSet) bool {
	switch signingThreshold {
	case Maj23SigningThreshold:
		return voteSet.TotalVotingPower >= currentValidatorSet.TotalVotingPower()*2/3+1
	case AllSigningThreshold:
		return voteSet.TotalVotingPower == currentValidatorSet.TotalVotingPower()
	default:
		panic("unknown signing threshold")
	}
}

func (voteSet *FnVoteSet) HaveWeAlreadySigned(ownValidatorIndex int) bool {
	return voteSet.VoteBitArray.GetIndex(ownValidatorIndex)
}

// Should be the first function to be invoked on vote set received from Peer
func (voteSet *FnVoteSet) IsValid(chainID string, currentValidatorSet *types.ValidatorSet, registry FnRegistry) error {
	var calculatedVotingPower int64

	// This if conditions are individual as, we want to pass different errors for each
	// condition in future.

	if voteSet.VoteBitArray == nil {
		return errors.New("voteSet.VoteBitArray can't be nil")
	}

	numValidators := voteSet.VoteBitArray.Size()

	if voteSet.Payload == nil {
		return errors.New("voteSet.Payload can't be nil")
	}

	if voteSet.ValidatorAddresses == nil {
		return errors.New("voteSet.ValidatorAddresses can't be nil")
	}

	if err := voteSet.Payload.IsValid(currentValidatorSet); err != nil {
		return errors.Wrapf(err, "voteSet.Payload isnt valid")
	}

	if registry.Get(voteSet.GetFnID()) == nil {
		return errors.New("voteSet's FnID cannot be found in FnRegistry")
	}

	if voteSet.ChainID != chainID {
		return errors.New("voteSet.ChainID doesn't match node's ChainID")
	}

	if !bytes.Equal(voteSet.ValidatorsHash, currentValidatorSet.Hash()) {
		return fmt.Errorf("voteSet.ValidatorHash doesn't match node's validator hash, Expected: %v, Got: %v",
			currentValidatorSet.Hash(), voteSet.ValidatorsHash)
	}

	if numValidators != len(voteSet.ValidatorAddresses) {
		return errors.New("voteSet.ValidatorAddresses has different length than node's validator list")
	}

	if numValidators != len(voteSet.ValidatorSignatures) {
		return errors.New("voteSet.ValidatorSignatures has different length than node's validator list")
	}

	if numValidators != currentValidatorSet.Size() {
		return errors.New("voteSet.VoteBitArray size is different than current node's validator list")
	}

	if numValidators != voteSet.Payload.Response.SignatureBitArray.Size() {
		return errors.New("voteSet.Payload.Response.SignatureBitArray size is different than current node's validator list")
	}

	var iteratingError error

	currentValidatorSet.Iterate(func(i int, val *types.Validator) bool {
		if !bytes.Equal(voteSet.ValidatorAddresses[i], val.Address) {
			iteratingError = errors.New("voteSet.ValidatorAddresses  and current validator set mismatch")
			return true
		}

		if voteSet.VoteBitArray.GetIndex(i) != voteSet.Payload.Response.SignatureBitArray.GetIndex(i) {
			iteratingError = errors.New("voteSet.VoteBitArray and voteSet.Payload.Response.SignatureBitArray mismatch")
			return true
		}

		if !voteSet.VoteBitArray.GetIndex(i) {
			return false
		}

		if voteSet.Payload.Response.OracleSignatures[i] == nil {
			iteratingError = errors.New("voteSet.Payload.Response.OracleSignature and voteSet.VoteBitArray mismatch")
			return true
		}

		if err := voteSet.VerifyValidatorSign(i, val.PubKey); err != nil {
			iteratingError = errors.New(fmt.Sprintf("unable to verify validator sign, PubKey: %s\n", val.PubKey))
			return true
		}

		calculatedVotingPower += val.VotingPower
		return false
	})

	if iteratingError != nil {
		return iteratingError
	}

	// Voting power contained in VoteSet should match the calculated voting power
	if voteSet.TotalVotingPower != calculatedVotingPower {
		return errors.New("voteSet.TotalVotingPower is not equal to calculated voting power")
	}

	return nil
}

func (voteSet *FnVoteSet) Merge(valSet *types.ValidatorSet, anotherSet *FnVoteSet) (bool, error) {
	hasChanged := false

	if !voteSet.CannonicalCompare(anotherSet) {
		return hasChanged, ErrFnVoteMergeDiffPayload
	}

	numValidators := voteSet.VoteBitArray.Size()

	hasPayloadChanged, err := voteSet.Payload.Merge(anotherSet.Payload)
	if err != nil {
		return false, err
	}

	hasChanged = hasPayloadChanged

	for i := 0; i < numValidators; i++ {
		if voteSet.VoteBitArray.GetIndex(i) || !anotherSet.VoteBitArray.GetIndex(i) {
			continue
		}

		_, currentValidator := valSet.GetByIndex(i)

		hasChanged = true

		voteSet.ValidatorSignatures[i] = anotherSet.ValidatorSignatures[i]
		voteSet.ValidatorAddresses[i] = anotherSet.ValidatorAddresses[i]

		voteSet.VoteBitArray.SetIndex(i, true)

		voteSet.TotalVotingPower += currentValidator.VotingPower
	}

	return hasChanged, nil
}

func (voteSet *FnVoteSet) MajResponse(signingThreshold SigningThreshold, validatorSet *types.ValidatorSet) *FnAggregateExecutionResponse {
	return voteSet.Payload.Response.ToMajResponse(signingThreshold, validatorSet)
}

func (voteSet *FnVoteSet) AddVote(nonce int64, individualExecutionResponse *FnIndividualExecutionResponse, currentValidatorSet *types.ValidatorSet, validatorIndex int, privValidator types.PrivValidator) error {
	if voteSet.Nonce != nonce {
		return fmt.Errorf("FnConsensusReactor: unable to add vote as nonce is different from voteset")
	}

	if voteSet.VoteBitArray.GetIndex(validatorIndex) {
		return ErrFnVoteAlreadyCast
	}

	if err := voteSet.Payload.Response.AddSignature(individualExecutionResponse, validatorIndex); err != nil {
		return fmt.Errorf("fnConsesnusReactor: unable to add vote as can't add signature, Error: %s", err.Error())
	}

	signBytes, err := voteSet.SignBytes(validatorIndex)
	if err != nil {
		return fmt.Errorf("fnConsensusReactor: unable to add vote as unable to get sign bytes. Error: %s", err.Error())
	}

	signature, err := privValidator.Sign(signBytes)
	if err != nil {
		return fmt.Errorf("fnConsensusReactor: unable to add vote as unable to sign signing bytes. Error: %s", err.Error())
	}

	voteSet.VoteBitArray.SetIndex(validatorIndex, true)

	voteSet.ValidatorSignatures[validatorIndex] = signature

	_, validator := currentValidatorSet.GetByIndex(validatorIndex)
	if validator == nil {
		return fmt.Errorf("fnConsensusReactor: unable to add vote as validatorIndex is not valid")
	}

	if !bytes.Equal(validator.Address, voteSet.ValidatorAddresses[validatorIndex]) {
		return fmt.Errorf("fnConsensusReactor: unable to add vote as validatorAddress does not match with one in the vote set")
	}

	voteSet.TotalVotingPower += validator.VotingPower

	return nil
}

func RegisterFnConsensusTypes() {
	cdc.RegisterConcrete(&FnExecutionRequest{}, "tendermint/fnConsensusReactor/FnExecutionRequest", nil)
	cdc.RegisterConcrete(&FnExecutionResponse{}, "tendermint/fnConsensusReactor/FnExecutionResponse", nil)
	cdc.RegisterConcrete(&FnVoteSet{}, "tendermint/fnConsensusReactor/FnVoteSet", nil)
	cdc.RegisterConcrete(&FnVotePayload{}, "tendermint/fnConsensusReactor/FnVotePayload", nil)
	cdc.RegisterConcrete(&FnIndividualExecutionResponse{}, "tendermint/fnConsensusReactor/FnIndividualExecutionResponse", nil)
	cdc.RegisterConcrete(&ReactorState{}, "tendermint/fnConsensusReactor/ReactorState", nil)
	cdc.RegisterConcrete(&reactorStateMarshallable{}, "tendermint/fnConsensusReactor/reactorStateMarshallable", nil)
	cdc.RegisterConcrete(&fnIDToNonce{}, "tendermint/fnConsensusReactor/fnIDToNonce", nil)
}
