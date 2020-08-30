package cview

import (
	"fmt"
	"strings"
	"sync"

	"github.com/gdamore/tcell/v2"
)

// listItem represents one item in a List.
type listItem struct {
	Enabled       bool   // Whether or not the list item is selectable.
	MainText      string // The main text of the list item.
	SecondaryText string // A secondary text to be shown underneath the main text.
	Shortcut      rune   // The key to select the list item directly, 0 if there is no shortcut.
	Selected      func() // The optional function which is called when the item is selected.
}

// List displays rows of items, each of which can be selected.
type List struct {
	*Box
	*ContextMenu

	// The items of the list.
	items []*listItem

	// The index of the currently selected item.
	currentItem int

	// Whether or not to show the secondary item texts.
	showSecondaryText bool

	// The item main text color.
	mainTextColor tcell.Color

	// The item secondary text color.
	secondaryTextColor tcell.Color

	// The item shortcut text color.
	shortcutColor tcell.Color

	// The text color for selected items.
	selectedTextColor tcell.Color

	// The style attributes for selected items.
	selectedTextAttributes tcell.AttrMask

	// Visibility of the scroll bar.
	scrollBarVisibility ScrollBarVisibility

	// The scroll bar color.
	scrollBarColor tcell.Color

	// The background color for selected items.
	selectedBackgroundColor tcell.Color

	// If true, the selection is only shown when the list has focus.
	selectedFocusOnly bool

	// If true, the selection must remain visible when scrolling.
	selectedAlwaysVisible bool

	// If true, the selection must remain centered when scrolling.
	selectedAlwaysCentered bool

	// If true, the entire row is highlighted when selected.
	highlightFullLine bool

	// Whether or not navigating the list will wrap around.
	wrapAround bool

	// Whether or not hovering over an item will highlight it.
	hover bool

	// The number of list items skipped at the top before the first item is drawn.
	offset int

	// An optional function which is called when the user has navigated to a list
	// item.
	changed func(index int, mainText, secondaryText string, shortcut rune)

	// An optional function which is called when a list item was selected. This
	// function will be called even if the list item defines its own callback.
	selected func(index int, mainText, secondaryText string, shortcut rune)

	// An optional function which is called when the user presses the Escape key.
	done func()

	// The height of the list the last time it was drawn.
	height int

	sync.RWMutex
}

// NewList returns a new form.
func NewList() *List {
	l := &List{
		Box:                     NewBox(),
		showSecondaryText:       true,
		scrollBarVisibility:     ScrollBarAuto,
		mainTextColor:           Styles.PrimaryTextColor,
		secondaryTextColor:      Styles.TertiaryTextColor,
		shortcutColor:           Styles.SecondaryTextColor,
		selectedTextColor:       Styles.PrimitiveBackgroundColor,
		scrollBarColor:          Styles.ScrollBarColor,
		selectedBackgroundColor: Styles.PrimaryTextColor,
	}

	l.ContextMenu = NewContextMenu(l)
	l.focus = l

	return l
}

// SetCurrentItem sets the currently selected item by its index, starting at 0
// for the first item. If a negative index is provided, items are referred to
// from the back (-1 = last item, -2 = second-to-last item, and so on). Out of
// range indices are clamped to the beginning/end.
//
// Calling this function triggers a "changed" event if the selection changes.
func (l *List) SetCurrentItem(index int) *List {
	l.Lock()

	if index < 0 {
		index = len(l.items) + index
	}
	if index >= len(l.items) {
		index = len(l.items) - 1
	}
	if index < 0 {
		index = 0
	}

	previousItem := l.currentItem
	l.currentItem = index

	l.updateOffset()

	if index != previousItem && l.changed != nil {
		item := l.items[index]
		l.Unlock()
		l.changed(index, item.MainText, item.SecondaryText, item.Shortcut)
		l.Lock()
	}

	l.Unlock()
	return l
}

// GetCurrentItem returns the index of the currently selected list item,
// starting at 0 for the first item.
func (l *List) GetCurrentItem() int {
	l.RLock()
	defer l.RUnlock()

	return l.currentItem
}

// RemoveItem removes the item with the given index (starting at 0) from the
// list. If a negative index is provided, items are referred to from the back
// (-1 = last item, -2 = second-to-last item, and so on). Out of range indices
// are clamped to the beginning/end, i.e. unless the list is empty, an item is
// always removed.
//
// The currently selected item is shifted accordingly. If it is the one that is
// removed, a "changed" event is fired.
func (l *List) RemoveItem(index int) *List {
	l.Lock()

	if len(l.items) == 0 {
		l.Unlock()
		return l
	}

	// Adjust index.
	if index < 0 {
		index = len(l.items) + index
	}
	if index >= len(l.items) {
		index = len(l.items) - 1
	}
	if index < 0 {
		index = 0
	}

	// Remove item.
	l.items = append(l.items[:index], l.items[index+1:]...)

	// If there is nothing left, we're done.
	if len(l.items) == 0 {
		l.Unlock()
		return l
	}

	// Shift current item.
	previousItem := l.currentItem
	if l.currentItem >= index && l.currentItem > 0 {
		l.currentItem--
	}

	// Fire "changed" event for removed items.
	if previousItem == index && l.changed != nil {
		item := l.items[l.currentItem]
		l.Unlock()
		l.changed(l.currentItem, item.MainText, item.SecondaryText, item.Shortcut)
	} else {
		l.Unlock()
	}

	return l
}

// SetOffset sets the number of list items skipped at the top before the first
// item is drawn.
func (l *List) SetOffset(offset int) *List {
	l.Lock()
	defer l.Unlock()

	if offset < 0 {
		offset = 0
	}

	l.offset = offset
	return l
}

// GetOffset returns the number of list items skipped at the top before the
// first item is drawn.
func (l *List) GetOffset() int {
	l.Lock()
	defer l.Unlock()

	return l.offset
}

// SetMainTextColor sets the color of the items' main text.
func (l *List) SetMainTextColor(color tcell.Color) *List {
	l.Lock()
	defer l.Unlock()

	l.mainTextColor = color
	return l
}

// SetSecondaryTextColor sets the color of the items' secondary text.
func (l *List) SetSecondaryTextColor(color tcell.Color) *List {
	l.Lock()
	defer l.Unlock()

	l.secondaryTextColor = color
	return l
}

// SetShortcutColor sets the color of the items' shortcut.
func (l *List) SetShortcutColor(color tcell.Color) *List {
	l.Lock()
	defer l.Unlock()

	l.shortcutColor = color
	return l
}

// SetSelectedTextColor sets the text color of selected items.
func (l *List) SetSelectedTextColor(color tcell.Color) *List {
	l.Lock()
	defer l.Unlock()

	l.selectedTextColor = color
	return l
}

// SetSelectedTextAttributes sets the style attributes of selected items.
func (l *List) SetSelectedTextAttributes(attr tcell.AttrMask) *List {
	l.Lock()
	defer l.Unlock()

	l.selectedTextAttributes = attr
	return l
}

// SetSelectedBackgroundColor sets the background color of selected items.
func (l *List) SetSelectedBackgroundColor(color tcell.Color) *List {
	l.Lock()
	defer l.Unlock()

	l.selectedBackgroundColor = color
	return l
}

// SetSelectedFocusOnly sets a flag which determines when the currently selected
// list item is highlighted. If set to true, selected items are only highlighted
// when the list has focus. If set to false, they are always highlighted.
func (l *List) SetSelectedFocusOnly(focusOnly bool) *List {
	l.Lock()
	defer l.Unlock()

	l.selectedFocusOnly = focusOnly
	return l
}

// SetSelectedAlwaysVisible sets a flag which determines whether the currently
// selected list item must remain visible when scrolling.
func (l *List) SetSelectedAlwaysVisible(alwaysVisible bool) *List {
	l.Lock()
	defer l.Unlock()

	l.selectedAlwaysVisible = alwaysVisible
	return l
}

// SetSelectedAlwaysCentered sets a flag which determines whether the currently
// selected list item must remain centered when scrolling.
func (l *List) SetSelectedAlwaysCentered(alwaysCentered bool) *List {
	l.Lock()
	defer l.Unlock()

	l.selectedAlwaysCentered = alwaysCentered
	return l
}

// SetHighlightFullLine sets a flag which determines whether the colored
// background of selected items spans the entire width of the view. If set to
// true, the highlight spans the entire view. If set to false, only the text of
// the selected item from beginning to end is highlighted.
func (l *List) SetHighlightFullLine(highlight bool) *List {
	l.Lock()
	defer l.Unlock()

	l.highlightFullLine = highlight
	return l
}

// ShowSecondaryText determines whether or not to show secondary item texts.
func (l *List) ShowSecondaryText(show bool) *List {
	l.Lock()
	defer l.Unlock()

	l.showSecondaryText = show
	return l
}

// SetScrollBarVisibility specifies the display of the scroll bar.
func (l *List) SetScrollBarVisibility(visibility ScrollBarVisibility) *List {
	l.Lock()
	defer l.Unlock()

	l.scrollBarVisibility = visibility
	return l
}

// SetScrollBarColor sets the color of the scroll bar.
func (l *List) SetScrollBarColor(color tcell.Color) *List {
	l.Lock()
	defer l.Unlock()

	l.scrollBarColor = color
	return l
}

// SetHover sets the flag that determines whether hovering over an item will
// highlight it (without triggering callbacks set with SetSelectedFunc).
func (l *List) SetHover(hover bool) *List {
	l.Lock()
	defer l.Unlock()

	l.hover = hover
	return l
}

// SetWrapAround sets the flag that determines whether navigating the list will
// wrap around. That is, navigating downwards on the last item will move the
// selection to the first item (similarly in the other direction). If set to
// false, the selection won't change when navigating downwards on the last item
// or navigating upwards on the first item.
func (l *List) SetWrapAround(wrapAround bool) *List {
	l.Lock()
	defer l.Unlock()

	l.wrapAround = wrapAround
	return l
}

// SetChangedFunc sets the function which is called when the user navigates to
// a list item. The function receives the item's index in the list of items
// (starting with 0), its main text, secondary text, and its shortcut rune.
//
// This function is also called when the first item is added or when
// SetCurrentItem() is called.
func (l *List) SetChangedFunc(handler func(index int, mainText string, secondaryText string, shortcut rune)) *List {
	l.Lock()
	defer l.Unlock()

	l.changed = handler
	return l
}

// SetSelectedFunc sets the function which is called when the user selects a
// list item by pressing Enter on the current selection. The function receives
// the item's index in the list of items (starting with 0), its main text,
// secondary text, and its shortcut rune.
func (l *List) SetSelectedFunc(handler func(int, string, string, rune)) *List {
	l.Lock()
	defer l.Unlock()

	l.selected = handler
	return l
}

// SetDoneFunc sets a function which is called when the user presses the Escape
// key.
func (l *List) SetDoneFunc(handler func()) *List {
	l.Lock()
	defer l.Unlock()

	l.done = handler
	return l
}

// AddItem calls InsertItem() with an index of -1.
func (l *List) AddItem(mainText, secondaryText string, shortcut rune, selected func()) *List {
	l.InsertItem(-1, mainText, secondaryText, shortcut, selected)
	return l
}

// InsertItem adds a new item to the list at the specified index. An index of 0
// will insert the item at the beginning, an index of 1 before the second item,
// and so on. An index of GetItemCount() or higher will insert the item at the
// end of the list. Negative indices are also allowed: An index of -1 will
// insert the item at the end of the list, an index of -2 before the last item,
// and so on. An index of -GetItemCount()-1 or lower will insert the item at the
// beginning.
//
// An item has a main text which will be highlighted when selected. It also has
// a secondary text which is shown underneath the main text (if it is set to
// visible) but which may remain empty.
//
// The shortcut is a key binding. If the specified rune is entered, the item
// is selected immediately. Set to 0 for no binding.
//
// The "selected" callback will be invoked when the user selects the item. You
// may provide nil if no such callback is needed or if all events are handled
// through the selected callback set with SetSelectedFunc().
//
// The currently selected item will shift its position accordingly. If the list
// was previously empty, a "changed" event is fired because the new item becomes
// selected.
func (l *List) InsertItem(index int, mainText, secondaryText string, shortcut rune, selected func()) *List {
	l.Lock()

	item := &listItem{
		Enabled:       true,
		MainText:      mainText,
		SecondaryText: secondaryText,
		Shortcut:      shortcut,
		Selected:      selected,
	}

	// Shift index to range.
	if index < 0 {
		index = len(l.items) + index + 1
	}
	if index < 0 {
		index = 0
	} else if index > len(l.items) {
		index = len(l.items)
	}

	// Shift current item.
	if l.currentItem < len(l.items) && l.currentItem >= index {
		l.currentItem++
	}

	// Insert item (make space for the new item, then shift and insert).
	l.items = append(l.items, nil)
	if index < len(l.items)-1 { // -1 because l.items has already grown by one item.
		copy(l.items[index+1:], l.items[index:])
	}
	l.items[index] = item

	// Fire a "change" event for the first item in the list.
	if len(l.items) == 1 && l.changed != nil {
		item := l.items[0]
		l.Unlock()
		l.changed(0, item.MainText, item.SecondaryText, item.Shortcut)
	} else {
		l.Unlock()
	}

	return l
}

// GetItemCount returns the number of items in the list.
func (l *List) GetItemCount() int {
	l.RLock()
	defer l.RUnlock()

	return len(l.items)
}

// GetItemText returns an item's texts (main and secondary). Panics if the index
// is out of range.
func (l *List) GetItemText(index int) (main, secondary string) {
	l.RLock()
	defer l.RUnlock()

	return l.items[index].MainText, l.items[index].SecondaryText
}

// SetItemText sets an item's main and secondary text. Panics if the index is
// out of range.
func (l *List) SetItemText(index int, main, secondary string) *List {
	l.Lock()
	defer l.Unlock()

	item := l.items[index]
	item.MainText = main
	item.SecondaryText = secondary
	return l
}

// SetItemEnabled sets whether an item is selectable. Panics if the index is
// out of range.
func (l *List) SetItemEnabled(index int, enabled bool) *List {
	l.Lock()
	defer l.Unlock()

	item := l.items[index]
	item.Enabled = enabled
	return l
}

// FindItems searches the main and secondary texts for the given strings and
// returns a list of item indices in which those strings are found. One of the
// two search strings may be empty, it will then be ignored. Indices are always
// returned in ascending order.
//
// If mustContainBoth is set to true, mainSearch must be contained in the main
// text AND secondarySearch must be contained in the secondary text. If it is
// false, only one of the two search strings must be contained.
//
// Set ignoreCase to true for case-insensitive search.
func (l *List) FindItems(mainSearch, secondarySearch string, mustContainBoth, ignoreCase bool) (indices []int) {
	l.RLock()
	defer l.RUnlock()

	if mainSearch == "" && secondarySearch == "" {
		return
	}

	if ignoreCase {
		mainSearch = strings.ToLower(mainSearch)
		secondarySearch = strings.ToLower(secondarySearch)
	}

	for index, item := range l.items {
		mainText := item.MainText
		secondaryText := item.SecondaryText
		if ignoreCase {
			mainText = strings.ToLower(mainText)
			secondaryText = strings.ToLower(secondaryText)
		}

		// strings.Contains() always returns true for a "" search.
		mainContained := strings.Contains(mainText, mainSearch)
		secondaryContained := strings.Contains(secondaryText, secondarySearch)
		if mustContainBoth && mainContained && secondaryContained ||
			!mustContainBoth && (mainText != "" && mainContained || secondaryText != "" && secondaryContained) {
			indices = append(indices, index)
		}
	}

	return
}

// Clear removes all items from the list.
func (l *List) Clear() *List {
	l.Lock()
	defer l.Unlock()

	l.items = nil
	l.currentItem = 0
	l.offset = 0
	return l
}

// Focus is called by the application when the primitive receives focus.
func (l *List) Focus(delegate func(p Primitive)) {
	l.Box.Focus(delegate)
	if l.ContextMenu.open {
		delegate(l.ContextMenu.list)
	}
}

// HasFocus returns whether or not this primitive has focus.
func (l *List) HasFocus() bool {
	if l.ContextMenu.open {
		return l.ContextMenu.list.HasFocus()
	}

	l.RLock()
	defer l.RUnlock()
	return l.hasFocus
}

// Transform modifies the current selection.
func (l *List) Transform(tr Transformation) {
	l.Lock()
	defer l.Unlock()

	l.transform(tr)
}

func (l *List) transform(tr Transformation) {
	var decreasing bool

	switch tr {
	case TransformFirstItem:
		l.currentItem = 0
		l.offset = 0
		decreasing = true
	case TransformLastItem:
		l.currentItem = len(l.items) - 1
	case TransformPreviousItem:
		l.currentItem -= 1
		decreasing = true
	case TransformNextItem:
		l.currentItem += 1
	case TransformPreviousPage:
		l.currentItem -= 5
		decreasing = true
	case TransformNextPage:
		l.currentItem += 5
	}

	for i := 0; i < len(l.items); i++ {
		if l.currentItem < 0 {
			if l.wrapAround {
				l.currentItem = len(l.items) - 1
			} else {
				l.currentItem = 0
				l.offset = 0
			}
		} else if l.currentItem >= len(l.items) {
			if l.wrapAround {
				l.currentItem = 0
				l.offset = 0
			} else {
				l.currentItem = len(l.items) - 1
			}
		}

		item := l.items[l.currentItem]
		if item.Enabled {
			break
		}

		if decreasing {
			l.currentItem--
		} else {
			l.currentItem++
		}
	}

	l.updateOffset()
}

func (l *List) updateOffset() {
	_, _, _, l.height = l.GetInnerRect()

	h := l.height
	if l.selectedAlwaysCentered {
		h /= 2
	}

	if l.currentItem < l.offset {
		l.offset = l.currentItem
	} else if l.showSecondaryText {
		if 2*(l.currentItem-l.offset) >= h-1 {
			l.offset = (2*l.currentItem + 3 - h) / 2
		}
	} else {
		if l.currentItem-l.offset >= h {
			l.offset = l.currentItem + 1 - h
		}
	}

	if l.showSecondaryText {
		if l.offset > len(l.items)-(l.height/2) {
			l.offset = len(l.items) - l.height/2
		}
	} else {
		if l.offset > len(l.items)-l.height {
			l.offset = len(l.items) - l.height
		}
	}

	if l.offset < 0 {
		l.offset = 0
	}
}

// Draw draws this primitive onto the screen.
func (l *List) Draw(screen tcell.Screen) {
	l.Box.Draw(screen)
	hasFocus := l.GetFocusable().HasFocus()

	l.Lock()
	defer l.Unlock()

	// Determine the dimensions.
	x, y, width, height := l.GetInnerRect()
	bottomLimit := y + height

	l.height = height

	screenWidth, _ := screen.Size()
	scrollBarHeight := height
	scrollBarX := x + (width - 1) + l.paddingLeft + l.paddingRight
	if scrollBarX > screenWidth-1 {
		scrollBarX = screenWidth - 1
	}

	// Halve scroll bar height when drawing two lines per list item.
	if l.showSecondaryText {
		scrollBarHeight /= 2
	}

	// Do we show any shortcuts?
	var showShortcuts bool
	for _, item := range l.items {
		if item.Shortcut != 0 {
			showShortcuts = true
			x += 4
			width -= 4
			break
		}
	}

	// Adjust offset to keep the current selection in view.
	if l.selectedAlwaysVisible || l.selectedAlwaysCentered {
		l.updateOffset()
	}

	// Draw the list items.
	for index, item := range l.items {
		if index < l.offset {
			continue
		}

		if y >= bottomLimit {
			break
		}

		if item.MainText == "" && item.SecondaryText == "" && item.Shortcut == 0 { // Divider
			Print(screen, string(tcell.RuneLTee), (x-5)-l.paddingLeft, y, 1, AlignLeft, l.mainTextColor)
			Print(screen, strings.Repeat(string(tcell.RuneHLine), width+4+l.paddingLeft+l.paddingRight), (x-4)-l.paddingLeft, y, width+4+l.paddingLeft+l.paddingRight, AlignLeft, l.mainTextColor)
			Print(screen, string(tcell.RuneRTee), (x-5)+width+5+l.paddingRight, y, 1, AlignLeft, l.mainTextColor)

			RenderScrollBar(screen, l.scrollBarVisibility, scrollBarX, y, scrollBarHeight, len(l.items), l.currentItem, index-l.offset, l.hasFocus, l.scrollBarColor)
			y++
			continue
		} else if !item.Enabled { // Disabled item
			// Shortcuts.
			if showShortcuts && item.Shortcut != 0 {
				Print(screen, fmt.Sprintf("(%s)", string(item.Shortcut)), x-5, y, 4, AlignRight, tcell.ColorDarkSlateGray.TrueColor())
			}

			// Main text.
			Print(screen, item.MainText, x, y, width, AlignLeft, tcell.ColorGray.TrueColor())

			RenderScrollBar(screen, l.scrollBarVisibility, scrollBarX, y, scrollBarHeight, len(l.items), l.currentItem, index-l.offset, l.hasFocus, l.scrollBarColor)
			y++
			continue
		}

		// Shortcuts.
		if showShortcuts && item.Shortcut != 0 {
			Print(screen, fmt.Sprintf("(%s)", string(item.Shortcut)), x-5, y, 4, AlignRight, l.shortcutColor)
		}

		// Main text.
		Print(screen, item.MainText, x, y, width, AlignLeft, l.mainTextColor)

		// Background color of selected text.
		if index == l.currentItem && (!l.selectedFocusOnly || hasFocus) {
			textWidth := width
			if !l.highlightFullLine {
				if w := TaggedStringWidth(item.MainText); w < textWidth {
					textWidth = w
				}
			}

			for bx := 0; bx < textWidth; bx++ {
				m, c, style, _ := screen.GetContent(x+bx, y)
				fg, _, _ := style.Decompose()
				if fg == l.mainTextColor {
					fg = l.selectedTextColor
				}
				style = SetAttributes(style.Background(l.selectedBackgroundColor).Foreground(fg), l.selectedTextAttributes)
				screen.SetContent(x+bx, y, m, c, style)
			}
		}

		RenderScrollBar(screen, l.scrollBarVisibility, scrollBarX, y, scrollBarHeight, len(l.items), l.currentItem, index-l.offset, l.hasFocus, l.scrollBarColor)

		y++

		if y >= bottomLimit {
			break
		}

		// Secondary text.
		if l.showSecondaryText {
			Print(screen, item.SecondaryText, x, y, width, AlignLeft, l.secondaryTextColor)

			RenderScrollBar(screen, l.scrollBarVisibility, scrollBarX, y, scrollBarHeight, len(l.items), l.currentItem, index-l.offset, l.hasFocus, l.scrollBarColor)

			y++
		}
	}

	// Overdraw scroll bar when necessary.
	for y < bottomLimit {
		RenderScrollBar(screen, l.scrollBarVisibility, scrollBarX, y, scrollBarHeight, len(l.items), l.currentItem, bottomLimit-y, l.hasFocus, l.scrollBarColor)

		y++
	}

	// Draw context menu.
	if hasFocus && l.ContextMenu.open {
		ctx := l.ContextMenuList()

		x, y, width, height = l.GetInnerRect()

		// What's the longest option text?
		maxWidth := 0
		for _, option := range ctx.items {
			strWidth := TaggedStringWidth(option.MainText)
			if option.Shortcut != 0 {
				strWidth += 4
			}
			if strWidth > maxWidth {
				maxWidth = strWidth
			}
		}

		lheight := len(ctx.items)
		lwidth := maxWidth

		// Add space for borders
		lwidth += 2
		lheight += 2

		lwidth += ctx.paddingLeft + ctx.paddingRight
		lheight += ctx.paddingTop + ctx.paddingBottom

		cx, cy := l.ContextMenu.x, l.ContextMenu.y
		if cx < 0 || cy < 0 {
			offsetX := 7
			if showShortcuts {
				offsetX += 4
			}
			offsetY := l.currentItem
			if l.showSecondaryText {
				offsetY *= 2
			}
			x, y, _, _ := l.GetInnerRect()
			cx, cy = x+offsetX, y+offsetY
		}

		_, sheight := screen.Size()
		if cy+lheight >= sheight && cy-2 > lheight-cy {
			for i := (cy + lheight) - sheight; i > 0; i-- {
				cy--
				if cy+lheight < sheight {
					break
				}
			}
			if cy < 0 {
				cy = 0
			}
		}
		if cy+lheight >= sheight {
			lheight = sheight - cy
		}

		if ctx.scrollBarVisibility == ScrollBarAlways || (ctx.scrollBarVisibility == ScrollBarAuto && len(ctx.items) > lheight) {
			lwidth++ // Add space for scroll bar
		}

		ctx.SetRect(cx, cy, lwidth, lheight)
		ctx.Draw(screen)
	}
}

// InputHandler returns the handler for this primitive.
func (l *List) InputHandler() func(event *tcell.EventKey, setFocus func(p Primitive)) {
	return l.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p Primitive)) {
		l.Lock()

		previousItem := l.currentItem

		if HitShortcut(event, Keys.Cancel) {
			if l.ContextMenu.open {
				l.Unlock()

				l.ContextMenu.hide(setFocus)
				return
			}

			if l.done != nil {
				l.Unlock()
				l.done()
			} else {
				l.Unlock()
			}
			return
		} else if HitShortcut(event, Keys.Select, Keys.Select2) {
			if l.currentItem >= 0 && l.currentItem < len(l.items) {
				item := l.items[l.currentItem]
				if item.Enabled {
					if item.Selected != nil {
						l.Unlock()
						item.Selected()
						l.Lock()
					}
					if l.selected != nil {
						l.Unlock()
						l.selected(l.currentItem, item.MainText, item.SecondaryText, item.Shortcut)
						l.Lock()
					}
				}
			}
		} else if HitShortcut(event, Keys.ShowContextMenu) {
			defer l.ContextMenu.show(l.currentItem, -1, -1, setFocus)
		} else if len(l.items) == 0 {
			l.Unlock()
			return
		}

		var matchesShortcut bool
		if event.Key() == tcell.KeyRune {
			ch := event.Rune()
			if ch != ' ' {
				// It's not a space bar. Is it a shortcut?
				for index, item := range l.items {
					if item.Enabled && item.Shortcut == ch {
						// We have a shortcut.
						matchesShortcut = true
						l.currentItem = index

						item := l.items[l.currentItem]
						if item.Selected != nil {
							l.Unlock()
							item.Selected()
							l.Lock()
						}
						if l.selected != nil {
							l.Unlock()
							l.selected(l.currentItem, item.MainText, item.SecondaryText, item.Shortcut)
							l.Lock()
						}

						break
					}
				}
			}
		}

		if !matchesShortcut {
			if HitShortcut(event, Keys.MoveFirst, Keys.MoveFirst2) {
				l.transform(TransformFirstItem)
			} else if HitShortcut(event, Keys.MoveLast, Keys.MoveLast2) {
				l.transform(TransformLastItem)
			} else if HitShortcut(event, Keys.MoveUp, Keys.MoveUp2, Keys.MovePreviousField) {
				l.transform(TransformPreviousItem)
			} else if HitShortcut(event, Keys.MoveDown, Keys.MoveDown2, Keys.MoveNextField) {
				l.transform(TransformNextItem)
			} else if HitShortcut(event, Keys.MovePreviousPage) {
				l.transform(TransformPreviousPage)
			} else if HitShortcut(event, Keys.MoveNextPage) {
				l.transform(TransformNextPage)
			}
		}

		if l.currentItem != previousItem && l.currentItem < len(l.items) && l.changed != nil {
			item := l.items[l.currentItem]
			l.Unlock()
			l.changed(l.currentItem, item.MainText, item.SecondaryText, item.Shortcut)
		} else {
			l.Unlock()
		}
	})
}

// indexAtY returns the index of the list item found at the given Y position
// or a negative value if there is no such list item.
func (l *List) indexAtY(y int) int {
	_, rectY, _, height := l.GetInnerRect()
	if y < rectY || y >= rectY+height {
		return -1
	}

	index := y - rectY
	if l.showSecondaryText {
		index /= 2
	}
	index += l.offset

	if index >= len(l.items) {
		return -1
	}
	return index
}

// indexAtPoint returns the index of the list item found at the given position
// or a negative value if there is no such list item.
func (l *List) indexAtPoint(x, y int) int {
	rectX, rectY, width, height := l.GetInnerRect()
	if x < rectX || x >= rectX+width || y < rectY || y >= rectY+height {
		return -1
	}

	index := y - rectY
	if l.showSecondaryText {
		index /= 2
	}
	index += l.offset

	if index >= len(l.items) {
		return -1
	}
	return index
}

// MouseHandler returns the mouse handler for this primitive.
func (l *List) MouseHandler() func(action MouseAction, event *tcell.EventMouse, setFocus func(p Primitive)) (consumed bool, capture Primitive) {
	return l.WrapMouseHandler(func(action MouseAction, event *tcell.EventMouse, setFocus func(p Primitive)) (consumed bool, capture Primitive) {
		l.Lock()

		// Pass events to context menu.
		if l.ContextMenuVisible() && l.ContextMenuList().InRect(event.Position()) {
			defer l.ContextMenuList().MouseHandler()(action, event, setFocus)
			consumed = true
			l.Unlock()
			return
		}

		if !l.InRect(event.Position()) {
			l.Unlock()
			return false, nil
		}

		// Process mouse event.
		switch action {
		case MouseLeftClick:
			if l.ContextMenuVisible() {
				defer l.ContextMenu.hide(setFocus)
				consumed = true
				l.Unlock()
				return
			}

			l.Unlock()
			setFocus(l)
			l.Lock()

			index := l.indexAtPoint(event.Position())
			if index != -1 {
				item := l.items[index]
				if item.Enabled {
					l.currentItem = index
					if item.Selected != nil {
						l.Unlock()
						item.Selected()
						l.Lock()
					}
					if l.selected != nil {
						l.Unlock()
						l.selected(index, item.MainText, item.SecondaryText, item.Shortcut)
						l.Lock()
					}
					if index != l.currentItem && l.changed != nil {
						l.Unlock()
						l.changed(index, item.MainText, item.SecondaryText, item.Shortcut)
						l.Lock()
					}
				}
			}
			consumed = true
		case MouseMiddleClick:
			if l.ContextMenu.open {
				defer l.ContextMenu.hide(setFocus)
				consumed = true
				l.Unlock()
				return
			}
		case MouseRightDown:
			if len(l.ContextMenuList().items) == 0 {
				l.Unlock()
				return
			}

			x, y := event.Position()

			index := l.indexAtPoint(event.Position())
			if index != -1 {
				item := l.items[index]
				if item.Enabled {
					l.currentItem = index
					if index != l.currentItem && l.changed != nil {
						l.Unlock()
						l.changed(index, item.MainText, item.SecondaryText, item.Shortcut)
						l.Lock()
					}
				}
			}

			defer l.ContextMenu.show(l.currentItem, x, y, setFocus)
			l.ContextMenu.drag = true
			consumed = true
		case MouseMove:
			if l.hover {
				_, y := event.Position()
				index := l.indexAtY(y)
				if index >= 0 {
					item := l.items[index]
					if item.Enabled {
						l.currentItem = index
					}
				}

				consumed = true
			}
		case MouseScrollUp:
			if l.offset > 0 {
				l.offset--
			}
			consumed = true
		case MouseScrollDown:
			lines := len(l.items) - l.offset
			if l.showSecondaryText {
				lines *= 2
			}
			if _, _, _, height := l.GetInnerRect(); lines > height {
				l.offset++
			}
			consumed = true
		}

		l.Unlock()
		return
	})
}
