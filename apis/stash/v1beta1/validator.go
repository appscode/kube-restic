package v1beta1

import (
	"fmt"
)

// TODO: complete
func (r BackupSession) IsValid() error {
	return nil
}

// TODO: complete
func (r RestoreSession) IsValid() error {
	// ========== spec.Rules validation================
	// We must ensure following:
	// 1. There is at most one rule with empty targetHosts field.
	// 2. No two rules with non-emtpy targetHosts matches for a host.
	// 3. If snapshot field is specified in a rule then paths is not specified.

	// ensure the there is at most one rule with source
	var ruleIdx []int
	for i, rule := range r.Spec.Rules {
		if len(rule.TargetHosts) == 0 {
			ruleIdx = append(ruleIdx, i)
		}
	}
	if len(ruleIdx) > 1 {
		return fmt.Errorf("\n\t"+
			"Error: Invalid RestoreSession specification.\n\t"+
			"Reason: %s.\n\t"+
			"Hints: There can be at most one rule with empty targetHosts.", multipleRuleWithEmptyTargetHostError(ruleIdx))
	}

	// ensure that no two rules with non-emtpy targetHosts matches for a host
	res := make(map[string]int, 0)
	for i, rule := range r.Spec.Rules {
		for _, host := range rule.TargetHosts {
			v, ok := res[host]
			if ok {
				return fmt.Errorf("\n\t"+
					"Error: Invalid RestoreSession specification.\n\t"+
					"Reason: Multiple rules (rule[%d] and rule[%d]) match for host %q.\n\t"+
					"Hints: There could be only one matching rule for a host.", v, i, host)
			} else {
				res[host] = i
			}
		}
	}

	// ensure that path is not specified in a rule if snapshot field is specified
	for i, rule := range r.Spec.Rules {
		if len(rule.Snapshots) != 0 && len(rule.Paths) != 0 {
			return fmt.Errorf("\n\t"+
				"Error: Invalid RestoreSession specification.\n\t"+
				"Reason: Both 'snapshots' and 'paths' fileds are specified in rule[%d].\n\t"+
				"Hints: A snpashot contains backup data of only one directory. So, you can't specify 'paths' if you specify snapshot field.", i)
		}
	}
	return nil
}

func multipleRuleWithEmptyTargetHostError(ruleIndexes []int) string {
	ids := ""
	for i, idx := range ruleIndexes {
		ids += fmt.Sprintf("rule[%d]", idx)
		if i < len(ruleIndexes)-1 {
			ids += ", "
		}
	}
	return fmt.Sprintf("%d rules found with empty targetHosts (Rules: %s)", len(ruleIndexes), ids)
}

func (b BackupBlueprint) IsValid() error {
	// We must ensure the following:
	// 1. Spec.schedule
	// 2. Spec.Backend.StorageSecretName
	if b.Spec.Schedule == "" {
		return fmt.Errorf("\n\t" +
			"Error:  Invalid BackupBlueprint specification.\n\t" +
			"Reason: Schedule hasn't been specified\n\t" +
			"Hints: Provide a cron expression(i.e. \"* */5 * * *\") in 'spec.schedule' filed.")
	}
	if b.Spec.Backend.StorageSecretName == "" {
		return fmt.Errorf("\n\t" +
			"Error:  Invalid BackupBlueprint specification.\n\t" +
			"Reason: Storage secret is not specified\n\t")
	}
	return nil

}

func (b BackupConfiguration) IsValid() error {
	if b.Spec.Schedule == "" {
		return fmt.Errorf("\n\t" +
			"Error:  Invalid BackupBlueprint specification.\n\t" +
			"Reason: Schedule hasn't been specified\n\t" +
			"Hints: Provide a cron expression(i.e. \"* */5 * * *\") in 'spec.schedule' filed.")
	}
	return nil
}
