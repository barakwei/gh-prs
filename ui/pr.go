package ui

import (
	"encoding/json"
	"fmt"
	"gh-prs/utils"
	"log"
	"os/exec"
	"time"

	"github.com/charmbracelet/lipgloss"
)

const (
	JsonFields = "title,mergeable,author,url,additions,deletions,headRefName,headRepository,isDraft,number,reviewDecision,state,statusCheckRollup,updatedAt"
	Limit      = "20"
)

type PullRequest struct {
	Number           int
	Title            string
	Author           Author
	UpdatedAt        time.Time
	Url              string
	State            string
	Mergeable        string
	ReviewDecision   string
	Additions        int
	Deletions        int
	HeadRefName      string
	Repository       Repository
	IsDraft          bool
	StatusCheckState string
}

type Author struct {
	Login string
}

type Repository struct {
	NameWithOwner string
	Name          string
}

type StatusCheck struct {
	Name        string
	Context     string
	State       string
	Status      string
	Conclusion  string
	StartedAt   string
	CompletedAt string
	DetailsUrl  string
	TargetUrl   string
}

type PullRequestsFetchedMsg struct {
	SectionId int
	Prs       []PullRequest
}

func fetchPullRequestsSearchQuery(query string) ([]PullRequest, error) {
	const jqFilter = `[.data.search.edges[] | (. * .node) | (del(.node)) | (. * (.commits.nodes[].commit.statusCheckRollup | .["statusCheckState"] = .state | del(.state))) | del(.commits)]`
	const queryTemplate = `query={
  search(query: "%s", type: ISSUE, first: 30) {
    issueCount
    edges {
      node {
        ... on PullRequest {
          number
          title
          author {
            login
          }
		  repository {
			name
			nameWithOwner
		  }
          url
          isDraft
          mergeable
          additions
          deletions
          baseRefName
          headRefName
          reviewDecision
          state
          updatedAt
          commits(last: 1) {
            nodes {
              commit {
                statusCheckRollup {
                  state
                }
              }
            }
          }
        }
      }
    }
  }
}
`

	var k = fmt.Sprintf("gh api graphql --paginate -f %s", fmt.Sprintf(queryTemplate, query))
	fmt.Println(k)

	out, err := exec.Command(
		//"echo",
		"gh",
		"api",
		"graphql",
		"--paginate",
		"--jq",
		jqFilter,
		"-f",
		fmt.Sprintf(queryTemplate, query),
	).Output()

	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	var prs []PullRequest
	if err := json.Unmarshal(out, &prs); err != nil {
		log.Fatal(err)
		return nil, err
	}

	return prs, nil
}

func (pr PullRequest) renderReviewStatus(isSelected bool) string {
	reviewCellStyle := makeRuneCellStyle(isSelected)
	if pr.ReviewDecision == "APPROVED" {
		return reviewCellStyle.Foreground(lipgloss.Color("42")).Render("")
	}

	if pr.ReviewDecision == "CHANGES_REQUESTED" {
		return reviewCellStyle.Foreground(lipgloss.Color("196")).Render("")
	}

	return reviewCellStyle.Faint(true).Render("")
}

func (pr PullRequest) renderMergeableStatus(isSelected bool) string {
	mergeCellStyle := makeRuneCellStyle(isSelected)
	switch pr.Mergeable {
	case "MERGEABLE":
		return mergeCellStyle.Foreground(lipgloss.Color("42")).Render("")
	case "CONFLICTING":
		return mergeCellStyle.Foreground(lipgloss.Color("196")).Render("")
	case "UNKNOWN":
		fallthrough
	default:
		return mergeCellStyle.Faint(true).Render("")
	}
}

func (pr PullRequest) renderCiStatus(isSelected bool) string {
	ciCellStyle := makeRuneCellStyle(isSelected).Width(ciCellWidth)
	if pr.StatusCheckState == "SUCCESS" {
		return ciCellStyle.Foreground(lipgloss.Color("42")).Render("")
	}

	if pr.StatusCheckState == "PENDING" {
		return ciCellStyle.Foreground(lipgloss.Color("214")).Render("")
	}

	return ciCellStyle.Foreground(lipgloss.Color("196")).Render("")
}

func (pr PullRequest) renderLines(isSelected bool) string {
	separator := makeCellStyle(isSelected).Faint(true).PaddingLeft(1).PaddingRight(1).Render("/")
	added := makeCellStyle(isSelected).Render(fmt.Sprintf("%d", pr.Additions))
	deletions := 0
	if pr.Deletions > 0 {
		deletions = pr.Deletions
	}
	removed := makeCellStyle(isSelected).Render(
		fmt.Sprintf("-%d", deletions),
	)

	return makeCellStyle(isSelected).
		Width(linesCellWidth).
		Render(lipgloss.JoinHorizontal(lipgloss.Left, added, separator, removed))
}

func (pr PullRequest) renderTitle(viewportWidth int, isSelected bool) string {
	number := lipgloss.NewStyle().Width(6).Faint(true).Render(
		fmt.Sprintf("#%s", fmt.Sprintf("%d", pr.Number)),
	)

	totalWidth := getTitleWidth(viewportWidth)
	title := lipgloss.NewStyle().Render(pr.Title)

	return makeCellStyle(isSelected).
		Width(totalWidth - 1).
		Render(lipgloss.JoinHorizontal(lipgloss.Left, title, number))
}

func (pr PullRequest) renderAuthor(isSelected bool) string {
	return makeCellStyle(isSelected).Width(prAuthorCellWidth).Render(
		utils.TruncateString(pr.Author.Login, prAuthorCellWidth-cellPadding),
	)
}

func (pr PullRequest) renderRepoName(isSelected bool) string {
	return makeCellStyle(isSelected).
		Width(prRepoCellWidth).
		Render(fmt.Sprintf("%-20s", utils.TruncateString(pr.Repository.Name, 20)))
}

func (pr PullRequest) renderUpdateAt(isSelected bool) string {
	return makeCellStyle(isSelected).
		Width(updatedAtCellWidth).
		Render(utils.TimeElapsed(pr.UpdatedAt))
}

func renderSelectionPointer(isSelected bool) string {
	return makeRuneCellStyle(isSelected).
		Width(emptyCellWidth).
		Render(func() string {
			if isSelected {
				return selectionPointerStyle.Render("")
			} else {
				return " "
			}
		}())
}

func (pr PullRequest) render(isSelected bool, viewPortWidth int) string {
	selectionPointerCell := renderSelectionPointer(isSelected)
	reviewCell := pr.renderReviewStatus(isSelected)
	mergeableCell := pr.renderMergeableStatus(isSelected)
	ciCell := pr.renderCiStatus(isSelected)
	linesCell := pr.renderLines(isSelected)
	prTitleCell := pr.renderTitle(viewPortWidth, isSelected)
	prAuthorCell := pr.renderAuthor(isSelected)
	prRepoCell := pr.renderRepoName(isSelected)
	updatedAtCell := pr.renderUpdateAt(isSelected)

	rowStyle := pullRequestStyle.Copy()
	return rowStyle.
		Width(viewPortWidth).
		MaxWidth(viewPortWidth).
		Render(lipgloss.JoinHorizontal(lipgloss.Left,
			selectionPointerCell,
			reviewCell,
			prTitleCell,
			mergeableCell,
			ciCell,
			linesCell,
			prAuthorCell,
			prRepoCell,
			updatedAtCell,
		))
}
