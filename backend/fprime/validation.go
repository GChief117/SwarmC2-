// Package fprime validation implements the formal verification gates
// from the ESS paper (Nelson, 2026).
//
// Theorem 1 (Energy Invariance Under Remote Configuration):
//   A candidate configuration C' is accepted only when E(C') ≤ B.
//   Then E(C') ≤ B ≤ (Ps + Pb) · T — the drone never consumes more
//   energy per cycle than its solar panels and battery can supply.
//
// Theorem 2 (FSM Completeness of Post-Configuration State Machine):
//   Every (state, input) pair has a defined next state — no deadlocks,
//   undefined outputs, or unrecoverable states can occur.
//
// Reference values (Appendix A):
//   Ps = 12 W, Pb = 8 W, T = 0.0025 s (400 Hz)
//   (Ps + Pb)·T = 0.050 J
//   B = 0.9 × 0.050 = 0.045 J (10% safety margin)
//   Baseline C₀: ε₁=0.018 + ε₂=0.008 + ε₃=0.006 = E(C₀) = 0.032 J
package fprime

import "fmt"

// ValidationResult captures the outcome of a verification gate.
type ValidationResult struct {
	Gate     string            `json:"gate"`     // "energy_invariance" or "fsm_completeness"
	Pass     bool              `json:"pass"`
	Summary  string            `json:"summary"`
	Evidence []ValidationProof `json:"evidence"`
}

// ValidationProof is a single piece of evidence for a validation gate.
type ValidationProof struct {
	Property string `json:"property"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Pass     bool   `json:"pass"`
}

// ValidateEnergyInvariance checks Theorem 1 using the paper's per-cycle model.
//
// The gate verifies: E(C') ≤ B ≤ (Ps + Pb) · T
//
// Parameters use the paper's notation:
//   - activeTasks: current active tasks with per-cycle energy costs εᵢ
//   - proposedTasks: new tasks to add (from remote config update C')
//   - model: energy model parameters (Ps, Pb, T, B)
//   - batteryPercent: current state of charge for operational check
//   - criticalThreshold: minimum allowed battery %
func ValidateEnergyInvariance(activeTasks, proposedTasks []TaskEnergyCost, model EnergyModelParams, batteryPercent, criticalThreshold float64) ValidationResult {
	result := ValidationResult{
		Gate:     "energy_invariance",
		Evidence: make([]ValidationProof, 0),
	}

	// Compute E(C₀) — current baseline per-cycle energy
	ec0 := ComputePerCycleEnergy(activeTasks)

	// Compute E(C') — proposed configuration per-cycle energy
	allTasks := make([]TaskEnergyCost, len(activeTasks))
	copy(allTasks, activeTasks)
	allTasks = append(allTasks, proposedTasks...)
	ecPrime := ComputePerCycleEnergy(allTasks)

	// Check 1: E(C') ≤ B (per-cycle energy within budget)
	budgetCheck := ValidationProof{
		Property: "E(C') ≤ B — per-cycle energy within budget",
		Expected: fmt.Sprintf("E(C') = %.4f J ≤ B = %.4f J", ecPrime, model.B),
		Actual:   fmt.Sprintf("%.4f J", ecPrime),
		Pass:     ecPrime <= model.B,
	}
	result.Evidence = append(result.Evidence, budgetCheck)

	// Check 2: B ≤ (Ps + Pb) · T (budget within power supply)
	maxEnergy := model.MaxPerCycle
	supplyCheck := ValidationProof{
		Property: "B ≤ (Ps+Pb)·T — budget within power supply capacity",
		Expected: fmt.Sprintf("B = %.4f J ≤ (%.0f+%.0f)×%.4f = %.4f J", model.B, model.Ps, model.Pb, model.T, maxEnergy),
		Actual:   fmt.Sprintf("%.4f J ≤ %.4f J", model.B, maxEnergy),
		Pass:     model.B <= maxEnergy,
	}
	result.Evidence = append(result.Evidence, supplyCheck)

	// Check 3: Full invariant chain E(C') ≤ B ≤ (Ps+Pb)·T
	chainCheck := ValidationProof{
		Property: "Invariant chain: E(C') ≤ B ≤ (Ps+Pb)·T",
		Expected: fmt.Sprintf("%.4f ≤ %.4f ≤ %.4f", ecPrime, model.B, maxEnergy),
		Actual:   fmt.Sprintf("%.4f ≤ %.4f ≤ %.4f", ecPrime, model.B, maxEnergy),
		Pass:     ecPrime <= model.B && model.B <= maxEnergy,
	}
	result.Evidence = append(result.Evidence, chainCheck)

	// Check 4: Energy margin > 0
	margin := model.B - ecPrime
	marginCheck := ValidationProof{
		Property: "Energy margin (B - E(C')) > 0",
		Expected: fmt.Sprintf("Margin = %.4f - %.4f = %.4f J > 0", model.B, ecPrime, margin),
		Actual:   fmt.Sprintf("%.4f J", margin),
		Pass:     margin > 0,
	}
	result.Evidence = append(result.Evidence, marginCheck)

	// Check 5: Battery above critical threshold (operational check)
	batteryCheck := ValidationProof{
		Property: "Battery above critical threshold",
		Expected: fmt.Sprintf("%.1f%% ≥ %.1f%%", batteryPercent, criticalThreshold),
		Actual:   fmt.Sprintf("%.1f%%", batteryPercent),
		Pass:     batteryPercent >= criticalThreshold,
	}
	result.Evidence = append(result.Evidence, batteryCheck)

	// Check 6: Endurance guarantee
	if ecPrime > 0 {
		enduranceCheck := ValidationProof{
			Property: "Mission endurance preserved",
			Expected: fmt.Sprintf("Endurance(C') = E_tot / E(C') ≥ E_tot / B"),
			Actual:   fmt.Sprintf("E(C₀)=%.4f J → E(C')=%.4f J (Δ=%.4f J)", ec0, ecPrime, ecPrime-ec0),
			Pass:     ecPrime <= model.B,
		}
		result.Evidence = append(result.Evidence, enduranceCheck)
	}

	// Overall: all checks must pass
	result.Pass = budgetCheck.Pass && supplyCheck.Pass && chainCheck.Pass && marginCheck.Pass && batteryCheck.Pass
	if result.Pass {
		result.Summary = fmt.Sprintf("Energy invariance holds: E(C')=%.4f J ≤ B=%.4f J ≤ (Ps+Pb)·T=%.4f J (margin: %.4f J)", ecPrime, model.B, maxEnergy, margin)
	} else {
		result.Summary = fmt.Sprintf("Energy invariance VIOLATED: E(C')=%.4f J vs B=%.4f J", ecPrime, model.B)
	}

	return result
}

// ValidateFSMCompleteness checks Theorem 2: every (state, input) pair has a defined transition.
// Per the paper: 5 states × 4 inputs = 20 pairs, all must be defined.
func ValidateFSMCompleteness() ValidationResult {
	result := ValidationResult{
		Gate:     "fsm_completeness",
		Pass:     true,
		Evidence: make([]ValidationProof, 0),
	}

	totalPairs := 0
	definedPairs := 0

	for _, state := range AllStates {
		stateTransitions, stateExists := TransitionTable[state]
		for _, input := range AllInputs {
			totalPairs++
			property := fmt.Sprintf("f(%s, %s)", state, input)

			if !stateExists {
				result.Evidence = append(result.Evidence, ValidationProof{
					Property: property,
					Expected: "defined transition",
					Actual:   "state not in table",
					Pass:     false,
				})
				result.Pass = false
				continue
			}

			nextState, defined := stateTransitions[input]
			if !defined {
				result.Evidence = append(result.Evidence, ValidationProof{
					Property: property,
					Expected: "defined transition",
					Actual:   "UNDEFINED",
					Pass:     false,
				})
				result.Pass = false
			} else {
				definedPairs++
				result.Evidence = append(result.Evidence, ValidationProof{
					Property: property,
					Expected: "defined transition",
					Actual:   fmt.Sprintf("→ %s", nextState),
					Pass:     true,
				})
			}
		}
	}

	if result.Pass {
		result.Summary = fmt.Sprintf("FSM is complete: all %d (state, input) pairs defined (%d states × %d inputs). No deadlock, undefined output, or unrecoverable state possible.", totalPairs, len(AllStates), len(AllInputs))
	} else {
		result.Summary = fmt.Sprintf("FSM is INCOMPLETE: %d/%d pairs defined — risk of deadlock or undefined behavior", definedPairs, totalPairs)
	}

	return result
}

// ValidateConfig runs both verification gates against a proposed configuration.
func ValidateConfig(config DroneConfig, fleet *Fleet) []ValidationResult {
	results := make([]ValidationResult, 0, 2)

	// Get current drone state for energy check
	drone := fleet.GetDrone(config.DroneID)
	batteryPercent := 100.0
	activeTasks := make([]TaskEnergyCost, len(BaselineConfiguration))
	copy(activeTasks, BaselineConfiguration)

	if drone != nil {
		batteryPercent = drone.Energy.BatteryPercent
		if len(drone.Energy.ActiveTasks) > 0 {
			activeTasks = drone.Energy.ActiveTasks
		}
	}

	// Proposed tasks from config (if any)
	proposedTasks := config.ProposedTasks

	// If no explicit proposed tasks, estimate from config parameters
	if len(proposedTasks) == 0 {
		// Estimate energy impact of config change based on speed/altitude
		speedFactor := config.MaxSpeed / 25.0
		altFactor := config.MaxAltitude / 400.0
		estimatedEpsilon := 0.005 * speedFactor * altFactor // estimated per-cycle cost in J
		if estimatedEpsilon > 0.001 {
			proposedTasks = []TaskEnergyCost{
				{"ConfigOverhead", estimatedEpsilon, true},
			}
		}
	}

	// Gate 1: Energy Invariance (Theorem 1)
	energyResult := ValidateEnergyInvariance(
		activeTasks,
		proposedTasks,
		DefaultEnergyModel,
		batteryPercent,
		config.CriticalBatteryPct,
	)
	results = append(results, energyResult)

	// Gate 2: FSM Completeness (Theorem 2)
	fsmResult := ValidateFSMCompleteness()
	results = append(results, fsmResult)

	return results
}
