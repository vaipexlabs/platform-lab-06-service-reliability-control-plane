// Package contract loads and validates service reliability contracts.
package contract

// ServiceReliability is the typed v1alpha1 reliability contract used for
// semantic validation after the canonical JSON Schema has accepted a document.
type ServiceReliability struct {
	APIVersion string   `json:"apiVersion" yaml:"apiVersion"`
	Kind       string   `json:"kind" yaml:"kind"`
	Metadata   Metadata `json:"metadata" yaml:"metadata"`
	Spec       Spec     `json:"spec" yaml:"spec"`
}

// Metadata identifies one service reliability contract.
type Metadata struct {
	Name   string            `json:"name" yaml:"name"`
	Labels map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
}

// Spec contains service identity, evaluation behavior, objectives, and the
// policy used to form delivery recommendations.
type Spec struct {
	Service        Service        `json:"service" yaml:"service"`
	Evaluation     Evaluation     `json:"evaluation" yaml:"evaluation"`
	Objectives     []Objective    `json:"objectives" yaml:"objectives"`
	DecisionPolicy DecisionPolicy `json:"decisionPolicy" yaml:"decisionPolicy"`
}

// Service describes operational ownership and discovery metadata.
type Service struct {
	DisplayName string `json:"displayName" yaml:"displayName"`
	Owner       Owner  `json:"owner" yaml:"owner"`
	Tier        string `json:"tier" yaml:"tier"`
	Repository  string `json:"repository" yaml:"repository"`
	Runbook     string `json:"runbook" yaml:"runbook"`
}

// Owner identifies the accountable team rather than an individual.
type Owner struct {
	Team    string `json:"team" yaml:"team"`
	Contact string `json:"contact" yaml:"contact"`
}

// Evaluation defines how often evidence is assessed and how old evidence may
// be before the result becomes unknown.
type Evaluation struct {
	Interval      string `json:"interval" yaml:"interval"`
	DataFreshness string `json:"dataFreshness" yaml:"dataFreshness"`
}

// Objective declares an SLI, SLO target, window, and burn-rate policies.
type Objective struct {
	ID                 string          `json:"id" yaml:"id"`
	DisplayName        string          `json:"displayName" yaml:"displayName"`
	Description        string          `json:"description" yaml:"description"`
	Indicator          Indicator       `json:"indicator" yaml:"indicator"`
	Target             float64         `json:"target" yaml:"target"`
	Window             string          `json:"window" yaml:"window"`
	MinimumTotalEvents *int64          `json:"minimumTotalEvents,omitempty" yaml:"minimumTotalEvents,omitempty"`
	BurnRateAlerts     []BurnRateAlert `json:"burnRateAlerts" yaml:"burnRateAlerts"`
}

// Indicator identifies the adapter and queries that return comparable good
// and total event values.
type Indicator struct {
	Category         string `json:"category" yaml:"category"`
	Provider         string `json:"provider" yaml:"provider"`
	QueryLanguage    string `json:"queryLanguage" yaml:"queryLanguage"`
	GoodEventsQuery  string `json:"goodEventsQuery" yaml:"goodEventsQuery"`
	TotalEventsQuery string `json:"totalEventsQuery" yaml:"totalEventsQuery"`
}

// BurnRateAlert combines short and long windows to detect meaningful budget
// consumption while reducing sensitivity to isolated spikes.
type BurnRateAlert struct {
	Name        string  `json:"name" yaml:"name"`
	Severity    string  `json:"severity" yaml:"severity"`
	ShortWindow string  `json:"shortWindow" yaml:"shortWindow"`
	LongWindow  string  `json:"longWindow" yaml:"longWindow"`
	Threshold   float64 `json:"threshold" yaml:"threshold"`
}

// DecisionPolicy translates reliability evidence into delivery guidance.
type DecisionPolicy struct {
	Caution CautionPolicy `json:"caution" yaml:"caution"`
	Pause   PausePolicy   `json:"pause" yaml:"pause"`
}

// CautionPolicy defines when remaining budget requires additional review.
type CautionPolicy struct {
	RemainingBudgetBelowPercent float64 `json:"remainingBudgetBelowPercent" yaml:"remainingBudgetBelowPercent"`
}

// PausePolicy defines the stronger conditions that recommend pausing delivery.
type PausePolicy struct {
	RemainingBudgetAtOrBelowPercent float64  `json:"remainingBudgetAtOrBelowPercent" yaml:"remainingBudgetAtOrBelowPercent"`
	ActiveBurnRateSeverities        []string `json:"activeBurnRateSeverities" yaml:"activeBurnRateSeverities"`
}
