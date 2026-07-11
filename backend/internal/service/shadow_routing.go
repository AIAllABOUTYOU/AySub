package service

func parentHealthyForShadow(account *Account, lookup func(int64) *Account) bool {
	if account == nil || !account.IsShadow() {
		return true
	}
	if lookup == nil {
		return false
	}
	parent := lookup(*account.ParentAccountID)
	return parent != nil && parent.IsOpenAIOAuth() && parent.IsCredentialUsableForShadow()
}

func sparkModelVariants() []string {
	out := make([]string, 0, 1)
	for alias, target := range codexModelMap {
		if target == "gpt-5.3-codex-spark" {
			out = append(out, alias)
		}
	}
	return out
}

func defaultSparkShadowModelMapping() map[string]any {
	variants := sparkModelVariants()
	mapping := make(map[string]any, len(variants))
	for _, model := range variants {
		mapping[model] = model
	}
	return mapping
}
