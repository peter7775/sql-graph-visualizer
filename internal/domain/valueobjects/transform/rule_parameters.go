package transform

type RuleParameters struct {
	conditions map[string]interface{}
	options    map[string]interface{}
}

func NewRuleParameters(conditions map[string]interface{}, options map[string]interface{}) RuleParameters {
	return RuleParameters{
		conditions: conditions,
		options:    options,
	}
}

func (rp RuleParameters) GetCondition(key string) interface{} {
	return rp.conditions[key]
}

func (rp RuleParameters) GetOption(key string) interface{} {
	return rp.options[key]
}
