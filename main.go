package main

import (
	"context"
	"log"
	"os"
	"os/exec"
	"strconv"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

const godel1Dir = "/Users/ricky/workspace/src/github.com/vsco/godel"
const godel2Dir = "/Users/ricky/workspace/src/github.com/vsco/godel2"

var godelDirs = []string{
	godel1Dir,
	godel2Dir,
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

	opt := github.PullRequestListOptions{
		State:       "all",
		ListOptions: github.ListOptions{},
	}

	for i := 1; i < 20; i++ {
		opt.ListOptions.Page = i

		prs, _, err := client.PullRequests.List(ctx, "vsco", "godel", &opt)
		if err != nil {
			panic(err)
		}

		for _, pr := range prs {
			for _, godelDir := range godelDirs {
				syncPR(pr, godelDir)
			}
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

func execCommandDirs(dirs []string, name string, arg ...string) {
	for _, dir := range dirs {
		execCommand(dir, name, arg...)
	}
}

func execCommand(dir string, name string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	cmd.Dir = dir
	_, err := cmd.Output()
	return err
}
