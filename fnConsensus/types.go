package fnConsensus

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	"github.com/tendermint/tendermint/crypto"
	cmn "github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/types"

	"crypto/sha256"
)

var ErrFnVoteInvalidValidatorAddress = errors.New("invalid validator address for FnVote")
var ErrFnVoteInvalidSignature = errors.New("invalid validator signature")
var ErrFnVoteInvalidProposerSignature = errors.New("invalid proposer signature")
var ErrFnVoteNotPresent = errors.New("Fn vote is not present for validator")
var ErrFnVoteAlreadyCast = errors.New("Fn vote is already cast")
var ErrFnResponseSignatureAlreadyPresent = errors.New("Fn Response signature is already present")

var ErrFnVoteMergeDiffPayload = errors.New("merging is not allowed, as votes have different payload")

type VoteType bool

const VoteTypeAgree = VoteType(true)
const VoteTypeDisAgree = VoteType(false)

type fnIDToNonce struct {
	Nonce int64
	FnID  string
}

type FnIndividualExecutionResponse struct {
	Status          int64
	Error           string
	Hash            []byte
	OracleSignature []byte
}

func (f *FnIndividualExecutionResponse) Marshal() ([]byte, error) {
	return cdc.MarshalBinaryLengthPrefixed(f)
}

type fnIDToProposalInfo struct {
	CurrentProposalInfo *ProposalInfo
	FnID                string
}

func (f *fnIDToProposalInfo) Marshal() ([]byte, error) {
	return cdc.MarshalBinaryLengthPrefixed(f)
}

type ProposalInfo struct {
	LastActiveValidators [][]byte
	CurrentTurn          int
}

func (p *ProposalInfo) CurrentProposer() []byte {
	return p.LastActiveValidators[p.CurrentTurn]
}

func (p *ProposalInfo) Hash() ([]byte, error) {
	hasher := sha256.New()

	for _, validatorAddress := range p.LastActiveValidators {
		_, err := hasher.Write(validatorAddress)
		if err != nil {
			return nil, err
		}
	}

	return hasher.Sum(nil), nil
}

func (p *ProposalInfo) Equal(other *ProposalInfo) (bool, error) {
	if other == nil {
		return false, nil
	}

	ownHash, err := p.Hash()
	if err != nil {
		return false, err
	}

	otherHash, err := other.Hash()
	if err != nil {
		return false, err
	}

	if !bytes.Equal(ownHash, otherHash) {
		return false, nil
	}

	if p.CurrentTurn != other.CurrentTurn {
		return false, nil
	}

	return true, nil
}

func (p *ProposalInfo) Marshal() ([]byte, error) {
	return cdc.MarshalBinaryLengthPrefixed(p)
}

func (p *ProposalInfo) Unmarshal(bz []byte) error {
	return cdc.UnmarshalBinaryLengthPrefixed(bz, p)
}

type reactorSetMarshallable struct {
	CurrentVoteSets          []*FnVoteSet
	CurrentNonces            []*fnIDToNonce
	CurrentProposalInfo      []*fnIDToProposalInfo
	PreviousTimedOutVoteSets []*FnVoteSet
	PreviousMaj23VoteSets    []*FnVoteSet
	PreviousValidatorSet     *types.ValidatorSet
	PetitionNonce            int64
}

type ReactorState struct {
	CurrentVoteSets          map[string]*FnVoteSet
	CurrentNonces            map[string]int64
	CurrentProposalInfo      map[string]*ProposalInfo
	PreviousTimedOutVoteSets map[string]*FnVoteSet
	PreviousMaj23VoteSets    map[string]*FnVoteSet
	PreviousValidatorSet     *types.ValidatorSet
	PetitionNonce            int64
}

func (p *ReactorState) Marshal() ([]byte, error) {
	reactorStateMarshallable := &reactorSetMarshallable{
		CurrentVoteSets:          make([]*FnVoteSet, len(p.CurrentVoteSets)),
		CurrentNonces:            make([]*fnIDToNonce, len(p.CurrentNonces)),
		CurrentProposalInfo:      make([]*fnIDToProposalInfo, len(p.CurrentProposalInfo)),
		PreviousTimedOutVoteSets: make([]*FnVoteSet, len(p.PreviousTimedOutVoteSets)),
		PreviousMaj23VoteSets:    make([]*FnVoteSet, len(p.PreviousMaj23VoteSets)),
		PreviousValidatorSet:     p.PreviousValidatorSet,
		PetitionNonce:            p.PetitionNonce,
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
	for fnID, currentProposalInfo := range p.CurrentProposalInfo {
		reactorStateMarshallable.CurrentProposalInfo[i] = &fnIDToProposalInfo{
			FnID:                fnID,
			CurrentProposalInfo: currentProposalInfo,
		}
		i++
	}

	i = 0
	for _, timedOutVoteSet := range p.PreviousTimedOutVoteSets {
		reactorStateMarshallable.PreviousTimedOutVoteSets[i] = timedOutVoteSet
		i++
	}

	i = 0
	for _, maj23VoteSet := range p.PreviousMaj23VoteSets {
		reactorStateMarshallable.PreviousMaj23VoteSets[i] = maj23VoteSet
		i++
	}

	return cdc.MarshalBinaryLengthPrefixed(reactorStateMarshallable)
}

func (p *ReactorState) Unmarshal(bz []byte) error {
	reactorStateMarshallable := &reactorSetMarshallable{}
	if err := cdc.UnmarshalBinaryLengthPrefixed(bz, reactorStateMarshallable); err != nil {
		return err
	}

	p.CurrentVoteSets = make(map[string]*FnVoteSet)
	p.CurrentNonces = make(map[string]int64)
	p.CurrentProposalInfo = make(map[string]*ProposalInfo)
	p.PreviousTimedOutVoteSets = make(map[string]*FnVoteSet)
	p.PreviousMaj23VoteSets = make(map[string]*FnVoteSet)
	p.PreviousValidatorSet = reactorStateMarshallable.PreviousValidatorSet
	p.PetitionNonce = reactorStateMarshallable.PetitionNonce

	for _, voteSet := range reactorStateMarshallable.CurrentVoteSets {
		p.CurrentVoteSets[voteSet.Payload.Request.FnID] = voteSet
	}

	for _, fnIDToNonce := range reactorStateMarshallable.CurrentNonces {
		p.CurrentNonces[fnIDToNonce.FnID] = fnIDToNonce.Nonce
	}

	for _, fnIDToProposalInfo := range reactorStateMarshallable.CurrentProposalInfo {
		p.CurrentProposalInfo[fnIDToProposalInfo.FnID] = fnIDToProposalInfo.CurrentProposalInfo
	}

	for _, timeOutVoteSet := range reactorStateMarshallable.PreviousTimedOutVoteSets {
		p.PreviousTimedOutVoteSets[timeOutVoteSet.Payload.Request.FnID] = timeOutVoteSet
	}

	for _, maj23VoteSet := range reactorStateMarshallable.PreviousMaj23VoteSets {
		p.PreviousMaj23VoteSets[maj23VoteSet.Payload.Request.FnID] = maj23VoteSet
	}

	return nil
}

func NewReactorState() *ReactorState {
	return &ReactorState{
		CurrentVoteSets:          make(map[string]*FnVoteSet),
		CurrentNonces:            make(map[string]int64),
		CurrentProposalInfo:      make(map[string]*ProposalInfo),
		PreviousTimedOutVoteSets: make(map[string]*FnVoteSet),
		PreviousMaj23VoteSets:    make(map[string]*FnVoteSet),
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

type FnExecutionResponse struct {
	Status int64
	Error  string
	Hash   []byte
	// Indexed by validator index in Current validator set
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
		f.SignatureBitArray.SetIndex(i, true)
	}

	return hasResponseChanged, nil
}

func (f *FnExecutionResponse) IsValid(currentValidatorSet *types.ValidatorSet) bool {
	if f.Hash == nil {
		return false
	}

	if f.SignatureBitArray == nil {
		return false
	}

	if currentValidatorSet.Size() != len(f.OracleSignatures) {
		return false
	}

	if currentValidatorSet.Size() != f.SignatureBitArray.Size() {
		return false
	}

	return true
}

func (f *FnExecutionResponse) CannonicalCompare(remoteResponse *FnExecutionResponse) bool {
	if f.Error != remoteResponse.Error {
		return false
	}

	if f.Status != remoteResponse.Status {
		return false
	}

	if !bytes.Equal(f.Hash, remoteResponse.Hash) {
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

func (f *FnExecutionResponse) CannonicalCompareWithIndividualExecution(individualExecution *FnIndividualExecutionResponse) bool {
	if f.Status != individualExecution.Status || f.Error != individualExecution.Error || !bytes.Equal(f.Hash, individualExecution.Hash) {
		return false
	}

	return true
}

func (f *FnExecutionResponse) SignBytes(validatorIndex int) ([]byte, error) {
	individualResponse := &FnIndividualExecutionResponse{
		Status:          f.Status,
		Error:           f.Error,
		Hash:            f.Hash,
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

	return true
}

func (f *FnExecutionResponse) AddSignature(validatorIndex int, signature []byte) error {
	if f.SignatureBitArray.GetIndex(validatorIndex) {
		return ErrFnResponseSignatureAlreadyPresent
	}

	f.OracleSignatures[validatorIndex] = signature
	f.SignatureBitArray.SetIndex(validatorIndex, true)
	return nil
}

func NewFnExecutionResponse(individualResponse *FnIndividualExecutionResponse, validatorIndex int, valSet *types.ValidatorSet) *FnExecutionResponse {
	newFnExecutionResponse := &FnExecutionResponse{
		Status: individualResponse.Status,
		Error:  individualResponse.Error,
		Hash:   individualResponse.Hash,
	}

	newFnExecutionResponse.OracleSignatures = make([][]byte, valSet.Size())
	newFnExecutionResponse.SignatureBitArray = cmn.NewBitArray(valSet.Size())

	newFnExecutionResponse.SignatureBitArray.SetIndex(validatorIndex, true)
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

func (f *FnVotePayload) IsValid(currentValidatorSet *types.ValidatorSet) bool {
	if f.Request == nil || f.Response == nil {
		return false
	}

	if !f.Response.IsValid(currentValidatorSet) {
		return false
	}

	return true
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

type FnPetition struct {
	ID                    string        `json:"id"`
	Nonce                 int64         `json:"nonce"`
	ChainID               string        `json:"chain_id"`
	CreationTime          int64         `json:"creation_time"`
	VoteBitArray          *cmn.BitArray `json:"vote_bitarray"`
	PetitionID            string        `json:"petition_id"`
	Payload               []byte        `json:"payload"`
	ValidatorsHash        []byte        `json:"validator_hash"`
	ValidatorSignatures   [][]byte      `json:"signature"`
	ValidatorAddresses    [][]byte      `json:"validator_address"`
	TotalAgreeVotingPower int64         `json:"total_agreed_voting_power"`
}

func (f *FnPetition) Marshal() ([]byte, error) {
	return cdc.MarshalBinaryLengthPrefixed(f)
}

func (f *FnPetition) Unmarshal(bz []byte) error {
	return cdc.UnmarshalBinaryLengthPrefixed(bz, f)
}

func (f *FnPetition) SignBytes(validatorIndex int) ([]byte, error) {
	var seperator = []byte{18, 20, 24, 30}

	prefix := []byte(fmt.Sprintf("ID:%s|NONCE:%d|CT:%d|CD:%s|VA:%s|PID:%s|PL:", f.ID, f.Nonce, f.CreationTime,
		f.ChainID, f.ValidatorAddresses[validatorIndex], f.PetitionID))

	signBytes := make([]byte, len(prefix)+len(seperator)+len(f.ValidatorsHash)+len(seperator)+len(f.Payload))

	numCopied := 0

	copy(signBytes[numCopied:], prefix)
	numCopied += len(prefix)

	copy(signBytes[numCopied:], seperator)
	numCopied += len(seperator)

	copy(signBytes[numCopied:], f.ValidatorsHash)
	numCopied += len(f.ValidatorsHash)

	copy(signBytes[numCopied:], seperator)
	numCopied += len(seperator)

	copy(signBytes[numCopied:], f.Payload)
	numCopied += len(f.Payload)

	return signBytes, nil
}

func (f *FnPetition) IsValid(nonce int64, chainID string, currentValidatorSet *types.ValidatorSet, maxPayloadSize int) bool {
	isValid := true
	var calculatedAgreedVotingPower int64

	if f.VoteBitArray == nil || f.PetitionID == "" || f.Payload == nil || f.ValidatorsHash == nil || f.ValidatorAddresses == nil || f.ValidatorSignatures == nil {
		isValid = false
		return false
	}

	if len(f.Payload) > maxPayloadSize {
		isValid = false
		return false
	}

	if f.Nonce != nonce {
		isValid = false
		return false
	}

	if f.ChainID != chainID {
		isValid = false
		return false
	}

	currentValidatorSet.Iterate(func(i int, val *types.Validator) bool {
		if !bytes.Equal(f.ValidatorAddresses[i], val.Address) {
			isValid = false
			return true
		}

		if !f.VoteBitArray.GetIndex(i) {
			return false
		}

		if err := f.VerifyValidatorSign(i, val.PubKey); err != nil {
			isValid = false
			return true
		}

		calculatedAgreedVotingPower += val.VotingPower

		return false
	})

	if calculatedAgreedVotingPower != f.TotalAgreeVotingPower {
		isValid = false
		return false
	}

	return isValid
}

func (f *FnPetition) VerifyValidatorSign(validatorIndex int, pubKey crypto.PubKey) error {
	if !f.VoteBitArray.GetIndex(validatorIndex) {
		return ErrFnVoteNotPresent
	}

	return f.verifyInternal(f.ValidatorSignatures[validatorIndex], validatorIndex,
		f.ValidatorAddresses[validatorIndex], pubKey)
}

func (f *FnPetition) verifyInternal(signature []byte, validatorIndex int, validatorAddress []byte, pubKey crypto.PubKey) error {
	if !bytes.Equal(pubKey.Address(), validatorAddress) {
		return ErrFnVoteInvalidValidatorAddress
	}

	signBytes, err := f.SignBytes(validatorIndex)
	if err != nil {
		return err
	}

	if !pubKey.VerifyBytes(signBytes, signature) {
		return ErrFnVoteInvalidSignature
	}
	return nil
}

func (f *FnPetition) HaveWeAlreadySigned(ownValidatorIndex int) bool {
	return f.VoteBitArray.GetIndex(ownValidatorIndex)
}

func (f *FnPetition) AddAgreeVote(nonce int64, currentValidatorSet *types.ValidatorSet, validatorIndex int, privValidator types.PrivValidator) error {
	if f.Nonce != nonce {
		return fmt.Errorf("FnConsensusReactor: unable to add vote as nonce is different from voteset")
	}

	if f.VoteBitArray.GetIndex(validatorIndex) {
		return ErrFnVoteAlreadyCast
	}

	signBytes, err := f.SignBytes(validatorIndex)
	if err != nil {
		return fmt.Errorf("fnConsensusReactor: unable to add vote as unable to get sign bytes. Error: %s", err.Error())
	}

	signature, err := privValidator.Sign(signBytes)
	if err != nil {
		return fmt.Errorf("fnConsensusReactor: unable to add vote as unable to sign signing bytes. Error: %s", err.Error())
	}

	f.VoteBitArray.SetIndex(validatorIndex, true)
	f.ValidatorSignatures[validatorIndex] = signature

	_, validator := currentValidatorSet.GetByIndex(validatorIndex)
	if validator == nil {
		return fmt.Errorf("fnConsensusReactor: unable to add vote as validatorIndex is not valid")
	}

	if !bytes.Equal(validator.Address, f.ValidatorAddresses[validatorIndex]) {
		return fmt.Errorf("fnConsensusReactor: unable to add vote as validatorAddress does not match with one in the vote set")
	}

	f.TotalAgreeVotingPower += validator.VotingPower

	return nil
}

func (f *FnPetition) IsAgree(signingThreshold SigningThreshold, currentValidatorSet *types.ValidatorSet) bool {
	switch signingThreshold {
	case Maj23SigningThreshold:
		return f.TotalAgreeVotingPower >= currentValidatorSet.TotalVotingPower()*2/3+1
	case AllSigningThreshold:
		return f.TotalAgreeVotingPower == currentValidatorSet.TotalVotingPower()
	default:
		panic("unknown signing threshold")
	}
}

func (f *FnPetition) HasConverged(signingThreshold SigningThreshold, currentValidatorSet *types.ValidatorSet) bool {
	return f.IsAgree(signingThreshold, currentValidatorSet)
}

func NewFnPetition(id string, nonce int64, chainId string, payload []byte, petitionID string, privValidator types.PrivValidator, validatorIndex int, valSet *types.ValidatorSet) (*FnPetition, error) {
	voteBitArray := cmn.NewBitArray(valSet.Size())
	signatures := make([][]byte, valSet.Size())
	validatorAddresses := make([][]byte, valSet.Size())

	var totalAgreeVotingPower int64

	valSet.Iterate(func(index int, validator *types.Validator) bool {
		if index == validatorIndex {
			totalAgreeVotingPower = validator.VotingPower
		}
		validatorAddresses[index] = validator.Address
		return false
	})

	voteBitArray.SetIndex(validatorIndex, true)

	if totalAgreeVotingPower == 0 {
		return nil, fmt.Errorf("fnConsensusReactor: unable to create new petition as validatorIndex is invalid")
	}

	newPetition := &FnPetition{
		ID:                    id,
		Nonce:                 nonce,
		ChainID:               chainId,
		CreationTime:          time.Now().Unix(),
		VoteBitArray:          voteBitArray,
		PetitionID:            petitionID,
		Payload:               payload,
		ValidatorsHash:        valSet.Hash(),
		ValidatorSignatures:   signatures,
		ValidatorAddresses:    validatorAddresses,
		TotalAgreeVotingPower: totalAgreeVotingPower,
	}

	signBytes, err := newPetition.SignBytes(validatorIndex)
	if err != nil {
		return nil, fmt.Errorf("fnConsesnusReactor: unable to create new petition as not able to get signbytes")
	}

	signature, err := privValidator.Sign(signBytes)
	if err != nil {
		return nil, fmt.Errorf("fnConsensusReactor: unable to create new petition as not able to sign initial payload")
	}

	newPetition.ValidatorSignatures[validatorIndex] = signature

	return newPetition, nil
}

type FnVoteSet struct {
	ID                       string         `json:"id"`
	Nonce                    int64          `json:"nonce"`
	ValidatorsHash           []byte         `json:"validator_hash"`
	ChainID                  string         `json:"chain_id"`
	TotalAgreeVotingPower    int64          `json:"total_voting_power"`
	TotalDisagreeVotingPower int64          `json:"total_disagree_voting_power"`
	CreationTime             int64          `json:"creation_time"`
	VoteBitArray             *cmn.BitArray  `json:"vote_bitarray"`
	VoteTypeBitArray         *cmn.BitArray  `json:"votetype_bitarray"`
	Payload                  *FnVotePayload `json:"vote_payload"`
	ExecutionContext         []byte         `json:"execution_context"`
	ValidatorSignatures      [][]byte       `json:"signature"`
	ProposerSignature        []byte         `json:"proposer_signature"`
	ValidatorAddresses       [][]byte       `json:"validator_address"`
	ProposalInfo             *ProposalInfo  `json:"proposal_info"`
}

func NewVoteSet(id string, nonce int64, chainID string, expiresIn time.Duration, validatorIndex int, executionContext []byte, initialPayload *FnVotePayload, privValidator types.PrivValidator, valSet *types.ValidatorSet, proposalInfo *ProposalInfo) (*FnVoteSet, error) {
	voteBitArray := cmn.NewBitArray(valSet.Size())
	voteTypeBitArray := cmn.NewBitArray(valSet.Size())
	signatures := make([][]byte, valSet.Size())
	validatorAddresses := make([][]byte, valSet.Size())

	var totalAgreeVotingPower int64

	if !initialPayload.IsValid(valSet) {
		return nil, fmt.Errorf("fnConsensusReactor: unable to create new voteSet as initialPayload passed is invalid")
	}

	valSet.Iterate(func(index int, validator *types.Validator) bool {
		if index == validatorIndex {
			totalAgreeVotingPower = validator.VotingPower
		}
		validatorAddresses[index] = validator.Address
		return false
	})

	voteBitArray.SetIndex(validatorIndex, true)

	// Added Non-nil vote
	voteTypeBitArray.SetIndex(validatorIndex, bool(VoteTypeAgree))

	if totalAgreeVotingPower == 0 {
		return nil, fmt.Errorf("fnConsensusReactor: unable to create new voteset as validatorIndex is invalid")
	}

	newVoteSet := &FnVoteSet{
		ID:                       id,
		Nonce:                    nonce,
		ValidatorsHash:           valSet.Hash(),
		ChainID:                  chainID,
		TotalAgreeVotingPower:    totalAgreeVotingPower,
		TotalDisagreeVotingPower: 0,
		CreationTime:             time.Now().Unix(),
		Payload:                  initialPayload,
		VoteBitArray:             voteBitArray,
		VoteTypeBitArray:         voteTypeBitArray,
		ExecutionContext:         executionContext,
		ValidatorSignatures:      signatures,
		ValidatorAddresses:       validatorAddresses,
		ProposalInfo:             proposalInfo,
	}

	signBytes, err := newVoteSet.SignBytes(validatorIndex, VoteTypeAgree)
	if err != nil {
		return nil, fmt.Errorf("fnConsesnusReactor: unable to create new voteset as not able to get signbytes")
	}

	signature, err := privValidator.Sign(signBytes)
	if err != nil {
		return nil, fmt.Errorf("fnConsensusReactor: unable to create new voteset as not able to sign initial payload")
	}

	newVoteSet.ValidatorSignatures[validatorIndex] = signature

	proposalSignBytes, err := newVoteSet.ProposalSignBytes(validatorIndex)
	if err != nil {
		return nil, fmt.Errorf("fnConsensusReactor: unable to create new voteset as not able to get proposal sign bytes")
	}
	proposerSignature, err := privValidator.Sign(proposalSignBytes)
	if err != nil {
		return nil, fmt.Errorf("fnConsensusReactor: unable to create new voteset as not able to sign proposal bytes")
	}

	newVoteSet.ProposerSignature = proposerSignature

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

	if voteSet.CreationTime != remoteVoteSet.CreationTime {
		return false
	}

	if voteSet.ID != remoteVoteSet.ID {
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

	areEqual, err := voteSet.ProposalInfo.Equal(remoteVoteSet.ProposalInfo)
	if err != nil {
		// TODO: pass the error
		return false
	}

	if !areEqual {
		return false
	}

	// For misbehaving nodes
	if len(voteSet.ValidatorAddresses) != len(remoteVoteSet.ValidatorAddresses) {
		return false
	}

	if !bytes.Equal(voteSet.ExecutionContext, remoteVoteSet.ExecutionContext) {
		return false
	}

	for i := 0; i < len(voteSet.ValidatorAddresses); i++ {
		if !bytes.Equal(voteSet.ValidatorAddresses[i], remoteVoteSet.ValidatorAddresses[i]) {
			return false
		}
	}

	return true
}

func (voteSet *FnVoteSet) ProposalSignBytes(validatorIndex int) ([]byte, error) {
	if voteSet.ProposalInfo != nil {
		return nil, fmt.Errorf("proposal info cant be nil")
	}

	signBytes, err := voteSet.SignBytes(validatorIndex, VoteTypeAgree)
	if err != nil {
		return nil, err
	}

	validatorAddress := voteSet.ValidatorAddresses[validatorIndex]
	currentProposer := voteSet.ProposalInfo.CurrentProposer()

	if !bytes.Equal(validatorAddress, currentProposer) {
		return nil, fmt.Errorf("func ProposalSignBytes can only be invoked in context of proposer")
	}

	return append(signBytes, validatorAddress...), nil
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

func (voteSet *FnVoteSet) SignBytes(validatorIndex int, voteType VoteType) ([]byte, error) {
	payloadBytes, err := voteSet.Payload.SignBytes(validatorIndex)
	if err != nil {
		return nil, err
	}

	var seperator = []byte{17, 19, 23, 29}

	proposalInfoHash, err := voteSet.ProposalInfo.Hash()
	if err != nil {
		return nil, err
	}

	prefix := []byte(fmt.Sprintf("ID:%s|NONCE:%d|CT:%d|CD:%s|VA:%s|PCT:%d|PIH:%s|VT:%v|PL:", voteSet.ID, voteSet.Nonce, voteSet.CreationTime,
		voteSet.ChainID, voteSet.ValidatorAddresses[validatorIndex], voteSet.ProposalInfo.CurrentTurn, proposalInfoHash, voteType))

	signBytes := make([]byte, len(prefix)+len(seperator)+len(voteSet.ExecutionContext)+len(seperator)+len(voteSet.ValidatorsHash)+len(seperator)+len(payloadBytes))

	numCopied := 0

	copy(signBytes[numCopied:], prefix)
	numCopied += len(prefix)

	copy(signBytes[numCopied:], seperator)
	numCopied += len(seperator)

	copy(signBytes[numCopied:], voteSet.ExecutionContext)
	numCopied += len(voteSet.ExecutionContext)

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

func (voteSet *FnVoteSet) VerifyProposerSign(validatorSet *types.ValidatorSet) error {

	proposerIndex := -1
	proposer := voteSet.ProposalInfo.CurrentProposer()
	for i, validatorAddress := range voteSet.ValidatorAddresses {
		if bytes.Equal(validatorAddress, proposer) {
			proposerIndex = i
			break
		}
	}

	if proposerIndex == -1 {
		return fmt.Errorf("proposer is not a part of validator addresses")
	}

	_, val := validatorSet.GetByIndex(proposerIndex)
	if val == nil {
		return fmt.Errorf("proposer is not part of current validator set")
	}

	proposerSignBytes, err := voteSet.ProposalSignBytes(proposerIndex)
	if err != nil {
		return err
	}

	if !val.PubKey.VerifyBytes(proposerSignBytes, voteSet.ProposerSignature) {
		return ErrFnVoteInvalidSignature
	}

	return nil
}

func (voteSet *FnVoteSet) VerifyValidatorSign(validatorIndex int, voteType VoteType, pubKey crypto.PubKey) error {
	if !voteSet.VoteBitArray.GetIndex(validatorIndex) {
		return ErrFnVoteNotPresent
	}

	return voteSet.verifyInternal(voteSet.ValidatorSignatures[validatorIndex], validatorIndex, voteType,
		voteSet.ValidatorAddresses[validatorIndex], pubKey)
}

func (voteSet *FnVoteSet) verifyInternal(signature []byte, validatorIndex int, voteType VoteType, validatorAddress []byte, pubKey crypto.PubKey) error {
	if !bytes.Equal(pubKey.Address(), validatorAddress) {
		return ErrFnVoteInvalidValidatorAddress
	}

	signBytes, err := voteSet.SignBytes(validatorIndex, voteType)
	if err != nil {
		return err
	}

	if !pubKey.VerifyBytes(signBytes, signature) {
		return ErrFnVoteInvalidSignature
	}
	return nil
}

func (voteSet *FnVoteSet) IsExpired(expiresAfter time.Duration) bool {
	creationTime := time.Unix(voteSet.CreationTime, 0)
	expiryTime := creationTime.Add(expiresAfter)

	return expiryTime.Before(time.Now().UTC())
}

func (voteSet *FnVoteSet) GetMessageHash() []byte {
	return voteSet.Payload.Response.Hash
}

func (voteSet *FnVoteSet) GetFnID() string {
	return voteSet.Payload.Request.FnID
}

func (voteSet *FnVoteSet) IsAgree(signingThreshold SigningThreshold, currentValidatorSet *types.ValidatorSet) bool {
	switch signingThreshold {
	case Maj23SigningThreshold:
		return voteSet.TotalAgreeVotingPower >= currentValidatorSet.TotalVotingPower()*2/3+1
	case AllSigningThreshold:
		return voteSet.TotalAgreeVotingPower == currentValidatorSet.TotalVotingPower()
	default:
		panic("unknown signing threshold")
	}
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

func (voteSet *FnVoteSet) NumberOfAgreeVotes() int {
	numberOfAgreeVotes := 0
	for i := 0; i < voteSet.VoteTypeBitArray.Size(); i++ {
		if VoteType(voteSet.VoteTypeBitArray.GetIndex(i)) == VoteTypeAgree {
			numberOfAgreeVotes++
		}
	}
	return numberOfAgreeVotes
}

func (voteSet *FnVoteSet) GetAgreeVoteIndexForValidatorIndex(validatorIndex int) (int, error) {
	if validatorIndex < 0 || validatorIndex > voteSet.VoteTypeBitArray.Size()-1 {
		return -1, fmt.Errorf("VoteSet: validator index passed is invalid")
	}

	if !voteSet.VoteTypeBitArray.GetIndex(validatorIndex) {
		return -1, fmt.Errorf("VoteSet: validator did not casted agree vote")
	}

	counter := 0
	for i := 0; i < validatorIndex; i++ {
		if VoteType(voteSet.VoteTypeBitArray.GetIndex(i)) == VoteTypeAgree {
			counter++
		}
	}

	return counter, nil
}

func (voteSet *FnVoteSet) IsDisagree(signingThreshold SigningThreshold, currentValidatorSet *types.ValidatorSet) bool {
	switch signingThreshold {
	case Maj23SigningThreshold:
		return voteSet.TotalDisagreeVotingPower >= currentValidatorSet.TotalVotingPower()*2/3+1
	case AllSigningThreshold:
		return voteSet.TotalDisagreeVotingPower == currentValidatorSet.TotalVotingPower()
	default:
		panic("unknown signing threshold")
	}
}

func (voteSet *FnVoteSet) HasConverged(signingThreshold SigningThreshold, currentValidatorSet *types.ValidatorSet) bool {
	return voteSet.IsAgree(signingThreshold, currentValidatorSet) || voteSet.IsDisagree(signingThreshold, currentValidatorSet)
}

func (voteSet *FnVoteSet) HaveWeAlreadySigned(ownValidatorIndex int) bool {
	return voteSet.VoteBitArray.GetIndex(ownValidatorIndex)
}

// Should be the first function to be invoked on vote set received from Peer
func (voteSet *FnVoteSet) IsValid(chainID string, maxContextSize int, checkForExpiration bool, expiresAfter time.Duration, currentValidatorSet *types.ValidatorSet, registry FnRegistry) bool {
	isValid := true

	var calculatedAgreedVotingPower int64
	var calculatedDisagreeVotingPower int64

	// This if conditions are individual as, we want to pass different errors for each
	// condition in future.

	if voteSet.VoteBitArray == nil || voteSet.VoteTypeBitArray == nil {
		isValid = false
		return isValid
	}

	numValidators := voteSet.VoteBitArray.Size()

	if voteSet.Payload == nil {
		isValid = false
		return isValid
	}

	if voteSet.ValidatorAddresses == nil {
		isValid = false
		return isValid
	}

	if voteSet.ProposalInfo == nil {
		isValid = false
		return isValid
	}

	if !voteSet.Payload.IsValid(currentValidatorSet) {
		isValid = false
		return isValid
	}

	if registry.Get(voteSet.GetFnID()) == nil {
		isValid = false
		return isValid
	}

	if voteSet.ChainID != chainID {
		isValid = false
		return isValid
	}

	if checkForExpiration {
		if voteSet.IsExpired(expiresAfter) {
			isValid = false
			return isValid
		}
	}

	if !bytes.Equal(voteSet.ValidatorsHash, currentValidatorSet.Hash()) {
		isValid = false
		return isValid
	}

	if numValidators != len(voteSet.ValidatorAddresses) || numValidators != len(voteSet.ValidatorSignatures) || numValidators != currentValidatorSet.Size() {
		isValid = false
		return isValid
	}

	if numValidators != voteSet.Payload.Response.SignatureBitArray.Size() {
		isValid = false
		return isValid
	}

	if len(voteSet.ExecutionContext) > maxContextSize {
		isValid = false
		return isValid
	}

	currentValidatorSet.Iterate(func(i int, val *types.Validator) bool {
		if !bytes.Equal(voteSet.ValidatorAddresses[i], val.Address) {
			isValid = false
			return true
		}

		if voteSet.VoteBitArray.GetIndex(i) != voteSet.Payload.Response.SignatureBitArray.GetIndex(i) {
			isValid = false
			return true
		}

		if !voteSet.VoteBitArray.GetIndex(i) {
			return false
		}

		voteType := VoteType(voteSet.VoteTypeBitArray.GetIndex(i))

		if (voteType == VoteTypeAgree) != (voteSet.Payload.Response.OracleSignatures[i] != nil) {
			isValid = false
			return true
		}

		if err := voteSet.VerifyValidatorSign(i, voteType, val.PubKey); err != nil {
			isValid = false
			return true
		}

		if voteType == VoteTypeAgree {
			calculatedAgreedVotingPower += val.VotingPower
		} else {
			calculatedDisagreeVotingPower += val.VotingPower
		}
		return false
	})

	// Voting power contained in VoteSet should match the calculated voting power
	if voteSet.TotalAgreeVotingPower != calculatedAgreedVotingPower {
		isValid = false
		return false
	}

	if voteSet.TotalDisagreeVotingPower != calculatedDisagreeVotingPower {
		isValid = false
		return false
	}

	if err := voteSet.VerifyProposerSign(currentValidatorSet); err != nil {
		isValid = false
		return false
	}

	return isValid
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

		anotherSetVoteType := VoteType(anotherSet.VoteTypeBitArray.GetIndex(i))

		if anotherSetVoteType == VoteTypeAgree {
			voteSet.TotalAgreeVotingPower += currentValidator.VotingPower
		} else {
			voteSet.TotalDisagreeVotingPower += currentValidator.VotingPower
		}

		voteSet.VoteTypeBitArray.SetIndex(i, bool(anotherSetVoteType))
	}

	return hasChanged, nil
}

func (voteSet *FnVoteSet) AddVote(nonce int64, individualExecutionResponse *FnIndividualExecutionResponse, currentValidatorSet *types.ValidatorSet, validatorIndex int, voteType VoteType, privValidator types.PrivValidator) error {
	if voteSet.Nonce != nonce {
		return fmt.Errorf("FnConsensusReactor: unable to add vote as nonce is different from voteset")
	}

	if voteSet.VoteBitArray.GetIndex(validatorIndex) {
		return ErrFnVoteAlreadyCast
	}

	if voteType != VoteTypeDisAgree {
		if !voteSet.Payload.Response.CannonicalCompareWithIndividualExecution(individualExecutionResponse) {
			return fmt.Errorf("fnConsensusReactor: unable to add vote as execution responses are different")
		}
	}

	if err := voteSet.Payload.Response.AddSignature(validatorIndex, individualExecutionResponse.OracleSignature); err != nil {
		return fmt.Errorf("fnConsesnusReactor: unable to add vote as can't add signature, Error: %s", err.Error())
	}

	signBytes, err := voteSet.SignBytes(validatorIndex, voteType)
	if err != nil {
		return fmt.Errorf("fnConsensusReactor: unable to add vote as unable to get sign bytes. Error: %s", err.Error())
	}

	signature, err := privValidator.Sign(signBytes)
	if err != nil {
		return fmt.Errorf("fnConsensusReactor: unable to add vote as unable to sign signing bytes. Error: %s", err.Error())
	}

	voteSet.VoteBitArray.SetIndex(validatorIndex, true)
	voteSet.VoteTypeBitArray.SetIndex(validatorIndex, bool(voteType))

	voteSet.ValidatorSignatures[validatorIndex] = signature

	_, validator := currentValidatorSet.GetByIndex(validatorIndex)
	if validator == nil {
		return fmt.Errorf("fnConsensusReactor: unable to add vote as validatorIndex is not valid")
	}

	if !bytes.Equal(validator.Address, voteSet.ValidatorAddresses[validatorIndex]) {
		return fmt.Errorf("fnConsensusReactor: unable to add vote as validatorAddress does not match with one in the vote set")
	}

	if voteType == VoteTypeAgree {
		voteSet.TotalAgreeVotingPower += validator.VotingPower
	} else {
		voteSet.TotalDisagreeVotingPower += validator.VotingPower
	}

	return nil
}

func RegisterFnConsensusTypes() {
	cdc.RegisterConcrete(&FnExecutionRequest{}, "tendermint/fnConsensusReactor/FnExecutionRequest", nil)
	cdc.RegisterConcrete(&FnExecutionResponse{}, "tendermint/fnConsensusReactor/FnExecutionResponse", nil)
	cdc.RegisterConcrete(&FnVoteSet{}, "tendermint/fnConsensusReactor/FnVoteSet", nil)
	cdc.RegisterConcrete(&FnVotePayload{}, "tendermint/fnConsensusReactor/FnVotePayload", nil)
	cdc.RegisterConcrete(&FnIndividualExecutionResponse{}, "tendermint/fnConsensusReactor/FnIndividualExecutionResponse", nil)
	cdc.RegisterConcrete(&ReactorState{}, "tendermint/fnConsensusReactor/ReactorState", nil)
	cdc.RegisterConcrete(&reactorSetMarshallable{}, "tendermint/fnConsensusReactor/reactorSetMarshallable", nil)
	cdc.RegisterConcrete(&fnIDToNonce{}, "tendermint/fnConsensusReactor/fnIDToNonce", nil)
}
