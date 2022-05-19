package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	tfe "github.com/hashicorp/go-tfe"
)

type Client struct {
	*tfe.Client
}

type Workspace struct {
	ID        string
	Name      string
	AutoApply bool
	Runs      *Runs
}

type Workspaces []Workspace

type Run struct {
	ID     string
	Status tfe.RunStatus
}

type Runs []Run

const APPLY = "Apply"
const DISCARD = "Discard"
const CANCEL = "Cancel"
const SKIP = "Skip"

func main() {
	org := flag.String("org", "", "Terraform Cloud organization name")
	search := flag.String("search", "", "Workspace search term")
	noop := flag.Bool("noop", false, "Do not perform any action, only show what would happen")
	flag.Parse()

	if *org == "" {
		flag.Usage()
		os.Exit(1)
	}

	token := os.Getenv("TFE_TOKEN")
	if token == "" {
		fmt.Println("Environment variable TFE_TOKEN not found")
		os.Exit(1)
	}

	if *noop {
		log.Println("noop=true,message=no action will be taken")
	}

	client, err := newClient(token)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	workspaces, err := client.ListWorkspacesWithRunStatus(ctx, *org, *search, tfe.RunCostEstimated)
	if err != nil {
		log.Fatal(err)
	}

	for _, ws := range *workspaces {
		for idx, run := range *ws.Runs {
			if idx == 0 {
				switch run.Status {
				// this should be the only run, so confirm it if the workspace is
				// configured to auto-apply
				case tfe.RunCostEstimated:
					if ws.AutoApply {
						client.RunAction(ctx, APPLY, run.ID, ws.Name, *noop)
					}
				// this run will be triggered automatically
				case tfe.RunPending:
					client.RunAction(ctx, SKIP, run.ID, ws.Name, *noop)
				}
			} else {
				switch run.Status {
				case tfe.RunCostEstimated:
					client.RunAction(ctx, DISCARD, run.ID, ws.Name, *noop)
				case tfe.RunPending:
					client.RunAction(ctx, CANCEL, run.ID, ws.Name, *noop)
				}
			}
		}
	}
}

func newClient(token string) (*Client, error) {
	config := &tfe.Config{
		Token: token,
	}

	client, err := tfe.NewClient(config)
	if err != nil {
		return &Client{}, err
	}

	return &Client{client}, nil
}

func (c *Client) ListWaitingRuns(ctx context.Context, workspaceID string) (*Runs, error) {
	var runs Runs
	n := 0
	for {
		opts := &tfe.RunListOptions{
			ListOptions: tfe.ListOptions{
				PageNumber: n,
			},
		}
		r, err := c.Runs.List(ctx, workspaceID, opts)
		if err != nil {
			return &runs, err
		}

		for _, run := range r.Items {
			if run.Status == tfe.RunCostEstimated || run.Status == tfe.RunPending {
				runs = append(runs, Run{
					ID:     run.ID,
					Status: run.Status,
				})
			}
		}

		// Only continue if the last run on the page is pending
		if len(r.Items) > 0 && r.Items[len(r.Items)-1].Status == tfe.RunPending && r.NextPage > n {
			n = r.NextPage
		} else {
			return &runs, nil
		}
	}
}

func (c *Client) ListWorkspacesWithRunStatus(ctx context.Context, org string, search string, runStatus tfe.RunStatus) (*Workspaces, error) {
	var workspaces Workspaces
	n := 0
	for {
		opts := &tfe.WorkspaceListOptions{
			ListOptions: tfe.ListOptions{
				PageNumber: n,
			},
			Search: search,
			Include: []tfe.WSIncludeOpt{
				"current_run",
			},
		}
		w, err := c.Workspaces.List(ctx, org, opts)
		if err != nil {
			return &workspaces, err
		}

		for _, ws := range w.Items {
			if ws.CurrentRun != nil {
				if ws.CurrentRun.Status == runStatus {
					runs, err := c.ListWaitingRuns(ctx, ws.ID)
					if err != nil {
						return &workspaces, err
					}
					workspaces = append(workspaces, Workspace{
						ID:        ws.ID,
						Name:      ws.Name,
						AutoApply: ws.AutoApply,
						Runs:      runs,
					})
				}
			}
		}

		if w.NextPage > n {
			n = w.NextPage
		} else {
			return &workspaces, nil
		}
	}
}

func (c *Client) RunAction(ctx context.Context, action string, runID string, workspace_name string, noop bool) error {
	log.Printf("run_id=%s,workspace_name=%s,action=%s", runID, workspace_name, strings.ToLower(action))

	if noop {
		return nil
	}

	comment := fmt.Sprintf("%sing run automatically", action)
	switch action {
	case APPLY:
		return c.Runs.Apply(ctx, runID, tfe.RunApplyOptions{
			Comment: &comment,
		})
	case DISCARD:
		return c.Runs.Discard(ctx, runID, tfe.RunDiscardOptions{
			Comment: &comment,
		})
	case CANCEL:
		return c.Runs.Cancel(ctx, runID, tfe.RunCancelOptions{
			Comment: &comment,
		})
	}

	return nil
}
