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
			if pr.GetUser().GetLogin() != "rickypai" {
				continue
			}

			// delete closed branches
			if pr.GetState() == "closed" {
				cmd := exec.Command("git", "show", pr.Head.GetRef())
				cmd.Dir = godelDir
				_, err := cmd.Output()
				if err != nil {
					continue
				}

				log.Printf("deleting '%v'", pr.Head.GetRef())

				execCommandDirs(
					[]string{
						godelDir,
						godel2Dir,
					},
					"git", "branch", "-D", pr.Head.GetRef(),
				)
			}

			if pr.GetState() == "open" {
				cmd := exec.Command("git", "show", pr.Head.GetRef())
				cmd.Dir = godelDir
				_, err := cmd.Output()
				if err != nil {
					continue
				}

				log.Printf("setting PR for '%v'", pr.Head.GetRef())

				execCommandDirs(
					[]string{
						godelDir,
						godel2Dir,
					},
					"twig", "--branch", pr.Head.GetRef(), "issue", strconv.Itoa(pr.GetNumber()),
				)

				execCommandDirs(
					[]string{
						godelDir,
						godel2Dir,
					},
					"twig", "--branch", pr.Head.GetRef(), "diff-branch", pr.Base.GetRef(),
				)
			}
		}
	}
}

const godelDir = "/Users/ricky/workspace/src/github.com/vsco/godel"
const godel2Dir = "/Users/ricky/workspace/src/github.com/vsco/godel2"

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
