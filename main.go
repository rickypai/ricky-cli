package main

import (
	"context"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

const godel1Dir = "/Users/ricky/workspace/src/github.com/vsco/godel"
const godel2Dir = "/Users/ricky/workspace/src/github.com/vsco/godel2"
const godel3Dir = "/Users/ricky/workspace/src/github.com/vsco/godel3"

var godelDirs = []string{
	godel1Dir,
	godel2Dir,
	godel3Dir,
}

const pagesToCheck = 3

var ghMap = map[string]map[string][]string{
	"vsco": map[string][]string{
		"godel": godelDirs,
		"chef": []string{
			"/Users/ricky/workspace/chef",
		},
		"web": []string{
			"/Users/ricky/workspace/web",
		},
		"kube-config": []string{
			"/Users/ricky/workspace/kube-config",
		},
		"titan-grpc": []string{
			"/Users/ricky/workspace/titan-grpc",
		},
		"infra": []string{
			"/Users/ricky/workspace/infra",
		},
		"rules_protobuf": []string{
			"/Users/ricky/workspace/rules_protobuf_vsco",
		},
		"jvm": []string{
			"/Users/ricky/workspace/jvm",
		},
		"js": []string{
			"/Users/ricky/workspace/js",
		},
		"uni": []string{
			"/Users/ricky/workspace/uni",
		},
		"vsco-cli": []string{
			"/Users/ricky/workspace/src/github.com/vsco/vsco-cli",
		},
	},
}

func main() {
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		panic("Need to set GITHUB_TOKEN.")
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	var wg sync.WaitGroup

	for user, repoMap := range ghMap {
		for repo, repoDirs := range repoMap {
			wg.Add(1)
			go func(user, repo string, repoDirs []string) {
				defer wg.Done()
				syncRepo(client, user, repo, repoDirs)
			}(user, repo, repoDirs)
		}
	}

	wg.Wait()
}

func syncRepo(client *github.Client, user, repo string, localDirs []string) {
	ctx := context.Background()

	trackedMap := make(map[string]map[int]bool)

	for _, localDir := range localDirs {
		trackedMap[localDir] = make(map[int]bool)
		issues, _ := trackedIssues(localDir)

		for _, issue := range issues {
			pr, _, err := client.PullRequests.Get(ctx, user, repo, issue)
			if err != nil {
				panic(err)
			}

			trackedMap[localDir][issue] = true
			syncPR(pr, localDir, true)
		}
	}

	opt := github.PullRequestListOptions{
		State:       "all",
		ListOptions: github.ListOptions{},
	}

	for i := 1; i < pagesToCheck; i++ {
		opt.ListOptions.Page = i

		prs, _, err := client.PullRequests.List(ctx, user, repo, &opt)
		if err != nil {
			panic(err)
		}

		for _, pr := range prs {
			for _, localDir := range localDirs {
				_, tracked := trackedMap[localDir][pr.GetNumber()]
				syncPR(pr, localDir, tracked)
			}
		}
	}
}

func syncPR(pr *github.PullRequest, localDir string, tracked bool) {
	if pr.GetUser().GetLogin() != "rickypai" {
		return
	}

	if err := branchExists(pr.Head.GetRef(), localDir); err != nil {
		return
	}

	if pr.GetState() == "closed" {
		syncClosedPR(pr, localDir)
	} else if pr.GetState() == "open" && !tracked {
		syncOpenPR(pr, localDir)
	}
}

func branchExists(branchName, localDir string) error {
	cmd := exec.Command("git", "show", branchName)
	cmd.Dir = localDir
	_, err := cmd.Output()
	if err != nil {
		return err
	}
	return nil
}

func syncClosedPR(pr *github.PullRequest, localDir string) {
	log.Printf("deleting '%v'", pr.Head.GetRef())

	// delete closed branches
	execCommand(
		localDir,
		"git", "branch", "-D", pr.Head.GetRef(),
	)
}

func syncOpenPR(pr *github.PullRequest, localDir string) {
	log.Printf("setting PR for '%v'", pr.Head.GetRef())

	// set GH issue
	execCommand(
		localDir,
		"twig", "--branch", pr.Head.GetRef(), "issue", strconv.Itoa(pr.GetNumber()),
	)

	// set diff branch
	execCommand(
		localDir,
		"twig", "--branch", pr.Head.GetRef(), "diff-branch", pr.Base.GetRef(),
	)
}

func trackedIssues(localDir string) ([]int, error) {
	out, err := execCommand(
		localDir,
		"git", "config", "--get-regexp", "branch\\.(.+).\\issue",
	)

	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(out), "\n")

	issues := make([]int, 0, len(lines))

	for _, line := range lines {
		tokens := strings.Split(line, " ")

		if len(tokens) == 2 {
			i, err := strconv.Atoi(tokens[1])
			if err != nil {
				continue
			}

			issues = append(issues, i)
		}
	}

	return issues, nil
}

func execCommandDirs(dirs []string, name string, arg ...string) {
	for _, dir := range dirs {
		execCommand(dir, name, arg...)
	}
}

func execCommand(dir string, name string, arg ...string) ([]byte, error) {
	cmd := exec.Command(name, arg...)
	cmd.Dir = dir
	return cmd.Output()
}
