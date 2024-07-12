package community

import (
	"testing"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/github/fakegithub"
	"k8s.io/test-infra/prow/labels"
	"k8s.io/test-infra/prow/repoowners"
)

type fakeOwnersClient struct {
	approvers map[string]sets.String
	reviewers map[string]sets.String
}

var _ repoowners.Interface = &fakeOwnersClient{}

func (f *fakeOwnersClient) LoadRepoAliases(org, repo, base string) (repoowners.RepoAliases, error) {
	return nil, nil
}

func (f *fakeOwnersClient) LoadRepoOwners(org, repo, base string) (repoowners.RepoOwner, error) {
	return &fakeRepoOwners{approvers: f.approvers, reviewers: f.reviewers}, nil
}

type fakeRepoOwners struct {
	approvers map[string]sets.String
	reviewers map[string]sets.String
}

var _ repoowners.RepoOwner = &fakeRepoOwners{}

func (f *fakeRepoOwners) FindApproverOwnersForFile(path string) string  { return "" }
func (f *fakeRepoOwners) FindReviewersOwnersForFile(path string) string { return "" }
func (f *fakeRepoOwners) FindLabelsForFile(path string) sets.String     { return nil }
func (f *fakeRepoOwners) IsNoParentOwners(path string) bool             { return false }
func (f *fakeRepoOwners) LeafApprovers(path string) sets.String         { return nil }
func (f *fakeRepoOwners) Approvers(path string) sets.String             { return f.approvers[path] }
func (f *fakeRepoOwners) LeafReviewers(path string) sets.String         { return nil }
func (f *fakeRepoOwners) Reviewers(path string) sets.String             { return f.reviewers[path] }
func (f *fakeRepoOwners) RequiredReviewers(path string) sets.String     { return nil }

func createFakeOwnersClient() *fakeOwnersClient {
	return &fakeOwnersClient{
		approvers: map[string]sets.String{
			"doc/README.md": {
				"user-1": {},
				"user-2": {},
			},
		},
		reviewers: map[string]sets.String{
			"doc/README.md": {
				"user-1": {},
				"user-3": {},
				"user-4": {},
				"user-5": {},
			},
		},
	}
}

func TestHandlePullRequest(t *testing.T) {
	SHA := "0bd3ed50c88cd53a09316bf7a298f900e9371652"
	treeSHA := "6dcb09b5b57875f334f61aebed695e2e4193db5e"
	cases := []struct {
		name          string
		event         github.PullRequestEvent
		collaborators []string
		labelsPresent []string
		changedFiles  []string
		containLabel  bool
		err           error
	}{
		{
			name: "new pull request by non-owner with no previous label",
			event: github.PullRequestEvent{
				Action: github.PullRequestActionOpened,
				PullRequest: github.PullRequest{
					Number: 101,
					Base: github.PullRequestBranch{
						Repo: github.Repo{
							Owner: github.User{
								Login: "kubernetes",
							},
							Name: "kubernetes",
						},
						Ref: "master",
					},
					User: github.User{
						Login: "non-owner",
						Name:  "non-owner",
					},
					MergeSHA: &SHA,
				},
				Sender: github.User{
					Name: "non-owner",
				},
			},
			collaborators: []string{},
			labelsPresent: []string{},
			changedFiles:  []string{},
			containLabel:  true,
		},
		{
			name: "new pull request by owner with no previous label",
			event: github.PullRequestEvent{
				Action: github.PullRequestActionOpened,
				PullRequest: github.PullRequest{
					Number: 101,
					Base: github.PullRequestBranch{
						Repo: github.Repo{
							Owner: github.User{
								Login: "kubernetes",
							},
							Name: "kubernetes",
						},
						Ref: "master",
					},
					User: github.User{
						Login: "user-1",
						Name:  "user-1",
					},
					MergeSHA: &SHA,
				},
				Sender: github.User{
					Name: "user-1",
				},
			},
			collaborators: []string{},
			labelsPresent: []string{},
			changedFiles:  []string{"doc/README.md"},
			containLabel:  false,
		},
		{
			name: "new pull request by owner wrong file",
			event: github.PullRequestEvent{
				Action: github.PullRequestActionOpened,
				PullRequest: github.PullRequest{
					Number: 101,
					Base: github.PullRequestBranch{
						Repo: github.Repo{
							Owner: github.User{
								Login: "kubernetes",
							},
							Name: "kubernetes",
						},
						Ref: "master",
					},
					User: github.User{
						Login: "user-1",
						Name:  "user-1",
					},
					MergeSHA: &SHA,
				},
				Sender: github.User{
					Name: "user-1",
				},
			},
			collaborators: []string{},
			labelsPresent: []string{},
			changedFiles:  []string{"random-path/README.md"},
			containLabel:  true,
		},
		{
			name: "new pull request by owner with previous label",
			event: github.PullRequestEvent{
				Action: github.PullRequestActionOpened,
				PullRequest: github.PullRequest{
					Number: 101,
					Base: github.PullRequestBranch{
						Repo: github.Repo{
							Owner: github.User{
								Login: "kubernetes",
							},
							Name: "kubernetes",
						},
						Ref: "master",
					},
					User: github.User{
						Login: "user-1",
						Name:  "user-1",
					},
					MergeSHA: &SHA,
				},
				Sender: github.User{
					Name: "user-1",
				},
			},
			collaborators: []string{},
			labelsPresent: []string{labels.CommunityContribution},
			changedFiles:  []string{"doc/README.md"},
			containLabel:  true,
		},
		{
			name: "pull request reopened by non-owner with no previous label",
			event: github.PullRequestEvent{
				Action: github.PullRequestActionReopened,
				PullRequest: github.PullRequest{
					Number: 101,
					Base: github.PullRequestBranch{
						Repo: github.Repo{
							Owner: github.User{
								Login: "kubernetes",
							},
							Name: "kubernetes",
						},
						Ref: "master",
					},
					User: github.User{
						Login: "non-owner",
						Name:  "non-owner",
					},
					MergeSHA: &SHA,
				},
				Sender: github.User{
					Name: "non-owner",
				},
			},
			collaborators: []string{},
			labelsPresent: []string{},
			changedFiles:  []string{},
			containLabel:  true,
		},
		{
			name: "pull request reopened by non-owner with previous label",
			event: github.PullRequestEvent{
				Action: github.PullRequestActionReopened,
				PullRequest: github.PullRequest{
					Number: 101,
					Base: github.PullRequestBranch{
						Repo: github.Repo{
							Owner: github.User{
								Login: "kubernetes",
							},
							Name: "kubernetes",
						},
						Ref: "master",
					},
					User: github.User{
						Login: "non-owner",
						Name:  "non-owner",
					},
					MergeSHA: &SHA,
				},
				Sender: github.User{
					Name: "non-owner",
				},
			},
			collaborators: []string{},
			labelsPresent: []string{labels.CommunityContribution},
			changedFiles:  []string{},
			containLabel:  true,
		},
		{
			name: "pull request reopened by owner with no previous label",
			event: github.PullRequestEvent{
				Action: github.PullRequestActionReopened,
				PullRequest: github.PullRequest{
					Number: 101,
					Base: github.PullRequestBranch{
						Repo: github.Repo{
							Owner: github.User{
								Login: "kubernetes",
							},
							Name: "kubernetes",
						},
						Ref: "master",
					},
					User: github.User{
						Login: "user-1",
						Name:  "user-1",
					},
					MergeSHA: &SHA,
				},
				Sender: github.User{
					Name: "user-1",
				},
			},
			collaborators: []string{},
			labelsPresent: []string{},
			changedFiles:  []string{"doc/README.md"},
			containLabel:  false,
		},
		{
			name: "pull request review request by non-owner with no previous label",
			event: github.PullRequestEvent{
				Action: github.PullRequestActionReviewRequested,
				PullRequest: github.PullRequest{
					Number: 101,
					Base: github.PullRequestBranch{
						Repo: github.Repo{
							Owner: github.User{
								Login: "kubernetes",
							},
							Name: "kubernetes",
						},
						Ref: "master",
					},
					User: github.User{
						Login: "non-owner",
						Name:  "non-owner",
					},
					MergeSHA: &SHA,
				},
				Sender: github.User{
					Name: "non-owner",
				},
			},
			collaborators: []string{},
			labelsPresent: []string{},
			changedFiles:  []string{},
			containLabel:  true,
		},
		{
			name: "pull request review request by owner with previous label",
			event: github.PullRequestEvent{
				Action: github.PullRequestActionReviewRequested,
				PullRequest: github.PullRequest{
					Number: 101,
					Base: github.PullRequestBranch{
						Repo: github.Repo{
							Owner: github.User{
								Login: "kubernetes",
							},
							Name: "kubernetes",
						},
						Ref: "master",
					},
					User: github.User{
						Login: "user-1",
						Name:  "user-1",
					},
					MergeSHA: &SHA,
				},
				Sender: github.User{
					Name: "user-1",
				},
			},
			collaborators: []string{},
			labelsPresent: []string{},
			changedFiles:  []string{"doc/README.md"},
			containLabel:  false,
		},
	}

	myFakeOwnersClient := createFakeOwnersClient()

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			changes := []github.PullRequestChange{}
			for _, changed := range c.changedFiles {
				changes = append(changes, github.PullRequestChange{
					Filename: changed,
				})
			}

			fakeGitHub := &fakegithub.FakeClient{
				PullRequests: map[int]*github.PullRequest{
					c.event.PullRequest.Number: {
						Base: github.PullRequestBranch{
							Ref: c.event.PullRequest.Base.Ref,
							Repo: github.Repo{
								Name: "test",
							},
						},
						Head: github.PullRequestBranch{
							SHA: SHA,
						},
					},
				},
				PullRequestChanges: map[int][]github.PullRequestChange{
					c.event.PullRequest.Number: changes,
				},
				Commits:       make(map[string]github.SingleCommit),
				Collaborators: c.collaborators,
			}
			commit := github.SingleCommit{}
			commit.Commit.Tree.SHA = treeSHA
			fakeGitHub.Commits[SHA] = commit

			// add the labels as adding them to the existingLabels field did not work
			for _, label := range c.labelsPresent {
				if err := fakeGitHub.AddLabel(c.event.PullRequest.Base.Repo.Owner.Login, c.event.PullRequest.Base.Repo.Name, c.event.PullRequest.Number, label); err != nil {
					t.Fatalf("%v", err)
				}
			}

			err := handle(fakeGitHub, myFakeOwnersClient, c.event)
			if err != nil && c.err == nil {
				t.Fatalf("handle function error: %v", err)
			}
			if c.err != nil && err == nil {
				t.Fatalf("handle function had no error, expected error: %v", c.err)
			}

			currentLabels, err := fakeGitHub.GetIssueLabels(c.event.PullRequest.Base.Repo.Owner.Login, c.event.PullRequest.Base.Repo.Name, c.event.PullRequest.Number)
			if err != nil {
				t.Fatal("failed to find issue labels")
			}

			// determine if the label is present
			hasLabel := false
			for _, l := range currentLabels {
				if l.Name == labels.CommunityContribution {
					hasLabel = true
				}
			}
			if !hasLabel && c.containLabel {
				t.Fatalf("%s label not applied", labels.CommunityContribution)
			}

		})
	}
}
