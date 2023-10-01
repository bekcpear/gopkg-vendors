package actionlint

import "strings"

// RuleEnvVar is a rule checker to check environment variables setup.
type RuleEnvVar struct {
	RuleBase
}

// NewRuleEnvVar creates new RuleEnvVar instance.
func NewRuleEnvVar() *RuleEnvVar {
	return &RuleEnvVar{
		RuleBase: RuleBase{
			name: "env-var",
			desc: "Checks for environment variables configuration at \"env:\"",
		},
	}
}

// VisitStep is callback when visiting Step node.
func (rule *RuleEnvVar) VisitStep(n *Step) error {
	rule.checkEnv(n.Env)
	return nil
}

// VisitJobPre is callback when visiting Job node before visiting its children.
func (rule *RuleEnvVar) VisitJobPre(n *Job) error {
	rule.checkEnv(n.Env)
	if n.Container != nil {
		rule.checkEnv(n.Container.Env)
	}
	for _, s := range n.Services {
		rule.checkEnv(s.Container.Env)
	}
	return nil
}

// VisitWorkflowPre is callback when visiting Workflow node before visiting its children.
func (rule *RuleEnvVar) VisitWorkflowPre(n *Workflow) error {
	rule.checkEnv(n.Env)
	return nil
}

func (rule *RuleEnvVar) checkEnv(env *Env) {
	if env == nil || env.Expression != nil {
		return
	}
	for _, v := range env.Vars {
		if v.Name.ContainsExpression() {
			continue // Key name can contain expressions (#312)
		}
		if strings.ContainsAny(v.Name.Value, "&= 	") {
			rule.Errorf(
				v.Name.Pos,
				"environment variable name %q is invalid. '&', '=' and spaces should not be contained",
				v.Name.Value,
			)
		}
	}
}
