package gateway

import (
	"fmt"
	"time"

	"github.com/go-kit/kit/metrics"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

type Metrics interface {
	MethodCalled(begin time.Time, method string, err error)
	FetchedMainnetEvents(numEvents int, kind string)
	SubmittedMainnetEvents(numEvents int)
	WithdrawalsSigned(numWithdrawals int)
	ContractCreatorsVerified(numCreators int)
}

type prometheusMetrics struct {
	methodCallCount              metrics.Counter
	methodDuration               metrics.Histogram
	fetchedMainnetEventCount     metrics.Counter
	submittedMainnetEventCount   metrics.Counter
	signedWithdrawalCount        metrics.Counter
	verifiedContractCreatorCount metrics.Counter
}

var _ Metrics = (*prometheusMetrics)(nil)

func NewMetrics(subsystem string) Metrics {
	const namespace = "loomchain"

	return &prometheusMetrics{
		methodCallCount: kitprometheus.NewCounterFrom(
			stdprometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "method_call_count",
				Help:      "Number of times a method has been invoked.",
			}, []string{"method", "error"}),
		methodDuration: kitprometheus.NewSummaryFrom(
			stdprometheus.SummaryOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "method_duration",
				Help:      "How long a method took to execute (in seconds).",
			}, []string{"method", "error"}),
		fetchedMainnetEventCount: kitprometheus.NewCounterFrom(
			stdprometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "fetched_mainnet_event_count",
				Help:      "Number of Mainnet events fetched from the Mainnet Gateway.",
			}, []string{"kind"}),
		submittedMainnetEventCount: kitprometheus.NewCounterFrom(
			stdprometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "submitted_mainnet_event_count",
				Help:      "Number of Mainnet events successfully submitted to the DAppChain Gateway.",
			}, nil),
		signedWithdrawalCount: kitprometheus.NewCounterFrom(
			stdprometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "signed_withdrawal_count",
				Help:      "Number of withdrawals signed.",
			}, nil),
		verifiedContractCreatorCount: kitprometheus.NewCounterFrom(
			stdprometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "verified_contract_creator_count",
				Help:      "Number of contract creator verifications performed.",
			}, nil),
	}
}

func (m *prometheusMetrics) MethodCalled(begin time.Time, method string, err error) {
	lvs := []string{"method", method, "error", fmt.Sprint(err != nil)}
	m.methodDuration.With(lvs...).Observe(time.Since(begin).Seconds())
	m.methodCallCount.With(lvs...).Add(1)
}

func (m *prometheusMetrics) FetchedMainnetEvents(numEvents int, kind string) {
	m.fetchedMainnetEventCount.With("kind", kind).Add(float64(numEvents))
}

func (m *prometheusMetrics) SubmittedMainnetEvents(numEvents int) {
	m.submittedMainnetEventCount.Add(float64(numEvents))
}

func (m *prometheusMetrics) WithdrawalsSigned(numWithdrawals int) {
	m.signedWithdrawalCount.Add(float64(numWithdrawals))
}

func (m *prometheusMetrics) ContractCreatorsVerified(numCreators int) {
	m.verifiedContractCreatorCount.Add(float64(numCreators))
}

type noopMetrics struct{}

var _ Metrics = (*noopMetrics)(nil)

// NewNoopMetrics creates a metrics collector that doesn't collect anything, useful for tests.
func NewNoopMetrics() Metrics {
	return &noopMetrics{}
}

func (m *noopMetrics) MethodCalled(begin time.Time, method string, err error) {
}

func (m *noopMetrics) FetchedMainnetEvents(numEvents int, kind string) {
}

func (m *noopMetrics) SubmittedMainnetEvents(numEvents int) {
}

func (m *noopMetrics) WithdrawalsSigned(numWithdrawals int) {
}

func (m *noopMetrics) ContractCreatorsVerified(numCreators int) {
}
