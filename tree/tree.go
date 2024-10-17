package tree

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	ltree "github.com/charmbracelet/lipgloss/tree"

	"github.com/charmbracelet/bubbles/key"
)

// Styles contains style definitions for this tree component. By default, these
// values are generated by DefaultStyles.
type Styles struct {
	SelectedNode    lipgloss.Style
	ItemStyle       lipgloss.Style
	SelectionCursor lipgloss.Style
}

// DefaultStyles returns a set of default style definitions for this tree
// component.
func DefaultStyles() (s Styles) {
	s.SelectedNode = lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("18")).
		Bold(true)
	s.SelectionCursor = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	s.ItemStyle = lipgloss.NewStyle()

	return s
}

// KeyMap is the key bindings for different actions within the tree.
type KeyMap struct {
	Down   key.Binding
	Up     key.Binding
	Toggle key.Binding
	Quit   key.Binding
}

// DefaultKeyMap is the default set of key bindings for navigating and acting
// upon the tree.
var DefaultKeyMap = KeyMap{
	Down:   key.NewBinding(key.WithKeys("down", "j", "ctrl+n"), key.WithHelp("down", "next line")),
	Up:     key.NewBinding(key.WithKeys("up", "k", "ctrl+p"), key.WithHelp("up", "previous line")),
	Toggle: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "toggle")),
	Quit:   key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}

// Model is the Bubble Tea model for this tree element.
type Model struct {
	root *Item
	// KeyMap encodes the keybindings recognized by the widget.
	KeyMap KeyMap

	// Styles sets the styling for the tree
	Styles Styles

	// OpenCharacter is the character used to represent an open node.
	OpenCharacter string

	// ClosedCharacter is the character used to represent a closed node.
	ClosedCharacter string

	// yOffset is the vertical offset of the selected node.
	yOffset int
}

// Item is a a node in the tree
// Item implements lipgloss's tree.Node
type Item struct {
	// tree is used as the renderer layer
	tree *ltree.Tree

	// yOffset is the vertical offset of the selected node.
	yOffset int

	// isRoot is true for every Item which was added with tree.Root
	isRoot bool
	open   bool

	// TODO: expose a getter for this in lipgloss
	rootStyle lipgloss.Style

	opts *itemOptions

	// TODO: move to lipgloss.Tree?
	size int
}

type itemOptions struct {
	openCharacter   string
	closedCharacter string
}

// Used to print the Item's tree
// TODO: Value is not called on the root node, so we need to repeat the open/closed character
// Should this be fixed in lipgloss?
func (t *Item) String() string {
	if t.open {
		return t.rootStyle.Render(t.opts.openCharacter+" ") + t.tree.String()
	}
	return t.rootStyle.Render(t.opts.closedCharacter+" ") + t.tree.String()
}

// Value returns the root name of this node.
func (t *Item) Value() string {
	if t.isRoot {
		if t.open {
			return t.opts.openCharacter + " " + t.tree.Value()
		}
		return t.opts.closedCharacter + " " + t.tree.Value()
	}
	return t.tree.Value()
}

// Children returns the children of an item.
func (t *Item) Children() ltree.Children {
	return t.tree.Children()
}

// Hidden returns whether this item is hidden.
func (t *Item) Hidden() bool {
	return t.tree.Hidden()
}

// ItemStyleFunc sets the item style function. Use this for conditional styling.
// For example:
//
//	t := tree.Root("root").
//		ItemStyleFunc(func(_ tree.Data, i int) lipgloss.Style {
//			if selected == i {
//				return lipgloss.NewStyle().Foreground(hightlightColor)
//			}
//			return lipgloss.NewStyle().Foreground(dimColor)
//		})
func (t *Item) ItemStyleFunc(f func(children ltree.Children, i int) lipgloss.Style) *Item {
	t.tree.ItemStyleFunc(f)
	return t
}

// TODO: support IndentStyleFunc in lipgloss so we can have a full background for the item?

// EnumeratorStyleFunc sets the enumeration style function. Use this function
// for conditional styling.
//
//	t := tree.Root("root").
//		EnumeratorStyleFunc(func(_ tree.Children, i int) lipgloss.Style {
//		    if selected == i {
//		        return lipgloss.NewStyle().Foreground(hightlightColor)
//		    }
//		    return lipgloss.NewStyle().Foreground(dimColor)
//		})
func (t *Item) EnumeratorStyleFunc(f func(children ltree.Children, i int) lipgloss.Style) *Item {
	t.tree.EnumeratorStyleFunc(f)
	return t
}

// RootStyle sets a style for the root element.
func (t *Item) RootStyle(style lipgloss.Style) *Item {
	t.tree.RootStyle(style)
	return t
}

// Child adds a child to this tree.
//
// If a Child Item is passed without a root, it will be parented to it's sibling
// child (auto-nesting).
//
//	tree.Root("Foo").Child(tree.Root("Bar").Child("Baz"), "Qux")
//
//	├── Foo
//	├── Bar
//	│   └── Baz
//	└── Qux
func (t *Item) Child(child any) *Item {
	switch child := child.(type) {
	case *Item:
		t.size = t.size + child.size
		t.open = t.size > 1
		child.open = child.size > 1
		t.tree.Child(child)
	default:
		item := new(Item)
		// TODO: should I create a tree for leaf nodes?
		// makes the code a bit simpler
		item.tree = ltree.Root(child)
		item.size = 1
		item.open = false
		t.size = t.size + item.size
		t.open = t.size > 1
		t.tree.Child(item)
	}

	return t
}

// Root returns a new tree with the root set.
func Root(root any) *Item {
	t := new(Item)
	t.size = 1
	t.open = true
	t.isRoot = true
	t.tree = ltree.Root(root)
	return t
}

// New creates a new model with default settings.
func New(t *Item) Model {
	m := Model{
		root:            t,
		KeyMap:          DefaultKeyMap,
		Styles:          DefaultStyles(),
		OpenCharacter:   "▼",
		ClosedCharacter: "▶",
	}
	m.setAttributes()
	m.updateStyles()
	return m
}

// Update is the Bubble Tea update loop.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.KeyMap.Down):
			m.yOffset = min(m.root.size-1, m.yOffset+1)
		case key.Matches(msg, m.KeyMap.Up):
			m.yOffset = max(0, m.yOffset-1)
		case key.Matches(msg, m.KeyMap.Toggle):
			node := findNode(m.root, m.yOffset)
			if node == nil {
				break
			}
			node.open = !node.open
			if node.open {
				node.tree.Offset(0, 0)
			} else {
				node.tree.Offset(0, node.tree.Children().Length())
			}
			m.setAttributes()
			m.yOffset = node.yOffset
		case key.Matches(msg, m.KeyMap.Quit):
			return m, tea.Quit
		}
	}

	// not sure why, but I think m.yOffset is captured in the closure, so we need to update the styles
	m.updateStyles()
	return m, tea.Batch(cmds...)
}

// View renders the component.
func (m Model) View() string {
	s := fmt.Sprintf("y=%d\n", m.yOffset)

	// TODO: remove
	debug := printDebugInfo(m.root)

	cursor := ""
	for i := 0; i < m.root.size; i++ {
		if i == m.yOffset {
			cursor = cursor + m.Styles.SelectedNode.Render("👉 ")
		} else {
			cursor = cursor + "  "
		}
		cursor = cursor + "\n"
	}

	t := lipgloss.JoinHorizontal(
		lipgloss.Top,
		lipgloss.NewStyle().Faint(true).MarginRight(1).Render(debug),
		cursor,
		m.root.String(),
	)
	return lipgloss.JoinVertical(lipgloss.Left, s, t)
}

// FlatItems returns all items in the tree as a flat list.
func (m *Model) FlatItems() []*Item {
	return m.root.FlatItems()
}

func (m *Model) setAttributes() {
	setSizes(m.root)
	setYOffsets(m.root)
}

// FlatItems returns all items in the tree as a flat list.
func (t *Item) FlatItems() []*Item {
	res := []*Item{t}
	children := t.tree.Children()
	for i := 0; i < children.Length(); i++ {
		child := children.At(i)
		res = append(res, child.(*Item).FlatItems()...)
	}
	return res
}

// setSizes updates each Item's size
// Note that if a child isn't open, its size is 1
func setSizes(t *Item) int {
	children := t.tree.Children()
	size := 1 + children.Length()
	for i := 0; i < children.Length(); i++ {
		child := children.At(i)
		size = size + setSizes(child.(*Item)) - 1
	}
	t.size = size
	return size
}

// setYOffsets updates each Item's yOffset based on how many items are "above" it
func setYOffsets(t *Item) {
	children := t.tree.Children()
	above := 0
	for i := 0; i < children.Length(); i++ {
		child := children.At(i)
		if child, ok := child.(*Item); ok {
			child.yOffset = t.yOffset + above + i + 1
			setYOffsets(child)
			above += child.size - 1
		}
	}
}

// YOffset returns the vertical offset of the selected node.
// Useful for scrolling to the selected node using a viewport.
func (m *Model) YOffset() int {
	return m.yOffset
}

// Since the selected node changes, we need to capture m.yOffset in the
// style function's closure again
func (m *Model) updateStyles() {
	m.root.rootStyle = m.nodeStyle(m.root)
	// TODO: add RootStyleFunc to the Node interface?
	m.root.RootStyle(m.root.rootStyle)
	m.root.ItemStyleFunc(m.selectedNodeStyle())

	items := m.FlatItems()
	opts := &itemOptions{
		openCharacter:   m.OpenCharacter,
		closedCharacter: m.ClosedCharacter,
	}
	for _, item := range items {
		item.opts = opts
	}
}

// selectedNodeStyle sets the node style
// and takes into account whether it's selected or not
func (m *Model) selectedNodeStyle() ltree.StyleFunc {
	return func(children ltree.Children, i int) lipgloss.Style {
		child := children.At(i)
		return m.nodeStyle(child)
	}
}

func (m *Model) nodeStyle(node ltree.Node) lipgloss.Style {
	if node, ok := node.(*Item); ok {
		if node.yOffset == m.yOffset {
			return m.Styles.SelectedNode
		}
	}
	return m.Styles.ItemStyle
}

// TODO: remove
func printDebugInfo(t *Item) string {
	debug := fmt.Sprintf("size=%-2d y=%d", t.size, t.yOffset)
	children := t.Children()
	for i := 0; i < children.Length(); i++ {
		child := children.At(i)
		if child, ok := child.(*Item); ok {
			debug = debug + "\n" + printDebugInfo(child)
		}
	}

	return debug
}

// findNode starts a DFS search for the node with the given yOffset
// starting from the given item
func findNode(t *Item, yOffset int) *Item {
	if t.yOffset == yOffset {
		return t
	}

	children := t.tree.Children()
	for i := 0; i < children.Length(); i++ {
		child := children.At(i)
		if child, ok := child.(*Item); ok {
			found := findNode(child, yOffset)
			if found != nil {
				return found
			}
		}
	}

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
