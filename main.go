package main

import (
	"github.com/DEEP-IMPACT-AG/hyperdrive/hview"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
	"log"
)

func main2() {
	app := tview.NewApplication()
	menu := hview.NewDropDown()
	table := tview.NewTable()
	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(populateMenu(menu), 1, 0, true).
		AddItem(table.SetDoneFunc(func(key tcell.Key) {
		app.SetFocus(menu)
	}), 0, 1, false)

	if err := app.SetRoot(flex, true).Run(); err != nil {
		log.Fatal(err)
	}
}

func populateMenu(menu *hview.SearchBar) *hview.SearchBar {
	return menu.
		AddOption("tag:", nil).
		AddOption("action!:", nil).
		AddOption("resource:", nil)
}

func main() {
	cfg, err := external.LoadDefaultAWSConfig(
		external.WithSharedConfigProfile("libra-dev"),
	)
	if err != nil {
		log.Fatal(err.Error())
	}
	cfs := cloudformation.New(cfg)
	app := tview.NewApplication()
	pages := tview.NewPages()
	details := tview.NewPages()
	menu := tview.NewList()
	submenu := tview.NewList().SetDoneFunc(func() {
		app.SetFocus(menu)
	})
	menu.
		AddItem("Browse", "Browse AWS", 'b', func() {
		submenu.Clear()
		submenu.AddItem("Cloudformation", "", 'c',
			func() {
				pages.AddAndSwitchToPage("browser", fetchCFS(cfs, app, submenu, pages, details), true)
				app.SetFocus(pages)
			})
		app.SetFocus(submenu)
	}).
		AddItem("Create", "Create Resources", 'c', func() {
		submenu.Clear()
		submenu.AddItem("DefaultVPC", "", 'v', nil)
		submenu.AddItem("HostedZone", "", 'z', nil)
		app.SetFocus(submenu)
	}).
		AddItem("Quit", "Press to exit", 'q', func() {
		app.Stop()
	})
	menus := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(menu, 0, 1, true).
		AddItem(submenu, 0, 1, true)
	content := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(pages, 0, 1, false).
		AddItem(details, 0, 1, false)
	flex := tview.NewFlex().
		AddItem(menus, 30, 1, true).
		AddItem(content, 0, 1, false)
	if err := app.SetRoot(flex, true).Run(); err != nil {
		panic(err)
	}
}

func fetchCFS(cfs *cloudformation.CloudFormation, app *tview.Application, submenu *tview.List, pages *tview.Pages, details *tview.Pages) tview.Primitive {
	table := tview.NewTable()
	table.
		SetCell(0, 0, headerCell("Stack Name")).
		SetCell(0, 1, headerCell("Created Time")).
		SetCell(0, 2, headerCell("Status")).
		SetCell(0, 3, headerCell("Description")).
		SetFixed(1, 0).
		SetSelectable(true, true).
		SetSelectedFunc(func(row, column int) {
		if row > 0 {
			stackName := table.GetCell(row, 0).Text
			request := cloudformation.DescribeStacksInput{
				StackName: &stackName,
			}
			res, err := cfs.DescribeStacksRequest(&request).Send()
			if err != nil {
				panic(err)
			}
			stack := res.Stacks[0]
			form := tview.NewForm().
				AddInputField("Stack Name", display(stack.StackName), 40, nil, nil).
				AddInputField("Stack ID", display(stack.StackId), 120, nil, nil).
				AddInputField("Status", string(stack.StackStatus), 40, nil, nil).
				AddCheckbox("Term Protection", *stack.EnableTerminationProtection, nil).
				AddInputField("Status Reason", display(stack.StackStatusReason), 120, nil, nil).
				AddInputField("IAM Role", display(stack.RoleARN), 120, nil, nil).
				AddButton("Switch Term Protection", func() {
				setTerminationProtection(cfs, *stack.StackId, !*stack.EnableTerminationProtection)
			}).
				AddButton("Delete Stack", func() {
				deleteStack(cfs, *stack.StackId)
			}).
				SetCancelFunc(func() {
				app.SetFocus(pages)
			})
			details.AddAndSwitchToPage("details", form, true)
			app.SetFocus(details)
		}
	}).SetDoneFunc(func(key tcell.Key) {
		if key != tcell.KeyEnter {
			app.SetFocus(submenu)
		}
	})
	go func() {
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
	return table
}

func setTerminationProtection(cfs *cloudformation.CloudFormation, stackId string, enabled bool) {
	_, err := cfs.UpdateTerminationProtectionRequest(&cloudformation.UpdateTerminationProtectionInput{
		StackName: &stackId,
		EnableTerminationProtection: &enabled,
	}).Send()
	if err != nil {
		panic(err)
	}
}

func deleteStack(cfs *cloudformation.CloudFormation, stackId string) {
	_, err := cfs.DeleteStackRequest(&cloudformation.DeleteStackInput{
		StackName: &stackId,
	}).Send()
	if err != nil {
		panic(err)
	}
}

func display(text *string) string {
	if text == nil {
		return ""
	}
	return *text
}

func headerCell(text string) *tview.TableCell {
	return tview.NewTableCell(text).
		SetAlign(tview.AlignLeft)
}
