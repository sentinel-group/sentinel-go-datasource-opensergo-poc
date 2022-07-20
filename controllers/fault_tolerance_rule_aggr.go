package controllers

import (
	"github.com/alibaba/sentinel-golang/core/flow"
	"github.com/sentinel-group/sentinel-go-datasource-opensergo-poc/api/v1alpha1"

	"reflect"
	"sync"
)

type FaultToleranceRuleAggregator struct {
	ruleMap              map[string]*v1alpha1.FaultToleranceRule
	rateLimitStrategyMap map[string]*v1alpha1.RateLimitStrategy

	updateMux sync.RWMutex
}

func (r *FaultToleranceRuleAggregator) getRule(name string) (*v1alpha1.FaultToleranceRule, bool) {
	r.updateMux.RLock()
	defer r.updateMux.RUnlock()

	rule, hasIt := r.ruleMap[name]
	return rule, hasIt
}

func (r *FaultToleranceRuleAggregator) getRateLimitStrategy(name string) (*v1alpha1.RateLimitStrategy, bool) {
	r.updateMux.RLock()
	defer r.updateMux.RUnlock()

	rule, hasIt := r.rateLimitStrategyMap[name]
	return rule, hasIt
}

func (r *FaultToleranceRuleAggregator) updateRule(name string, newRule *v1alpha1.FaultToleranceRule) (bool, error) {
	r.updateMux.Lock()
	defer r.updateMux.Unlock()

	oldRule, hasIt := r.ruleMap[name]
	if hasIt && reflect.DeepEqual(oldRule, newRule) {
		// The updated rule remains unchanged
		return false, nil
	}
	// Update ruleMap (TODO: resolve consistency)
	if newRule == nil {
		// Delete operation
		delete(r.ruleMap, name)
	} else {
		r.ruleMap[name] = newRule
	}

	return true, r.onAnyChange()
}

func (r *FaultToleranceRuleAggregator) updateRateLimitStrategy(name string, newStrategy *v1alpha1.RateLimitStrategy) (bool, error) {
	r.updateMux.Lock()
	defer r.updateMux.Unlock()

	oldStrategy, hasIt := r.rateLimitStrategyMap[name]
	if hasIt && reflect.DeepEqual(oldStrategy, newStrategy) {
		// The updated strategy remains unchanged
		return false, nil
	}
	// Update rateLimitStrategyMap (TODO: resolve consistency)
	if newStrategy == nil {
		// Delete operation
		delete(r.rateLimitStrategyMap, name)
	} else {
		r.rateLimitStrategyMap[name] = newStrategy
	}

	return true, r.onAnyChange()
}

// NOTE: this must be guarded by write lock.
func (r *FaultToleranceRuleAggregator) onAnyChange() error {
	flowRules, err := r.assembleFlowRules(r.ruleMap, r.rateLimitStrategyMap)
	if err != nil {
		return err
	}
	_, err = flow.LoadRules(flowRules)
	return err
}

func (r *FaultToleranceRuleAggregator) assembleFlowRules(ruleMap map[string]*v1alpha1.FaultToleranceRule, rlStrategyMap map[string]*v1alpha1.RateLimitStrategy) ([]*flow.Rule, error) {
	rules := make([]*flow.Rule, 0)
	for _, ftRule := range r.ruleMap {
		for _, strategyRef := range ftRule.Spec.Strategies {
			if strategyRef.Kind != v1alpha1.RateLimitStrategyKind {
				continue
			}
			rlStrategy, hasIt := rlStrategyMap[strategyRef.Name]
			if hasIt {
				for _, target := range ftRule.Spec.Targets {
					sentinelFlowRule := &flow.Rule{Resource: target.TargetResourceName}
					r.fillFlowRuleWithRateLimitStrategy(sentinelFlowRule, rlStrategy)
					rules = append(rules, sentinelFlowRule)
				}
			}
		}
	}
	return rules, nil
}

func (r *FaultToleranceRuleAggregator) fillFlowRuleWithRateLimitStrategy(targetRule *flow.Rule, rls *v1alpha1.RateLimitStrategy) *flow.Rule {
	if targetRule == nil {
		return nil
	}
	if rls == nil {
		return targetRule
	}
	targetRule.Threshold = float64(rls.Spec.Threshold)
	targetRule.TokenCalculateStrategy = flow.Direct
	targetRule.ControlBehavior = flow.Reject
	targetRule.RelationStrategy = flow.CurrentResource
	targetRule.StatIntervalInMs = uint32(rls.Spec.StatDurationSeconds * 1000)
	return targetRule
}

func NewRuleAggregator() *FaultToleranceRuleAggregator {
	return &FaultToleranceRuleAggregator{
		ruleMap:              make(map[string]*v1alpha1.FaultToleranceRule),
		rateLimitStrategyMap: make(map[string]*v1alpha1.RateLimitStrategy),
	}
}
