package ai

// RoutingContext contains the context for routing decisions.
type RoutingContext struct {
	// Task type (chat, embedding, image, video)
	taskType TaskType

	// Token estimation
	estimatedTokens int

	// Required capabilities
	requireStream bool
	requireTools  bool
	requireVision bool
	requireJSON   bool

	// Context window requirements
	minContextWindow int

	// Cost constraints
	maxCostPer1K float64

	// Optimization preference (cost, quality, speed)
	optimize string

	// Provider preferences
	preferredProviders []string
	excludedProviders  []string
	preferredModels    []string

	// Health status (injected by routing manager)
	providerHealth map[string]bool

	// Group override
	groupID string

	// Additional metadata
	metadata map[string]any
}

// NewRoutingContext creates a new routing context with defaults.
func NewRoutingContext() *RoutingContext {
	return &RoutingContext{
		taskType:       TaskTypeChat,
		providerHealth: make(map[string]bool),
		metadata:       make(map[string]any),
	}
}

// Getters
func (c *RoutingContext) TaskType() TaskType           { return c.taskType }
func (c *RoutingContext) EstimatedTokens() int         { return c.estimatedTokens }
func (c *RoutingContext) RequireStream() bool          { return c.requireStream }
func (c *RoutingContext) RequireTools() bool           { return c.requireTools }
func (c *RoutingContext) RequireVision() bool          { return c.requireVision }
func (c *RoutingContext) RequireJSON() bool            { return c.requireJSON }
func (c *RoutingContext) MinContextWindow() int        { return c.minContextWindow }
func (c *RoutingContext) MaxCostPer1K() float64        { return c.maxCostPer1K }
func (c *RoutingContext) Optimize() string             { return c.optimize }
func (c *RoutingContext) PreferredProviders() []string { return c.preferredProviders }
func (c *RoutingContext) ExcludedProviders() []string  { return c.excludedProviders }
func (c *RoutingContext) PreferredModels() []string    { return c.preferredModels }
func (c *RoutingContext) ProviderHealth() map[string]bool { return c.providerHealth }
func (c *RoutingContext) GroupID() string              { return c.groupID }
func (c *RoutingContext) Metadata() map[string]any     { return c.metadata }

// Setters
func (c *RoutingContext) SetTaskType(t TaskType)                { c.taskType = t }
func (c *RoutingContext) SetEstimatedTokens(n int)              { c.estimatedTokens = n }
func (c *RoutingContext) SetRequireStream(s bool)               { c.requireStream = s }
func (c *RoutingContext) SetRequireTools(t bool)                { c.requireTools = t }
func (c *RoutingContext) SetRequireVision(v bool)               { c.requireVision = v }
func (c *RoutingContext) SetRequireJSON(j bool)                 { c.requireJSON = j }
func (c *RoutingContext) SetMinContextWindow(w int)             { c.minContextWindow = w }
func (c *RoutingContext) SetMaxCostPer1K(c2 float64)            { c.maxCostPer1K = c2 }
func (c *RoutingContext) SetOptimize(o string)                  { c.optimize = o }
func (c *RoutingContext) SetPreferredProviders(p []string)      { c.preferredProviders = p }
func (c *RoutingContext) SetExcludedProviders(e []string)       { c.excludedProviders = e }
func (c *RoutingContext) SetPreferredModels(m []string)         { c.preferredModels = m }
func (c *RoutingContext) SetProviderHealth(h map[string]bool)   { c.providerHealth = h }
func (c *RoutingContext) SetGroupID(g string)                   { c.groupID = g }
func (c *RoutingContext) SetMetadata(m map[string]any)          { c.metadata = m }
func (c *RoutingContext) AddMetadata(key string, val any)       { c.metadata[key] = val }

// RequiredCapabilities returns the list of required capabilities.
func (c *RoutingContext) RequiredCapabilities() []Capability {
	caps := []Capability{CapabilityChat}

	if c.requireStream {
		caps = append(caps, CapabilityStream)
	}
	if c.requireTools {
		caps = append(caps, CapabilityTools)
	}
	if c.requireVision {
		caps = append(caps, CapabilityVision)
	}
	if c.requireJSON {
		caps = append(caps, CapabilityJSON)
	}

	return caps
}

// ScoredCandidate represents a candidate with scoring information.
type ScoredCandidate struct {
	provider       *Provider
	model          *Model
	score          float64
	scoreBreakdown map[string]float64
	reasons        []string
}

// NewScoredCandidate creates a new scored candidate.
func NewScoredCandidate(p *Provider, m *Model) *ScoredCandidate {
	return &ScoredCandidate{
		provider:       p,
		model:          m,
		score:          0,
		scoreBreakdown: make(map[string]float64),
		reasons:        make([]string, 0),
	}
}

// Getters
func (c *ScoredCandidate) Provider() *Provider             { return c.provider }
func (c *ScoredCandidate) Model() *Model                   { return c.model }
func (c *ScoredCandidate) Score() float64                  { return c.score }
func (c *ScoredCandidate) ScoreBreakdown() map[string]float64 { return c.scoreBreakdown }
func (c *ScoredCandidate) Reasons() []string               { return c.reasons }

// AddScore adds a score from a strategy.
func (c *ScoredCandidate) AddScore(strategy string, score float64, reason string) {
	c.score += score
	c.scoreBreakdown[strategy] = score
	if reason != "" {
		c.reasons = append(c.reasons, reason)
	}
}

// RoutingResult represents the routing result.
type RoutingResult struct {
	provider  *Provider
	model     *Model
	score     float64
	reason    string
	accountID *string // Provider account ID if using pool
	apiKey    string  // Decrypted API key (from pool or provider)
}

// NewRoutingResult creates a new routing result.
func NewRoutingResult(provider *Provider, model *Model, score float64, reason string) *RoutingResult {
	return &RoutingResult{
		provider: provider,
		model:    model,
		score:    score,
		reason:   reason,
	}
}

// Getters
func (r *RoutingResult) Provider() *Provider { return r.provider }
func (r *RoutingResult) Model() *Model       { return r.model }
func (r *RoutingResult) Score() float64      { return r.score }
func (r *RoutingResult) Reason() string      { return r.reason }
func (r *RoutingResult) AccountID() *string  { return r.accountID }
func (r *RoutingResult) APIKey() string      { return r.apiKey }

// Setters
func (r *RoutingResult) SetAccountID(id *string) { r.accountID = id }
func (r *RoutingResult) SetAPIKey(key string)    { r.apiKey = key }

// RoutingStrategy defines the interface for routing strategies.
type RoutingStrategy interface {
	// Name returns the strategy name.
	Name() string

	// Priority returns the priority (higher = runs first).
	Priority() int

	// Filter filters candidates.
	Filter(ctx *RoutingContext, candidates []*ScoredCandidate) []*ScoredCandidate

	// Score scores candidates.
	Score(ctx *RoutingContext, candidates []*ScoredCandidate) []*ScoredCandidate
}
