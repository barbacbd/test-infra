package community

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/labels"
	"k8s.io/test-infra/prow/pluginhelp"
	"k8s.io/test-infra/prow/plugins"
	"k8s.io/test-infra/prow/repoowners"
)

const (
	// PluginName defines this plugin's registered name.
	PluginName = "community"
)

func init() {
	plugins.RegisterPullRequestHandler(PluginName, handlePullRequest, helpProvider)
}

func helpProvider(config *plugins.Configuration, enabledRepos []string) (*pluginhelp.PluginHelp, error) {
	return &pluginhelp.PluginHelp{
			Description: "The community plugin automatically applies the '" + labels.CommunityContribution + "' label to PRs where the author is not in the OWNERS file(s).",
		},
		nil
}

type ownersClient interface {
	FindLabelsForFile(path string) sets.String
}

type githubClient interface {
	AddLabel(org, repo string, number int, label string) error
	GetIssueLabels(org, repo string, number int) ([]github.Label, error)
	GetRepoLabels(owner, repo string) ([]github.Label, error)
	GetPullRequestChanges(org, repo string, number int) ([]github.PullRequestChange, error)
}

func handlePullRequest(pc plugins.Agent, pre github.PullRequestEvent) error {
	if pre.Action != github.PullRequestActionOpened &&
		pre.Action != github.PullRequestActionReopened &&
		pre.Action != github.PullRequestActionSynchronize {
		return nil
	}

	return handle(pc.GitHubClient, pc.OwnersClient, pre)
}

func handle(gc githubClient, oc repoowners.Interface, pre github.PullRequestEvent) error {
	org := pre.PullRequest.Base.Repo.Owner.Login
	repo := pre.PullRequest.Base.Repo.Name
	number := pre.PullRequest.Number

	ro, err := oc.LoadRepoOwners(org, repo, pre.PullRequest.Base.Ref)
	if err != nil {
		return fmt.Errorf("error loading RepoOwners: %w", err)
	}

	filenames, err := getChangedFiles(gc, org, repo, number)
	if err != nil {
		return err
	}

	if !loadReviewers(ro, filenames).Has(github.NormLogin(pre.Sender.Name)) {
		currentLabels, err := gc.GetIssueLabels(org, repo, number)
		if err != nil {
			return fmt.Errorf("could not get labels for PR %s/%s:%d in %s plugin: %w", org, repo, number, PluginName, err)
		}

		// determine if there is a label already
		hasLabel := false
		for _, l := range currentLabels {
			if l.Name == labels.CommunityContribution {
				hasLabel = true
			}
		}

		// label does not already exist, add it
		if !hasLabel {
			if err := gc.AddLabel(org, repo, number, labels.CommunityContribution); err != nil {
				return fmt.Errorf("github failed to add the following label: %s", labels.CommunityContribution)
			}
		}
	}

	return nil
}

// loadReviewers returns all reviewers and approvers from all OWNERS files that
// cover the provided filenames.
func loadReviewers(ro repoowners.RepoOwner, filenames []string) sets.String {
	reviewers := sets.String{}
	for _, filename := range filenames {
		reviewers = reviewers.Union(ro.Approvers(filename)).Union(ro.Reviewers(filename))
	}
	return reviewers
}

// getChangedFiles returns all the changed files for the provided pull request.
func getChangedFiles(gc githubClient, org, repo string, number int) ([]string, error) {
	changes, err := gc.GetPullRequestChanges(org, repo, number)
	if err != nil {
		return nil, fmt.Errorf("%s failed to get PR changes for %s/%s#%d", PluginName, org, repo, number)
	}
	var filenames []string
	for _, change := range changes {
		filenames = append(filenames, change.Filename)
	}
	return filenames, nil
}
