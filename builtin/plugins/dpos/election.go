package main

import (
	"sort"

	loom "github.com/loomnetwork/go-loom"
)

type FullVote struct {
	CandidateAddress loom.Address
	VoteSize         uint64
	Power            uint64
}

type VoteResult struct {
	CandidateAddress loom.Address
	VoteTotal        uint64
	PowerTotal       uint64
}

type byPower []*VoteResult

func (s byPower) Len() int {
	return len(s)
}

func (s byPower) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s byPower) Less(i, j int) bool {
	return s[i].PowerTotal < s[j].PowerTotal
}

func runElection(votes []*FullVote) ([]*VoteResult, error) {
	resultSet := make(map[string]*VoteResult)

	for _, vote := range votes {
		key := vote.CandidateAddress.String()
		res := resultSet[key]
		if res == nil {
			res = &VoteResult{
				CandidateAddress: vote.CandidateAddress,
			}
			resultSet[key] = res
		}

		res.VoteTotal += vote.VoteSize
		res.PowerTotal += vote.Power
	}

	results := make([]*VoteResult, 0, len(resultSet))
	for _, res := range resultSet {
		results = append(results, res)
	}

	sort.Sort(sort.Reverse(byPower(results)))
	return results, nil
}
