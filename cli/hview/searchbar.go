package hview

import (
	"strings"

	"github.com/gdamore/tcell"
	runewidth "github.com/mattn/go-runewidth"
	"github.com/rivo/tview"
)

// searchBarOption is one option that can be selected in a drop-down primitive.
type searchBarOption struct {
	Text     string // The text to be displayed in the drop-down.
	Selected func() // The (optional) callback for when this option was selected.
}

// SearchBar implements a selection widget...
type SearchBar struct {
	*Box

	// The options from which the user can choose.
	options []*searchBarOption

	// The index of the currently selected option. Negative if no option is
	// currently selected.
	currentOption int

	// Set to true if the options are visible and selectable.
	open bool

	// The runes typed so far to directly access one of the list items.
	prefix string

	// The list element for the options.
	list *tview.List

	// The text to be displayed before the input area.
	label string

	// The label color.
	labelColor tcell.Color

	// The background color of the input area.
	fieldBackgroundColor tcell.Color

	// The text color of the input area.
	fieldTextColor tcell.Color

	// The color for prefixes.
	prefixTextColor tcell.Color

	// The screen width of the label area. A value of 0 means use the width of
	// the label text.
	labelWidth int

	// The screen width of the input area. A value of 0 means extend as much as
	// possible.
	fieldWidth int

	// An optional function which is called when the user indicated that they
	// are done selecting options. The key which was pressed is provided (tab,
	// shift-tab, or escape).
	done func(tcell.Key)

	// A callback function set by the Form class and called when the user leaves
	// this form item.
	finished func(tcell.Key)
}

// NewDropDown returns a new drop-down.
func NewDropDown() *SearchBar {
	list := tview.NewList().ShowSecondaryText(false)
	list.SetMainTextColor(tview.Styles.PrimitiveBackgroundColor).
		SetSelectedTextColor(tview.Styles.PrimitiveBackgroundColor).
		SetSelectedBackgroundColor(tview.Styles.PrimaryTextColor).
		SetBackgroundColor(tview.Styles.MoreContrastBackgroundColor)

	d := &SearchBar{
		Box:                  NewBox(),
		currentOption:        -1,
		list:                 list,
		labelColor:           tview.Styles.SecondaryTextColor,
		fieldBackgroundColor: tview.Styles.ContrastBackgroundColor,
		fieldTextColor:       tview.Styles.PrimaryTextColor,
		prefixTextColor:      tview.Styles.ContrastSecondaryTextColor,
	}

	d.focus = d

	return d
}

// SetCurrentOption sets the index of the currently selected option. This may
// be a negative value to indicate that no option is currently selected.
func (d *SearchBar) SetCurrentOption(index int) *SearchBar {
	d.currentOption = index
	d.list.SetCurrentItem(index)
	return d
}

// GetCurrentOption returns the index of the currently selected option as well
// as its text. If no option was selected, -1 and an empty string is returned.
func (d *SearchBar) GetCurrentOption() (int, string) {
	var text string
	if d.currentOption >= 0 && d.currentOption < len(d.options) {
		text = d.options[d.currentOption].Text
	}
	return d.currentOption, text
}

// SetLabel sets the text to be displayed before the input area.
func (d *SearchBar) SetLabel(label string) *SearchBar {
	d.label = label
	return d
}

// GetLabel returns the text to be displayed before the input area.
func (d *SearchBar) GetLabel() string {
	return d.label
}

// SetLabelWidth sets the screen width of the label. A value of 0 will cause the
// primitive to use the width of the label string.
func (d *SearchBar) SetLabelWidth(width int) *SearchBar {
	d.labelWidth = width
	return d
}

// SetLabelColor sets the color of the label.
func (d *SearchBar) SetLabelColor(color tcell.Color) *SearchBar {
	d.labelColor = color
	return d
}

// SetFieldBackgroundColor sets the background color of the options area.
func (d *SearchBar) SetFieldBackgroundColor(color tcell.Color) *SearchBar {
	d.fieldBackgroundColor = color
	return d
}

// SetFieldTextColor sets the text color of the options area.
func (d *SearchBar) SetFieldTextColor(color tcell.Color) *SearchBar {
	d.fieldTextColor = color
	return d
}

// SetPrefixTextColor sets the color of the prefix string. The prefix string is
// shown when the user starts typing text, which directly selects the first
// option that starts with the typed string.
func (d *SearchBar) SetPrefixTextColor(color tcell.Color) *SearchBar {
	d.prefixTextColor = color
	return d
}

// SetFormAttributes sets attributes shared by all form items.
func (d *SearchBar) SetFormAttributes(labelWidth int, labelColor, bgColor, fieldTextColor, fieldBgColor tcell.Color) tview.FormItem {
	d.labelWidth = labelWidth
	d.labelColor = labelColor
	d.backgroundColor = bgColor
	d.fieldTextColor = fieldTextColor
	d.fieldBackgroundColor = fieldBgColor
	return d
}

// SetFieldWidth sets the screen width of the options area. A value of 0 means
// extend to as long as the longest option text.
func (d *SearchBar) SetFieldWidth(width int) *SearchBar {
	d.fieldWidth = width
	return d
}

// GetFieldWidth returns this primitive's field screen width.
func (d *SearchBar) GetFieldWidth() int {
	if d.fieldWidth > 0 {
		return d.fieldWidth
	}
	fieldWidth := 0
	for _, option := range d.options {
		width := tview.StringWidth(option.Text)
		if width > fieldWidth {
			fieldWidth = width
		}
	}
	return fieldWidth
}

// AddOption adds a new selectable option to this drop-down. The "selected"
// callback is called when this option was selected. It may be nil.
func (d *SearchBar) AddOption(text string, selected func()) *SearchBar {
	d.options = append(d.options, &searchBarOption{Text: text, Selected: selected})
	d.list.AddItem(text, "", 0, nil)
	return d
}

// SetOptions replaces all current options with the ones provided and installs
// one callback function which is called when one of the options is selected.
// It will be called with the option's text and its index into the options
// slice. The "selected" parameter may be nil.
func (d *SearchBar) SetOptions(texts []string, selected func(text string, index int)) *SearchBar {
	d.list.Clear()
	d.options = nil
	for index, text := range texts {
		func(t string, i int) {
			d.AddOption(text, func() {
				if selected != nil {
					selected(t, i)
				}
			})
		}(text, index)
	}
	return d
}

// SetDoneFunc sets a handler which is called when the user is done selecting
// options. The callback function is provided with the key that was pressed,
// which is one of the following:
//
//   - KeyEscape: Abort selection.
//   - KeyTab: Move to the next field.
//   - KeyBacktab: Move to the previous field.
func (d *SearchBar) SetDoneFunc(handler func(key tcell.Key)) *SearchBar {
	d.done = handler
	return d
}

// SetFinishedFunc sets a callback invoked when the user leaves this form item.
func (d *SearchBar) SetFinishedFunc(handler func(key tcell.Key)) tview.FormItem {
	d.finished = handler
	return d
}

// Draw draws this primitive onto the screen.
func (d *SearchBar) Draw(screen tcell.Screen) {
	d.Box.Draw(screen)

	// Prepare.
	x, y, width, height := d.GetInnerRect()
	rightLimit := x + width
	if height < 1 || rightLimit <= x {
		return
	}

	// Draw label.
	if d.labelWidth > 0 {
		labelWidth := d.labelWidth
		if labelWidth > rightLimit-x {
			labelWidth = rightLimit - x
		}
		tview.Print(screen, d.label, x, y, labelWidth, tview.AlignLeft, d.labelColor)
		x += labelWidth
	} else {
		_, drawnWidth := tview.Print(screen, d.label, x, y, rightLimit-x, tview.AlignLeft, d.labelColor)
		x += drawnWidth
	}

	// What's the longest option text?
	maxWidth := 0
	for _, option := range d.options {
		strWidth := tview.StringWidth(option.Text)
		if strWidth > maxWidth {
			maxWidth = strWidth
		}
	}

	// Draw selection area.
	fieldWidth := d.fieldWidth
	if fieldWidth == 0 {
		fieldWidth = maxWidth
	}
	if rightLimit-x < fieldWidth {
		fieldWidth = rightLimit - x
	}
	fieldStyle := tcell.StyleDefault.Background(d.fieldBackgroundColor)
	if d.GetFocusable().HasFocus() && !d.open {
		fieldStyle = fieldStyle.Background(d.fieldTextColor)
	}
	for index := 0; index < fieldWidth; index++ {
		screen.SetContent(x+index, y, ' ', nil, fieldStyle)
	}

	// Draw selected text.
	if d.open && len(d.prefix) > 0 {
		// Show the prefix.
		tview.Print(screen, d.prefix, x, y, fieldWidth, tview.AlignLeft, d.prefixTextColor)
		prefixWidth := runewidth.StringWidth(d.prefix)
		listItemText := d.options[d.list.GetCurrentItem()].Text
		if prefixWidth < fieldWidth && len(d.prefix) < len(listItemText) {
			tview.Print(screen, listItemText[len(d.prefix):], x+prefixWidth, y, fieldWidth-prefixWidth, tview.AlignLeft, d.fieldTextColor)
		}
	} else {
		if d.currentOption >= 0 && d.currentOption < len(d.options) {
			color := d.fieldTextColor
			// Just show the current selection.
			if d.GetFocusable().HasFocus() && !d.open {
				color = d.fieldBackgroundColor
			}
			tview.Print(screen, d.options[d.currentOption].Text, x, y, fieldWidth, tview.AlignLeft, color)
		}
	}

	// Draw options list.
	if d.HasFocus() && d.open {
		// We prefer to drop down but if there is no space, maybe drop up?
		lx := x
		ly := y + 1
		lwidth := maxWidth
		lheight := len(d.options)
		_, sheight := screen.Size()
		if ly+lheight >= sheight && ly-2 > lheight-ly {
			ly = y - lheight
			if ly < 0 {
				ly = 0
			}
		}
		if ly+lheight >= sheight {
			lheight = sheight - ly
		}
		d.list.SetRect(lx, ly, lwidth, lheight)
		d.list.Draw(screen)
	}
}

var Dictionnary = []string{
	"tag:",
	"test",
	"create ",
	"action!:",
	"resource:",
}

var Default = []string{
	"tag:",
	"action!:",
	"resource:",
}

func findDictionary(dic []string, prefix string) []string {
	res := make([]string, 0, 3)
	for _, el := range dic {
		if strings.HasPrefix(el, prefix) {
			res = append(res, el)
		}
	}
	return res
}

// InputHandler returns the handler for this primitive.
func (d *SearchBar) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return d.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
		// A helper function which selects an item in the drop-down list based on
		// the current prefix.
		evalPrefix := func() {
			if len(d.prefix) > 0 {
				search := findDictionary(Dictionnary, d.prefix)
				d.SetOptions(search, nil)
				if len(search) > 0 {
					d.list.SetCurrentItem(0)
				}
			} else {
				d.SetOptions(Default, nil)
			}
		}

		// Process key event.
		switch key := event.Key(); key {
		case tcell.KeyEnter, tcell.KeyRune, tcell.KeyDown:
			d.prefix = ""

			// If the first key was a letter already, it becomes part of the prefix.
			if r := event.Rune(); key == tcell.KeyRune && r != ' ' {
				d.prefix += string(r)
				evalPrefix()
			}

			// Hand control over to the list.
			d.open = true
			d.list.SetSelectedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
				// An option was selected. Close the list again.
				d.open = false
				setFocus(d)
				d.currentOption = index

				// Trigger "selected" event.
				if d.options[d.currentOption].Selected != nil {
					d.options[d.currentOption].Selected()
				}
			}).SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
				if event.Key() == tcell.KeyRune {
					d.prefix += string(event.Rune())
					evalPrefix()
				} else if event.Key() == tcell.KeyBackspace || event.Key() == tcell.KeyBackspace2 {
					if len(d.prefix) > 0 {
						r := []rune(d.prefix)
						d.prefix = string(r[:len(r)-1])
					}
					evalPrefix()
				} else {
					d.prefix = ""
				}
				return event
			})
			setFocus(d.list)
		case tcell.KeyEscape, tcell.KeyTab, tcell.KeyBacktab:
			if d.done != nil {
				d.done(key)
			}
			if d.finished != nil {
				d.finished(key)
			}
		}
	})
}

// Focus is called by the application when the primitive receives focus.
func (d *SearchBar) Focus(delegate func(p tview.Primitive)) {
	d.Box.Focus(delegate)
	if d.open {
		delegate(d.list)
	}
}

// HasFocus returns whether or not this primitive has focus.
func (d *SearchBar) HasFocus() bool {
	if d.open {
		return d.list.HasFocus()
	}
	return d.hasFocus
}
