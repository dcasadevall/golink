package go_protogen

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/language"
	"github.com/bazelbuild/bazel-gazelle/repo"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

const (
	DirectiveMultirunRulePath = "protogen_multirun_rule_path"
	DirectiveMultirunRuleName = "protogen_multirun_rule_name"
	GenCommentFmt             = "# %s statement generated by go_protogen + gazelle"
)

type xlang struct {
	allProtogenRules []string

	// multirunRootPath defines the path, relative to the repository root,
	// where the multirun rule should be inserted
	multirunRootPath string

	// multirunRuleName defines the name of the multirun rule to be created
	multirunRuleName string
}

func NewLanguage() language.Language {
	return &xlang{
		multirunRootPath: "",
		multirunRuleName: "go_protogen",
	}
}

func (x *xlang) Name() string {
	return "go_protogen"
}

func (x *xlang) Kinds() map[string]rule.KindInfo {
	return map[string]rule.KindInfo{
		"go_protogen": {},
	}
}

func (x *xlang) Loads() []rule.LoadInfo {
	return []rule.LoadInfo{
		{
			Name:    "@go_protogen//proto:proto.bzl",
			Symbols: []string{"go_protogen"},
		},
	}
}

func (x *xlang) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
}

func (x *xlang) CheckFlags(fs *flag.FlagSet, c *config.Config) error {
	return nil
}

func (x *xlang) KnownDirectives() []string {
	return []string{
		DirectiveMultirunRulePath,
		DirectiveMultirunRuleName,
	}
}

func (x *xlang) Configure(c *config.Config, rel string, file *rule.File) {

	if file == nil {
		return
	}

	// Only allow configuration on root BUILD file
	if rel != "" {
		return
	}

	for _, directive := range file.Directives {
		switch directive.Key {
		case DirectiveMultirunRulePath:
			x.multirunRootPath = directive.Value
		case DirectiveMultirunRuleName:
			x.multirunRuleName = directive.Value
		}
	}
}

func (x *xlang) GenerateRules(args language.GenerateArgs) language.GenerateResult {
	rules := make([]*rule.Rule, 0)
	imports := make([]interface{}, 0)

	for _, r := range args.OtherGen {
		if r.Kind() == "go_proto_library" {
			depName := r.Name()
			protogenRule := rule.NewRule("go_protogen", r.Name()+"_gen")
			protogenRule.SetAttr("dep", ":"+depName)
			protogenRule.SetAttr("version", "v1")
			protogenRule.SetAttr("visibility", []string{"//visibility:public"})
			protogenRule.AddComment(fmt.Sprintf(GenCommentFmt, "Rule"))
			rules = append(rules, protogenRule)
			imports = append(imports, nil)
			x.allProtogenRules = append(x.allProtogenRules, "//"+args.Rel+":"+protogenRule.Name())
		}
	}

	return language.GenerateResult{
		Gen:     rules,
		Imports: imports,
	}
}

func (x *xlang) Fix(c *config.Config, f *rule.File) {
	// No specific fix needed for this example
}

func (x *xlang) Imports(c *config.Config, r *rule.Rule, f *rule.File) []resolve.ImportSpec {
	return nil
}

func (x *xlang) Embeds(r *rule.Rule, from label.Label) []label.Label {
	return nil
}

func (x *xlang) Resolve(c *config.Config, ix *resolve.RuleIndex, rc *repo.RemoteCache, r *rule.Rule, imports interface{}, from label.Label) {
	// Defer to ensure root `BUILD` modification happens only once
	defer func() {
		rootBuildPath := filepath.Join(c.RepoRoot, x.multirunRootPath, "BUILD")
		rootFile, err := rule.LoadFile(rootBuildPath, "")
		if err != nil {
			log.Printf("Error loading root BUILD file: %v", err)
			return // Exit if there's an error loading the file
		}

		// Check for existing multirun rule and delete if it exists
		for _, r := range rootFile.Rules {
			if r.Kind() == "multirun" && r.Name() == x.multirunRuleName {
				r.Delete()
			}
		}

		// Create a new multirun rule to include all proto links
		multirunRule := rule.NewRule("multirun", x.multirunRuleName)
		multirunRule.SetAttr("commands", x.allProtogenRules)
		multirunRule.SetAttr("jobs", 0)
		multirunRule.AddComment(fmt.Sprintf(GenCommentFmt, "Rule"))
		multirunRule.AddComment("# Used to run all go_proto_library rules in the Workspace")
		multirunRule.Insert(rootFile)

		// Ensure the load statement is present
		load := rule.NewLoad("@rules_multirun//:defs.bzl")
		load.Add("command")
		load.Add("multirun")
		load.AddComment(fmt.Sprintf(GenCommentFmt, "Load"))
		load.Insert(rootFile, 0)

		if err := rootFile.Save(rootBuildPath); err != nil {
			log.Printf("Error saving root BUILD file: %v", err)
		}
	}()
}
