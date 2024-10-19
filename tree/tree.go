package tree

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	ltree "github.com/charmbracelet/lipgloss/tree"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
)

// StyleFunc allows the tree to be styled per item.
type StyleFunc func(children Nodes, i int) lipgloss.Style

// Styles contains style definitions for this tree component. By default, these
// values are generated by DefaultStyles.
type Styles struct {
	SelectionCursor       lipgloss.Style
	HelpStyle             lipgloss.Style
	TreeStyle             lipgloss.Style
	selectedNodeFunc      StyleFunc
	SelectedNodeStyle     lipgloss.Style
	SelectedNodeStyleFunc StyleFunc
	nodeFunc              StyleFunc
	NodeStyle             lipgloss.Style
	NodeStyleFunc         StyleFunc
}

// DefaultStyles returns a set of default style definitions for this tree
// component.
func DefaultStyles() (s Styles) {
	s.NodeStyle = lipgloss.NewStyle()
	s.SelectedNodeStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("18")).
		Bold(true)
	s.nodeFunc = func(_ Nodes, _ int) lipgloss.Style {
		return s.NodeStyle
	}
	s.selectedNodeFunc = func(_ Nodes, _ int) lipgloss.Style {
		return s.SelectedNodeStyle
	}
	s.SelectionCursor = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	s.HelpStyle = lipgloss.NewStyle().PaddingTop(1)

	return s
}

// KeyMap is the key bindings for different actions within the tree.
type KeyMap struct {
	Down   key.Binding
	Up     key.Binding
	Toggle key.Binding

	// Help toggle keybindings.
	ShowFullHelp  key.Binding
	CloseFullHelp key.Binding

	Quit key.Binding
}

// DefaultKeyMap is the default set of key bindings for navigating and acting
// upon the tree.
var DefaultKeyMap = KeyMap{
	Down: key.NewBinding(
		key.WithKeys("down", "j", "ctrl+n"),
		key.WithHelp("↓/j", "down"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "k", "ctrl+p"),
		key.WithHelp("↑/k", "up"),
	),
	Toggle: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("⏎", "toggle"),
	),

	// Toggle help.
	ShowFullHelp: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "more"),
	),
	CloseFullHelp: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "close help"),
	),

	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

// Model is the Bubble Tea model for this tree element.
type Model struct {
	showHelp bool
	// OpenCharacter is the character used to represent an open node.
	OpenCharacter string
	// ClosedCharacter is the character used to represent a closed node.
	ClosedCharacter string
	// KeyMap encodes the keybindings recognized by the widget.
	KeyMap KeyMap
	// styles sets the styling for the tree
	styles Styles
	Help   help.Model

	// Additional key mappings for the short and full help views. This allows
	// you to add additional key mappings to the help menu without
	// re-implementing the help component. Of course, you can also disable the
	// list's help component and implement a new one if you need more
	// flexibility.
	AdditionalShortHelpKeys func() []key.Binding
	AdditionalFullHelpKeys  func() []key.Binding

	root   *Node
	width  int
	height int
	// yOffset is the vertical offset of the selected node.
	yOffset int
}

// Node is a a node in the tree
// Node implements lipgloss's tree.Node
type Node struct {
	// tree is used as the renderer layer
	tree *ltree.Tree

	// yOffset is the vertical offset of the selected node.
	yOffset int

	// depth is the depth of the node in the tree
	depth int

	// isRoot is true for every Node which was added with tree.Root
	isRoot bool
	open   bool

	// value is the root value of the node
	value any

	// TODO: expose a getter for this in lipgloss?
	rootStyle lipgloss.Style

	opts *itemOptions

	// TODO: move to lipgloss.Tree?
	size int
}

// IsSelected returns whether this item is selected.
func (t *Node) IsSelected() bool {
	return t.yOffset == t.opts.treeYOffset
}

// Depth returns the depth of the node in the tree.
func (t *Node) Depth() int {
	return t.depth
}

// Size returns the number of nodes in the tree.
// Note that if a child isn't open, its size is 1
func (t *Node) Size() int {
	return t.size
}

// YOffset returns the vertical offset of the Node
func (t *Node) YOffset() int {
	return t.yOffset
}

type itemOptions struct {
	openCharacter   string
	closedCharacter string
	treeYOffset     int
}

// Used to print the Node's tree
// TODO: Value is not called on the root node, so we need to repeat the open/closed character
// Should this be fixed in lipgloss?
func (t *Node) String() string {
	s := t.rootStyle.UnsetWidth()
	if t.open {
		return s.Render(t.opts.openCharacter+" ") + t.tree.String()
	}
	return s.Render(t.opts.closedCharacter+" ") + t.tree.String()
}

// Value returns the root name of this node.
func (t *Node) Value() string {
	s := lipgloss.NewStyle()
	if t.isRoot {
		if t.open {
			return s.Render(t.opts.openCharacter + " " + t.tree.Value())
		}
		return s.Render(t.opts.closedCharacter + " " + t.tree.Value())
	}
	return s.Render(t.tree.Value())
}

// GivenValue returns the value passed to the node.
func (t *Node) GivenValue() any {
	return t.value
}

// Children returns the children of an item.
func (t *Node) Children() ltree.Children {
	return t.tree.Children()
}

// Hidden returns whether this item is hidden.
func (t *Node) Hidden() bool {
	return t.tree.Hidden()
}

// Nodes are a list of tree nodes.
type Nodes []*Node

// Children returns the children of an item.
func (t Nodes) At(index int) *Node {
	return t[index]
}

// Children returns the children of an item.
func (t Nodes) Length() int {
	return len(t)
}

// ItemStyle sets a static style for all items.
func (t *Node) ItemStyle(s lipgloss.Style) *Node {
	t.tree.ItemStyle(s)
	return t
}

// ItemStyleFunc sets the item style function. Use this for conditional styling.
// For example:
//
//	t := tree.Root("root").
//		ItemStyleFunc(func(_ tree.Nodes, i int) lipgloss.Style {
//			if selected == i {
//				return lipgloss.NewStyle().Foreground(hightlightColor)
//			}
//			return lipgloss.NewStyle().Foreground(dimColor)
//		})
func (t *Node) ItemStyleFunc(f StyleFunc) *Node {
	t.tree.ItemStyleFunc(func(children ltree.Children, i int) lipgloss.Style {
		c := make(Nodes, children.Length())
		// TODO: if we expose Depth and Size in lipgloss, we can avoid this
		for i := 0; i < children.Length(); i++ {
			c[i] = children.At(i).(*Node)
		}
		return f(c, i)
	})
	return t
}

// TODO: support IndentStyleFunc in lipgloss so we can have a full background for the item

// TODO: should we re-export RoundedEnumerator from lipgloss?
// Enumerator sets the enumerator implementation. This can be used to change the
// way the branches indicators look.  Lipgloss includes predefined enumerators
// for a classic or rounded tree. For example, you can have a rounded tree:
//
//	tree.New().
//		Enumerator(ltree.RoundedEnumerator)
func (t *Node) Enumerator(enumerator ltree.Enumerator) *Node {
	t.tree.Enumerator(enumerator)
	return t
}

// Indenter sets the indenter implementation. This is used to change the way
// the tree is indented. The default indentor places a border connecting sibling
// elements and no border for the last child.
//
//	└── Foo
//	    └── Bar
//	        └── Baz
//	            └── Qux
//	                └── Quux
//
// You can define your own indenter.
//
//	func ArrowIndenter(children tree.Children, index int) string {
//		return "→ "
//	}
//
//	→ Foo
//	→ → Bar
//	→ → → Baz
//	→ → → → Qux
//	→ → → → → Quux
func (t *Node) Indenter(indenter ltree.Indenter) *Node {
	t.tree.Indenter(indenter)
	return t
}

// EnumeratorStyle sets a static style for all enumerators.
//
// Use EnumeratorStyleFunc to conditionally set styles based on the tree node.
func (t *Node) EnumeratorStyle(style lipgloss.Style) *Node {
	t.tree.EnumeratorStyle(style)
	return t
}

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
func (t *Node) EnumeratorStyleFunc(f func(children ltree.Children, i int) lipgloss.Style) *Node {
	t.tree.EnumeratorStyleFunc(f)
	return t
}

// RootStyle sets a style for the root element.
func (t *Node) RootStyle(style lipgloss.Style) *Node {
	t.tree.RootStyle(style)
	return t
}

// Child adds a child to this tree.
//
// If a Child Node is passed without a root, it will be parented to it's sibling
// child (auto-nesting).
//
//	tree.Root("Foo").Child(tree.Root("Bar").Child("Baz"), "Qux")
//
//	├── Foo
//	├── Bar
//	│   └── Baz
//	└── Qux
func (t *Node) Child(children ...any) *Node {
	for _, child := range children {
		switch child := child.(type) {
		case *Node:
			t.size = t.size + child.size
			t.open = t.size > 1
			child.open = child.size > 1
			t.tree.Child(child)
		default:
			item := new(Node)
			item.tree = ltree.Root(child)
			item.size = 1
			item.open = false
			item.value = child
			t.size = t.size + item.size
			t.open = t.size > 1
			t.tree.Child(item)
		}
	}

	return t
}

// Root returns a new tree with the root set.
func Root(root any) *Node {
	t := new(Node)
	t.size = 1
	t.open = true
	t.value = root
	t.isRoot = true
	t.tree = ltree.Root(root)
	return t
}

// New creates a new model with default settings.
func New(t *Node, width, height int) Model {
	m := Model{
		KeyMap:          DefaultKeyMap,
		OpenCharacter:   "▼",
		ClosedCharacter: "▶",
		Help:            help.New(),

		showHelp: true,
		root:     t,
	}
	m.SetStyles(DefaultStyles())
	m.setSize(width, height)
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
		case key.Matches(msg, m.KeyMap.ShowFullHelp):
			fallthrough
		case key.Matches(msg, m.KeyMap.CloseFullHelp):
			m.Help.ShowAll = !m.Help.ShowAll
		}
	}

	// not sure why, but I think m.yOffset is captured in the closure, so we need to update the styles
	m.updateStyles()
	return m, tea.Batch(cmds...)
}

// View renders the component.
func (m Model) View() string {
	var treeView, leftDebugView, cursor string
	// TODO: remove
	if os.Getenv("DEBUG") == "true" {
		// topDebugView += fmt.Sprintf("y=%2d\n", m.yOffset)
		leftDebugView = printDebugInfo(m.root) + " "
		for i := 0; i < m.root.size; i++ {
			if i == m.yOffset {
				cursor = cursor + "👉 "
			} else {
				cursor = cursor + "  "
			}
			cursor = cursor + "\n"
		}
	}

	treeView = m.styles.TreeStyle.Render(m.root.String())

	treeView = lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftDebugView,
		cursor,
		treeView,
	)

	var help string
	if m.showHelp {
		help = m.helpView()
	}

	return lipgloss.JoinVertical(lipgloss.Left, treeView, help)
}

// SetStyles sets the styles for this component.
func (m *Model) SetStyles(styles Styles) {
	if styles.NodeStyleFunc != nil {
		styles.nodeFunc = styles.NodeStyleFunc
	} else {
		styles.nodeFunc = func(_ Nodes, _ int) lipgloss.Style {
			return styles.NodeStyle
		}
	}
	if styles.SelectedNodeStyleFunc != nil {
		styles.selectedNodeFunc = styles.SelectedNodeStyleFunc
	} else {
		styles.selectedNodeFunc = func(_ Nodes, _ int) lipgloss.Style {
			return styles.SelectedNodeStyle
		}
	}
	m.styles = styles
	m.updateStyles()
}

// SetShowHelp shows or hides the help view.
func (m *Model) SetShowHelp(v bool) {
	m.showHelp = v
}

// SetSize sets the width and height of this component.
func (m *Model) SetSize(width, height int) {
	m.setSize(width, height)
}

// SetWidth sets the width of this component.
func (m *Model) SetWidth(v int) {
	m.setSize(v, m.height)
}

// SetHeight sets the height of this component.
func (m *Model) SetHeight(v int) {
	m.setSize(m.width, v)
}

func (m *Model) setSize(width, height int) {
	m.width = width
	m.height = height
	m.Help.Width = width
}

// ShortHelp returns bindings to show in the abbreviated help view. It's part
// of the help.KeyMap interface.
func (m Model) ShortHelp() []key.Binding {
	kb := []key.Binding{
		m.KeyMap.Up,
		m.KeyMap.Down,
		m.KeyMap.Toggle,
	}

	if m.AdditionalShortHelpKeys != nil {
		kb = append(kb, m.AdditionalShortHelpKeys()...)
	}

	kb = append(kb, m.KeyMap.Quit, m.KeyMap.ShowFullHelp)

	return kb
}

// FullHelp returns bindings to show the full help view. It's part of the
// help.KeyMap interface.
func (m Model) FullHelp() [][]key.Binding {
	kb := [][]key.Binding{
		{
			m.KeyMap.Up,
			m.KeyMap.Down,
			m.KeyMap.Toggle,
		},
	}

	if m.AdditionalFullHelpKeys != nil {
		kb = append(kb, m.AdditionalFullHelpKeys())
	}

	kb = append(kb, []key.Binding{
		m.KeyMap.Quit,
		m.KeyMap.CloseFullHelp,
	})

	return kb
}

func (m Model) helpView() string {
	return m.styles.HelpStyle.Render(m.Help.View(m))
}

// FlatNodes returns all items in the tree as a flat list.
func (m *Model) FlatNodes() []*Node {
	return m.root.FlatNodes()
}

func (m *Model) setAttributes() {
	setDepths(m.root, 0)
	setSizes(m.root)
	setYOffsets(m.root)
}

// FlatNodes returns all descendant items in as a flat list.
func (t *Node) FlatNodes() []*Node {
	res := []*Node{t}
	children := t.tree.Children()
	for i := 0; i < children.Length(); i++ {
		child := children.At(i)
		res = append(res, child.(*Node).FlatNodes()...)
	}
	return res
}

// setSizes updates each Node's size
// Note that if a child isn't open, its size is 1
func setDepths(t *Node, depth int) {
	t.depth = depth
	children := t.tree.Children()
	for i := 0; i < children.Length(); i++ {
		child := children.At(i)
		setDepths(child.(*Node), depth+1)
	}
}

// setSizes updates each Node's size
// Note that if a child isn't open, its size is 1
func setSizes(t *Node) int {
	children := t.tree.Children()
	size := 1 + children.Length()
	for i := 0; i < children.Length(); i++ {
		child := children.At(i)
		size = size + setSizes(child.(*Node)) - 1
	}
	t.size = size
	return size
}

// setYOffsets updates each Node's yOffset based on how many items are "above" it
func setYOffsets(t *Node) {
	children := t.tree.Children()
	above := 0
	for i := 0; i < children.Length(); i++ {
		child := children.At(i)
		if child, ok := child.(*Node); ok {
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

// Node returns the item at the given yoffset
func (m *Model) Node(yoffset int) *Node {
	return findNode(m.root, yoffset)
}

// NodeAtCurrentOffset returns the item at the current yoffset
func (m *Model) NodeAtCurrentOffset() *Node {
	return findNode(m.root, m.yOffset)
}

// Since the selected node changes, we need to capture m.yOffset in the
// style function's closure again
func (m *Model) updateStyles() {
	m.root.rootStyle = m.rootStyle()
	// TODO: add RootStyleFunc to the Node interface?
	m.root.RootStyle(m.root.rootStyle)
	m.root.ItemStyleFunc(m.selectedNodeStyle())

	items := m.FlatNodes()
	opts := &itemOptions{
		openCharacter:   m.OpenCharacter,
		closedCharacter: m.ClosedCharacter,
		treeYOffset:     m.yOffset,
	}
	for _, item := range items {
		item.opts = opts
	}
}

// selectedNodeStyle sets the node style
// and takes into account whether it's selected or not
func (m *Model) selectedNodeStyle() StyleFunc {
	return func(children Nodes, i int) lipgloss.Style {
		child := children.At(i)
		if child.yOffset == m.yOffset {
			return m.styles.selectedNodeFunc(children, i)
		}

		return m.styles.nodeFunc(children, i)
	}
}

func (m *Model) rootStyle() lipgloss.Style {
	if m.styles.nodeFunc == nil || m.styles.selectedNodeFunc == nil {
		return lipgloss.NewStyle()
	}
	if m.root.yOffset == m.yOffset {
		s := m.styles.selectedNodeFunc(Nodes{m.root}, 0)
		// TODO: if we call Value on the root node in lipgloss, we wouldn't need this
		return s.Width(s.GetWidth() - lipgloss.Width(m.OpenCharacter) - 1)
	}

	s := m.styles.nodeFunc(Nodes{m.root}, 0)
	return s.Width(s.GetWidth() - lipgloss.Width(m.OpenCharacter) - 1)
}

// TODO: remove
func printDebugInfo(t *Node) string {
	debug := fmt.Sprintf("size=%2d y=%2d depth=%2d", t.size, t.yOffset, t.depth)
	children := t.Children()
	for i := 0; i < children.Length(); i++ {
		child := children.At(i)
		if child, ok := child.(*Node); ok {
			debug = debug + "\n" + printDebugInfo(child)
		}
	}

	return debug
}

// findNode starts a DFS search for the node with the given yOffset
// starting from the given item
func findNode(t *Node, yOffset int) *Node {
	if t.yOffset == yOffset {
		return t
	}

	children := t.tree.Children()
	for i := 0; i < children.Length(); i++ {
		child := children.At(i)
		if child, ok := child.(*Node); ok {
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
