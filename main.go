package main

import (
	"github.com/rivo/tview"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"log"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
)

func main() {
	cfg, err := external.LoadDefaultAWSConfig(
		external.WithSharedConfigProfile("libra-dev"),
	)
	if err != nil {
		log.Fatal(err.Error())
	}
	app := tview.NewApplication()
	pages := tview.NewPages()
	list := tview.NewList().
		AddItem("Browse", "Browse AWS", 'b', func() {
			if !pages.HasPage("browser") {
				table := fetchCFS(cfg, app)
				pages.AddPage("browser", table, true, false)
			}
			pages.SwitchToPage("browser")
		}).
		AddItem("Create", "Create Resources", 'c', func() {
			pages.SwitchToPage("create")
		}).
		AddItem("Quit", "Press to exit", 'q', func() {
			app.Stop()
		})
	flex := tview.NewFlex().
		AddItem(list, 30, 1, true).
		AddItem(pages, 0, 1, false)
	if err := app.SetRoot(flex, true).Run(); err != nil {
		panic(err)
	}
}
func fetchCFS(cfg aws.Config, app *tview.Application) tview.Primitive {
	table := tview.NewTable()
	table.
		SetCell(0, 0, headerCell("Stack Name")).
		SetCell(0, 1, headerCell("Created Time")).
		SetCell(0, 2, headerCell("Status")).
		SetCell(0, 3, headerCell("Description")).
		SetFixed(1, 0).
		SetSelectable(true, false)
	go func() {
		cfs := cloudformation.New(cfg)
		request := cloudformation.DescribeStacksInput{}
		res, err := cfs.DescribeStacksRequest(&request).Send()
		if err != nil {
			panic(err)
		}
		for i, stack := range res.Stacks {
			row := i + 1
			table.
				SetCell(row, 0, tview.NewTableCell(*stack.StackName)).
				SetCell(row, 1, tview.NewTableCell(stack.CreationTime.String())).
				SetCell(row, 2, tview.NewTableCell(string(stack.StackStatus)))
		}
		app.Draw()
	}()
	return table;
}

func headerCell(text string) *tview.TableCell {
	return tview.NewTableCell(text).
		SetAlign(tview.AlignLeft)
}
