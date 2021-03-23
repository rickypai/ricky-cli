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

const pagesToCheck = 3

var ghMap = map[string]map[string][]string{
	"rickypai": {
		"web-template": {
			"/home/rickypai/workspace/web-template",
			"/mnt/t/workspace/web-template",
			"/Users/rickypai/workspace/web-template",
		},
		"rickysdc": {
			"/home/rickypai/workspace/rickysdc",
			"/Users/rickypai/workspace/rickysdc",
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

	wg := sync.WaitGroup{}

	for i := 1; i <= 3; i++ {
		wg.Add(1)
		go func(page int) {
			defer wg.Done()
			syncPRs(client, page)
		}(i)
	}

	wg.Wait()
}

func syncPRs(client *github.Client, page int) {
	ctx := context.Background()

	issues, _, _ := client.Issues.List(ctx, true, &github.IssueListOptions{
		Filter: "created",
		State:  "all",
		Sort:   "updated",
		ListOptions: github.ListOptions{
			PerPage: 20,
			Page:    page,
		},
	})

	for _, issue := range issues {
		if !issue.IsPullRequest() {
			continue
		}

		// fmt.Printf("%+v\n", issue)

		ownerName := *issue.Repository.Owner.Login

		if _, found := ghMap[ownerName]; !found {
			continue
		}

		repoName := *issue.Repository.Name

		if _, found := ghMap[ownerName][repoName]; !found {
			continue
		}

		localDirs := ghMap[ownerName][repoName]

		pr, _, err := client.PullRequests.Get(ctx, ownerName, repoName, *issue.Number)
		if err != nil {
			panic(err)
		}

		for _, localDir := range localDirs {
			syncPR(pr, localDir)
		}
	}
}

func syncPR(pr *github.PullRequest, localDir string) {
	if pr.GetUser().GetLogin() != "rickypai" {
		return
	}

	if err := branchExists(pr.Head.GetRef(), localDir); err != nil {
		return
	}

	if pr.GetState() == "closed" {
		syncClosedPR(pr, localDir)
	} else if pr.GetState() == "open" {
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
	issue, _ := execCommand(
		localDir,
		"twig", "--branch", pr.Head.GetRef(), "issue",
	)

	issueNum, _ := strconv.Atoi(string(strings.TrimSpace(string(issue))))

	if issueNum != pr.GetNumber() {
		log.Printf("setting PR for '%v'", pr.Head.GetRef())

		// set GH issue
		execCommand(
			localDir,
			"twig", "--branch", pr.Head.GetRef(), "issue", strconv.Itoa(pr.GetNumber()),
		)
	}

	// set diff branch
	execCommand(
		localDir,
		"twig", "--branch", pr.Head.GetRef(), "diff-branch", pr.Base.GetRef(),
	)

	if pr.Mergeable != nil && *pr.Mergeable == false {
		log.Printf("setting needs-rebase for '%v'", pr.Head.GetRef())

		// set needs-rebase
		execCommand(
			localDir,
			"twig", "--branch", pr.Head.GetRef(), "needs-rebase", "true",
		)
	} else {
		// unset needs-rebase
		execCommand(
			localDir,
			"twig", "--branch", pr.Head.GetRef(), "--unset", "needs-rebase",
		)
	}
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
