package experiments

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"go.uber.org/zap"
)

// Service handles feature flags and A/B experiments
type Service struct {
	repo      RepositoryInterface
	flagCache map[string]*FeatureFlag
	cacheMu   sync.RWMutex
	cacheAt   time.Time
	cacheTTL  time.Duration
}

// NewService creates a new experiments service
func NewService(repo RepositoryInterface) *Service {
	return &Service{
		repo:      repo,
		flagCache: make(map[string]*FeatureFlag),
		cacheTTL:  30 * time.Second,
	}
}

// ========================================
// FEATURE FLAG EVALUATION
// ========================================

// IsEnabled checks if a feature flag is enabled for a given user context
func (s *Service) IsEnabled(ctx context.Context, flagKey string, userCtx *UserContext) (bool, error) {
	result, err := s.EvaluateFlag(ctx, flagKey, userCtx)
	if err != nil {
		return false, err
	}
	return result.Enabled, nil
}

// EvaluateFlag evaluates a feature flag for a user
func (s *Service) EvaluateFlag(ctx context.Context, flagKey string, userCtx *UserContext) (*EvaluateFlagResponse, error) {
	flag, err := s.getCachedFlag(ctx, flagKey)
	if err != nil {
		return nil, err
	}
	if flag == nil {
		return &EvaluateFlagResponse{
			Key:     flagKey,
			Enabled: false,
			Source:  "not_found",
		}, nil
	}

	// Inactive flags are always off
	if flag.Status != FlagStatusActive {
		return &EvaluateFlagResponse{
			Key:     flagKey,
			Enabled: false,
			Source:  "inactive",
		}, nil
	}

	// Check per-user override first
	if userCtx != nil {
		override, err := s.repo.GetOverride(ctx, flag.ID, userCtx.UserID)
		if err == nil && override != nil {
			return &EvaluateFlagResponse{
				Key:     flagKey,
				Enabled: override.Enabled,
				Source:  "override",
			}, nil
		}

		// Check blocked users
		for _, blockedID := range flag.BlockedUserIDs {
			if blockedID == userCtx.UserID {
				return &EvaluateFlagResponse{
					Key:     flagKey,
					Enabled: false,
					Source:  "blocked",
				}, nil
			}
		}

		// Check allowed users
		for _, allowedID := range flag.AllowedUserIDs {
			if allowedID == userCtx.UserID {
				return &EvaluateFlagResponse{
					Key:     flagKey,
					Enabled: true,
					Source:  "user_list",
				}, nil
			}
		}
	}

	// Evaluate based on flag type
	switch flag.FlagType {
	case FlagTypeBoolean:
		return &EvaluateFlagResponse{
			Key:     flagKey,
			Enabled: flag.Enabled,
			Source:  "default",
		}, nil

	case FlagTypePercentage:
		if userCtx == nil {
			return &EvaluateFlagResponse{
				Key:     flagKey,
				Enabled: false,
				Source:  "no_context",
			}, nil
		}
		enabled := s.isInPercentage(flagKey, userCtx.UserID, flag.RolloutPercentage)
		return &EvaluateFlagResponse{
			Key:     flagKey,
			Enabled: enabled,
			Source:  "percentage",
		}, nil

	case FlagTypeUserList:
		// Already checked above, so this user is not in the list
		return &EvaluateFlagResponse{
			Key:     flagKey,
			Enabled: false,
			Source:  "user_list",
		}, nil

	case FlagTypeSegment:
		if userCtx == nil || flag.SegmentRules == nil {
			return &EvaluateFlagResponse{
				Key:     flagKey,
				Enabled: flag.Enabled,
				Source:  "default",
			}, nil
		}
		matches := s.matchesSegment(userCtx, flag.SegmentRules)
		return &EvaluateFlagResponse{
			Key:     flagKey,
			Enabled: matches,
			Source:  "segment",
		}, nil
	}

	return &EvaluateFlagResponse{
		Key:     flagKey,
		Enabled: flag.Enabled,
		Source:  "default",
	}, nil
}

// EvaluateFlags evaluates multiple flags at once
func (s *Service) EvaluateFlags(ctx context.Context, flagKeys []string, userCtx *UserContext) (*EvaluateFlagsResponse, error) {
	result := &EvaluateFlagsResponse{
		Flags: make(map[string]EvaluateFlagResponse),
	}

	for _, key := range flagKeys {
		eval, err := s.EvaluateFlag(ctx, key, userCtx)
		if err != nil {
			continue
		}
		result.Flags[key] = *eval
	}

	return result, nil
}

// ========================================
// FEATURE FLAG MANAGEMENT
// ========================================

// CreateFlag creates a new feature flag
func (s *Service) CreateFlag(ctx context.Context, adminID uuid.UUID, req *CreateFlagRequest) (*FeatureFlag, error) {
	// Check for duplicate key
	existing, _ := s.repo.GetFlagByKey(ctx, req.Key)
	if existing != nil {
		return nil, common.NewBadRequestError("flag key already exists", nil)
	}

	now := time.Now()
	flag := &FeatureFlag{
		ID:                uuid.New(),
		Key:               req.Key,
		Name:              req.Name,
		Description:       req.Description,
		FlagType:          req.FlagType,
		Status:            FlagStatusActive,
		Enabled:           req.Enabled,
		RolloutPercentage: req.RolloutPercentage,
		SegmentRules:      req.SegmentRules,
		Tags:              req.Tags,
		CreatedBy:         adminID,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := s.repo.CreateFlag(ctx, flag); err != nil {
		return nil, common.NewInternalServerError("failed to create flag")
	}

	s.invalidateCache()

	logger.Info("Feature flag created",
		zap.String("key", flag.Key),
		zap.String("type", string(flag.FlagType)),
	)

	return flag, nil
}

// UpdateFlag updates a feature flag
func (s *Service) UpdateFlag(ctx context.Context, flagID uuid.UUID, req *UpdateFlagRequest) error {
	flag, err := s.repo.GetFlagByID(ctx, flagID)
	if err != nil || flag == nil {
		return common.NewNotFoundError("flag not found", err)
	}

	if req.Name != nil {
		flag.Name = *req.Name
	}
	if req.Description != nil {
		flag.Description = *req.Description
	}
	if req.Enabled != nil {
		flag.Enabled = *req.Enabled
	}
	if req.RolloutPercentage != nil {
		flag.RolloutPercentage = *req.RolloutPercentage
	}
	if req.SegmentRules != nil {
		flag.SegmentRules = req.SegmentRules
	}
	if req.Tags != nil {
		flag.Tags = req.Tags
	}

	flag.UpdatedAt = time.Now()

	if err := s.repo.UpdateFlag(ctx, flag); err != nil {
		return common.NewInternalServerError("failed to update flag")
	}

	s.invalidateCache()

	logger.Info("Feature flag updated", zap.String("key", flag.Key))
	return nil
}

// ToggleFlag quickly enables or disables a flag
func (s *Service) ToggleFlag(ctx context.Context, flagID uuid.UUID, enabled bool) error {
	flag, err := s.repo.GetFlagByID(ctx, flagID)
	if err != nil || flag == nil {
		return common.NewNotFoundError("flag not found", err)
	}

	flag.Enabled = enabled
	flag.UpdatedAt = time.Now()

	if err := s.repo.UpdateFlag(ctx, flag); err != nil {
		return common.NewInternalServerError("failed to toggle flag")
	}

	s.invalidateCache()

	logger.Info("Feature flag toggled",
		zap.String("key", flag.Key),
		zap.Bool("enabled", enabled),
	)
	return nil
}

// ArchiveFlag archives a feature flag
func (s *Service) ArchiveFlag(ctx context.Context, flagID uuid.UUID) error {
	if err := s.repo.UpdateFlagStatus(ctx, flagID, FlagStatusArchived); err != nil {
		return common.NewInternalServerError("failed to archive flag")
	}
	s.invalidateCache()
	return nil
}

// ListFlags lists feature flags
func (s *Service) ListFlags(ctx context.Context, status *FlagStatus, limit, offset int) ([]*FeatureFlag, error) {
	if limit == 0 {
		limit = 50
	}
	return s.repo.ListFlags(ctx, status, limit, offset)
}

// GetFlag gets a single flag by key
func (s *Service) GetFlag(ctx context.Context, key string) (*FeatureFlag, error) {
	return s.repo.GetFlagByKey(ctx, key)
}

// ========================================
// FLAG OVERRIDES
// ========================================

// CreateOverride creates a per-user flag override
func (s *Service) CreateOverride(ctx context.Context, adminID uuid.UUID, flagID uuid.UUID, req *CreateOverrideRequest) error {
	flag, err := s.repo.GetFlagByID(ctx, flagID)
	if err != nil || flag == nil {
		return common.NewNotFoundError("flag not found", err)
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			return common.NewBadRequestError("invalid expires_at format", err)
		}
		expiresAt = &t
	}

	override := &FlagOverride{
		ID:        uuid.New(),
		FlagID:    flagID,
		UserID:    req.UserID,
		Enabled:   req.Enabled,
		Reason:    req.Reason,
		ExpiresAt: expiresAt,
		CreatedBy: adminID,
		CreatedAt: time.Now(),
	}

	if err := s.repo.CreateOverride(ctx, override); err != nil {
		return common.NewInternalServerError("failed to create override")
	}

	logger.Info("Flag override created",
		zap.String("flag_key", flag.Key),
		zap.String("user_id", req.UserID.String()),
		zap.Bool("enabled", req.Enabled),
	)

	return nil
}

// ListOverrides lists overrides for a flag
func (s *Service) ListOverrides(ctx context.Context, flagID uuid.UUID) ([]*FlagOverride, error) {
	return s.repo.ListOverrides(ctx, flagID)
}

// DeleteOverride removes a flag override
func (s *Service) DeleteOverride(ctx context.Context, flagID, userID uuid.UUID) error {
	return s.repo.DeleteOverride(ctx, flagID, userID)
}

// ========================================
// A/B EXPERIMENT MANAGEMENT
// ========================================

// CreateExperiment creates a new A/B experiment
func (s *Service) CreateExperiment(ctx context.Context, adminID uuid.UUID, req *CreateExperimentRequest) (*Experiment, error) {
	// Check for duplicate key
	existing, _ := s.repo.GetExperimentByKey(ctx, req.Key)
	if existing != nil {
		return nil, common.NewBadRequestError("experiment key already exists", nil)
	}

	// Validate variant weights sum to 100
	totalWeight := 0
	hasControl := false
	for _, v := range req.Variants {
		totalWeight += v.Weight
		if v.IsControl {
			hasControl = true
		}
	}
	if totalWeight != 100 {
		return nil, common.NewBadRequestError("variant weights must sum to 100", nil)
	}
	if !hasControl {
		return nil, common.NewBadRequestError("at least one variant must be marked as control", nil)
	}

	// Set defaults
	confidenceLevel := req.ConfidenceLevel
	if confidenceLevel == 0 {
		confidenceLevel = 0.95
	}
	minSampleSize := req.MinSampleSize
	if minSampleSize == 0 {
		minSampleSize = 100
	}

	now := time.Now()
	experiment := &Experiment{
		ID:                uuid.New(),
		Key:               req.Key,
		Name:              req.Name,
		Description:       req.Description,
		Hypothesis:        req.Hypothesis,
		Status:            ExperimentStatusDraft,
		TrafficPercentage: req.TrafficPercentage,
		SegmentRules:      req.SegmentRules,
		PrimaryMetric:     req.PrimaryMetric,
		SecondaryMetrics:  req.SecondaryMetrics,
		MinSampleSize:     minSampleSize,
		ConfidenceLevel:   confidenceLevel,
		CreatedBy:         adminID,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	var variants []*Variant
	for _, v := range req.Variants {
		variant := &Variant{
			ID:           uuid.New(),
			ExperimentID: experiment.ID,
			Key:          v.Key,
			Name:         v.Name,
			Description:  v.Description,
			IsControl:    v.IsControl,
			Weight:       v.Weight,
			Config:       v.Config,
			CreatedAt:    now,
		}
		variants = append(variants, variant)
	}

	if err := s.repo.CreateExperiment(ctx, experiment, variants); err != nil {
		return nil, common.NewInternalServerError("failed to create experiment")
	}

	logger.Info("Experiment created",
		zap.String("key", experiment.Key),
		zap.Int("variants", len(variants)),
	)

	return experiment, nil
}

// StartExperiment begins running an experiment
func (s *Service) StartExperiment(ctx context.Context, experimentID uuid.UUID) error {
	exp, err := s.repo.GetExperimentByID(ctx, experimentID)
	if err != nil || exp == nil {
		return common.NewNotFoundError("experiment not found", err)
	}

	if exp.Status != ExperimentStatusDraft && exp.Status != ExperimentStatusPaused {
		return common.NewBadRequestError("experiment cannot be started from current status", nil)
	}

	if err := s.repo.UpdateExperimentStatus(ctx, experimentID, ExperimentStatusRunning); err != nil {
		return common.NewInternalServerError("failed to start experiment")
	}

	logger.Info("Experiment started", zap.String("key", exp.Key))
	return nil
}

// PauseExperiment pauses a running experiment
func (s *Service) PauseExperiment(ctx context.Context, experimentID uuid.UUID) error {
	if err := s.repo.UpdateExperimentStatus(ctx, experimentID, ExperimentStatusPaused); err != nil {
		return common.NewInternalServerError("failed to pause experiment")
	}
	return nil
}

// ConcludeExperiment ends an experiment
func (s *Service) ConcludeExperiment(ctx context.Context, experimentID uuid.UUID) error {
	if err := s.repo.UpdateExperimentStatus(ctx, experimentID, ExperimentStatusCompleted); err != nil {
		return common.NewInternalServerError("failed to conclude experiment")
	}
	return nil
}

// ListExperiments lists experiments
func (s *Service) ListExperiments(ctx context.Context, status *ExperimentStatus, limit, offset int) ([]*Experiment, error) {
	if limit == 0 {
		limit = 20
	}
	return s.repo.ListExperiments(ctx, status, limit, offset)
}

// GetExperiment gets an experiment with its variants
func (s *Service) GetExperiment(ctx context.Context, experimentID uuid.UUID) (*Experiment, []*Variant, error) {
	exp, err := s.repo.GetExperimentByID(ctx, experimentID)
	if err != nil || exp == nil {
		return nil, nil, common.NewNotFoundError("experiment not found", err)
	}

	variants, err := s.repo.GetVariants(ctx, experimentID)
	if err != nil {
		return nil, nil, common.NewInternalServerError("failed to get variants")
	}

	return exp, variants, nil
}

// ========================================
// EXPERIMENT ASSIGNMENT & TRACKING
// ========================================

// GetVariantForUser assigns and returns a variant for a user in an experiment
func (s *Service) GetVariantForUser(ctx context.Context, experimentKey string, userCtx *UserContext) (*Variant, error) {
	exp, err := s.repo.GetExperimentByKey(ctx, experimentKey)
	if err != nil || exp == nil {
		return nil, nil // Experiment not found, treat as not enrolled
	}

	if exp.Status != ExperimentStatusRunning {
		return nil, nil
	}

	// Check if user is in traffic percentage
	if !s.isInPercentage(experimentKey, userCtx.UserID, exp.TrafficPercentage) {
		return nil, nil
	}

	// Check segment rules
	if exp.SegmentRules != nil && !s.matchesSegment(userCtx, exp.SegmentRules) {
		return nil, nil
	}

	// Check existing assignment
	existing, _ := s.repo.GetAssignment(ctx, exp.ID, userCtx.UserID)
	if existing != nil {
		variants, _ := s.repo.GetVariants(ctx, exp.ID)
		for _, v := range variants {
			if v.ID == existing.VariantID {
				return v, nil
			}
		}
	}

	// Assign to a variant
	variants, err := s.repo.GetVariants(ctx, exp.ID)
	if err != nil || len(variants) == 0 {
		return nil, nil
	}

	variant := s.assignVariant(experimentKey, userCtx.UserID, variants)

	assignment := &ExperimentAssignment{
		ID:           uuid.New(),
		ExperimentID: exp.ID,
		UserID:       userCtx.UserID,
		VariantID:    variant.ID,
		AssignedAt:   time.Now(),
	}

	_ = s.repo.CreateAssignment(ctx, assignment)

	return variant, nil
}

// TrackEvent records an experiment event for a user
func (s *Service) TrackEvent(ctx context.Context, userID uuid.UUID, req *TrackEventRequest) error {
	exp, err := s.repo.GetExperimentByKey(ctx, req.ExperimentKey)
	if err != nil || exp == nil {
		return nil // Silently ignore unknown experiments
	}

	// Find the user's assignment
	assignment, err := s.repo.GetAssignment(ctx, exp.ID, userID)
	if err != nil || assignment == nil {
		return nil // User not assigned to this experiment
	}

	event := &ExperimentEvent{
		ID:           uuid.New(),
		ExperimentID: exp.ID,
		UserID:       userID,
		VariantID:    assignment.VariantID,
		EventType:    req.EventType,
		EventValue:   req.EventValue,
		Metadata:     req.Metadata,
		CreatedAt:    time.Now(),
	}

	return s.repo.RecordEvent(ctx, event)
}

// GetResults calculates experiment results
func (s *Service) GetResults(ctx context.Context, experimentID uuid.UUID) (*ExperimentResults, error) {
	exp, err := s.repo.GetExperimentByID(ctx, experimentID)
	if err != nil || exp == nil {
		return nil, common.NewNotFoundError("experiment not found", err)
	}

	metrics, err := s.repo.GetVariantMetrics(ctx, experimentID)
	if err != nil {
		return nil, common.NewInternalServerError("failed to get metrics")
	}

	results := &ExperimentResults{
		Experiment: exp,
		Variants:   metrics,
	}

	// Analyze results
	s.analyzeResults(results, exp.MinSampleSize, exp.ConfidenceLevel)

	return results, nil
}

// ========================================
// INTERNAL HELPERS
// ========================================

// isInPercentage uses consistent hashing to determine if a user falls within a rollout percentage
func (s *Service) isInPercentage(key string, userID uuid.UUID, percentage int) bool {
	if percentage >= 100 {
		return true
	}
	if percentage <= 0 {
		return false
	}

	// Create deterministic hash from flag key + user ID
	h := sha256.Sum256([]byte(key + ":" + userID.String()))
	bucket := int(binary.BigEndian.Uint32(h[:4]) % 100)

	return bucket < percentage
}

// matchesSegment checks if a user matches the segment rules
func (s *Service) matchesSegment(userCtx *UserContext, rules *SegmentRules) bool {
	// Role check
	if len(rules.Roles) > 0 {
		found := false
		for _, r := range rules.Roles {
			if r == userCtx.Role {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Country check
	if len(rules.Countries) > 0 {
		found := false
		for _, c := range rules.Countries {
			if c == userCtx.Country {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// City check
	if len(rules.Cities) > 0 {
		found := false
		for _, c := range rules.Cities {
			if c == userCtx.City {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Ride count checks
	if rules.MinRides != nil && userCtx.TotalRides < *rules.MinRides {
		return false
	}
	if rules.MaxRides != nil && userCtx.TotalRides > *rules.MaxRides {
		return false
	}

	// Rating check
	if rules.MinRating != nil && userCtx.Rating < *rules.MinRating {
		return false
	}

	// Account age check
	if rules.AccountAge != nil && userCtx.AccountAge < *rules.AccountAge {
		return false
	}

	// Platform check
	if len(rules.Platform) > 0 {
		found := false
		for _, p := range rules.Platform {
			if p == userCtx.Platform {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Loyalty tier check
	if len(rules.LoyaltyTier) > 0 {
		found := false
		for _, t := range rules.LoyaltyTier {
			if t == userCtx.LoyaltyTier {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// assignVariant assigns a user to a variant based on weighted distribution
func (s *Service) assignVariant(key string, userID uuid.UUID, variants []*Variant) *Variant {
	h := sha256.Sum256([]byte(key + ":variant:" + userID.String()))
	bucket := int(binary.BigEndian.Uint32(h[:4]) % 100)

	// Walk through variants by weight
	cumulative := 0
	for _, v := range variants {
		cumulative += v.Weight
		if bucket < cumulative {
			return v
		}
	}

	// Fallback to last variant
	return variants[len(variants)-1]
}

// analyzeResults performs statistical analysis on experiment results
func (s *Service) analyzeResults(results *ExperimentResults, minSampleSize int, confidenceLevel float64) {
	if len(results.Variants) < 2 {
		results.RecommendedAction = "insufficient_variants"
		return
	}

	// Check if we have enough samples
	totalSamples := 0
	for _, v := range results.Variants {
		totalSamples += v.SampleSize
	}

	results.CanConclude = true
	for _, v := range results.Variants {
		if v.SampleSize < minSampleSize {
			results.CanConclude = false
			break
		}
	}

	// Find control and best variant
	var control *VariantMetrics
	var bestVariant *VariantMetrics
	bestRate := -1.0

	for i := range results.Variants {
		v := &results.Variants[i]
		if v.VariantKey == "control" || i == 0 {
			if control == nil {
				control = v
			}
		}
		if v.ConversionRate > bestRate {
			bestRate = v.ConversionRate
			bestVariant = v
		}
	}

	if control == nil || bestVariant == nil {
		results.RecommendedAction = "continue"
		return
	}

	// Calculate uplift
	if control.ConversionRate > 0 {
		uplift := (bestVariant.ConversionRate - control.ConversionRate) / control.ConversionRate * 100
		results.Uplift = &uplift
	}

	// Simplified significance test (Z-test for proportions)
	if control.SampleSize > 0 && bestVariant.SampleSize > 0 {
		p1 := control.ConversionRate
		p2 := bestVariant.ConversionRate
		n1 := float64(control.SampleSize)
		n2 := float64(bestVariant.SampleSize)

		pooledP := (p1*n1 + p2*n2) / (n1 + n2)
		if pooledP > 0 && pooledP < 1 {
			se := math.Sqrt(pooledP * (1 - pooledP) * (1/n1 + 1/n2))
			if se > 0 {
				z := math.Abs(p2-p1) / se

				// Z critical values for common confidence levels
				var zCritical float64
				switch {
				case confidenceLevel >= 0.99:
					zCritical = 2.576
				case confidenceLevel >= 0.95:
					zCritical = 1.96
				default:
					zCritical = 1.645
				}

				results.IsSignificant = z > zCritical

				// Approximate p-value from z-score (simplified)
				pValue := 2 * (1 - normalCDF(z))
				results.PValue = &pValue
			}
		}
	}

	// Determine recommendation
	if !results.CanConclude {
		results.RecommendedAction = "continue"
	} else if results.IsSignificant && bestVariant.VariantKey != "control" {
		results.Winner = &bestVariant.VariantKey
		results.RecommendedAction = "conclude_winner"
	} else if results.IsSignificant {
		results.RecommendedAction = "conclude_no_improvement"
	} else {
		results.RecommendedAction = "continue"
	}
}

// getCachedFlag retrieves a flag from cache or database
func (s *Service) getCachedFlag(ctx context.Context, key string) (*FeatureFlag, error) {
	s.cacheMu.RLock()
	if time.Since(s.cacheAt) < s.cacheTTL {
		flag, ok := s.flagCache[key]
		s.cacheMu.RUnlock()
		if ok {
			return flag, nil
		}
		// Key not in cache; fall through to DB
		return s.repo.GetFlagByKey(ctx, key)
	}
	s.cacheMu.RUnlock()

	// Refresh cache
	if err := s.refreshCache(ctx); err != nil {
		// Fallback to direct DB query
		return s.repo.GetFlagByKey(ctx, key)
	}

	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()
	flag, ok := s.flagCache[key]
	if !ok {
		return nil, nil
	}
	return flag, nil
}

// refreshCache reloads all active flags from the database
func (s *Service) refreshCache(ctx context.Context) error {
	flags, err := s.repo.GetAllActiveFlags(ctx)
	if err != nil {
		return err
	}

	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	s.flagCache = make(map[string]*FeatureFlag, len(flags))
	for _, f := range flags {
		s.flagCache[f.Key] = f
	}
	s.cacheAt = time.Now()

	return nil
}

// invalidateCache forces a cache refresh on next access
func (s *Service) invalidateCache() {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	s.cacheAt = time.Time{} // Zero time forces refresh
}

// normalCDF approximates the standard normal CDF
func normalCDF(x float64) float64 {
	return 0.5 * (1 + math.Erf(x/math.Sqrt(2)))
}
