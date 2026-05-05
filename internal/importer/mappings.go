package importer

// CommonMappings provides glob patterns for resource kinds consistent across
// most providers. Providers compose these with their own patterns.
var CommonMappings = []KindMapping{
	{Pattern: "hooks/*.sh", Kind: KindHookScript, Layout: FlatFile},
	{Pattern: "hooks/**", Kind: KindHookScript, Layout: FlatFile},
	{Pattern: "agents/*.md", Kind: KindAgent, Layout: FlatFile},
	{Pattern: "skills/*/SKILL.md", Kind: KindSkill, Layout: DirectoryPerEntry},
	{Pattern: "skills/*/references/**", Kind: KindSkillAsset, Layout: DirectoryPerEntry},
	{Pattern: "skills/*/scripts/**", Kind: KindSkillAsset, Layout: DirectoryPerEntry},
	{Pattern: "skills/*/assets/**", Kind: KindSkillAsset, Layout: DirectoryPerEntry},
	{Pattern: "rules/**/*.md", Kind: KindRule, Layout: FlatFile},
}
