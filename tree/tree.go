package tree

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
)

// StyleFunc allows the tree to be styled per item.
type StyleFunc func(children Nodes, i int) lipgloss.Style

// Styles contains style definitions for this tree component. By default, these
// values are generated by DefaultStyles.
type Styles struct {
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
	s.TreeStyle = lipgloss.NewStyle()
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
	s.HelpStyle = lipgloss.NewStyle().PaddingTop(1)

	return s
}

const spacebar = " "

// KeyMap is the key bindings for different actions within the tree.
type KeyMap struct {
	Down         key.Binding
	Up           key.Binding
	PageDown     key.Binding
	PageUp       key.Binding
	HalfPageUp   key.Binding
	HalfPageDown key.Binding
	GoToTop      key.Binding
	GoToBottom   key.Binding

	Toggle key.Binding
	Open   key.Binding
	Close  key.Binding

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
	PageDown: key.NewBinding(
		key.WithKeys("pgdown", spacebar, "f"),
		key.WithHelp("f/pgdn", "page down"),
	),
	PageUp: key.NewBinding(
		key.WithKeys("pgup", "b"),
		key.WithHelp("b/pgup", "page up"),
	),
	HalfPageDown: key.NewBinding(
		key.WithKeys("d", "ctrl+d"),
		key.WithHelp("d", "½ page down"),
	),
	HalfPageUp: key.NewBinding(
		key.WithKeys("u", "ctrl+u"),
		key.WithHelp("u", "½ page up"),
	),
	GoToTop: key.NewBinding(
		key.WithKeys("g", "home"),
		key.WithHelp("g", "top"),
	),
	GoToBottom: key.NewBinding(
		key.WithKeys("G", "end"),
		key.WithHelp("G", "bottom"),
	),

	Toggle: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("⏎", "toggle"),
	),
	Open: key.NewBinding(
		key.WithKeys("l", "right"),
		key.WithHelp("→/l", "open"),
	),
	Close: key.NewBinding(
		key.WithKeys("h", "left"),
		key.WithHelp("←/h", "close"),
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
	// ScrollOff is the minimal number of lines to keep visible above and below the selected node.
	ScrollOff int
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

	root *Node

	viewport viewport.Model
	width    int
	height   int
	// yOffset is the vertical offset of the selected node.
	yOffset int
}

// New creates a new model with default settings.
func New(t *Node, width, height int) Model {
	m := Model{
		KeyMap:          DefaultKeyMap,
		OpenCharacter:   "▼",
		ClosedCharacter: "▶",
		Help:            help.New(),
		ScrollOff:       5,

		showHelp: true,
		root:     t,
		viewport: viewport.Model{},
	}
	m.SetStyles(DefaultStyles())
	m.SetSize(width, height)
	m.setAttributes()
	m.updateStyles()
	m.viewport.SetContent(m.root.String())
	return m
}

// Update is the Bubble Tea update loop.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.KeyMap.Down):
			m.updateViewport(1)
		case key.Matches(msg, m.KeyMap.Up):
			m.updateViewport(-1)
		case key.Matches(msg, m.KeyMap.PageDown):
			m.updateViewport(m.viewport.Height)
		case key.Matches(msg, m.KeyMap.PageUp):
			m.updateViewport(-m.viewport.Height)
		case key.Matches(msg, m.KeyMap.HalfPageDown):
			m.updateViewport(m.viewport.Height / 2)
		case key.Matches(msg, m.KeyMap.HalfPageUp):
			m.updateViewport(-m.viewport.Height / 2)
		case key.Matches(msg, m.KeyMap.GoToTop):
			m.updateViewport(-m.yOffset)
		case key.Matches(msg, m.KeyMap.GoToBottom):
			m.updateViewport(m.root.size)

		case key.Matches(msg, m.KeyMap.Toggle):
			node := findNode(m.root, m.yOffset)
			if node == nil {
				break
			}
			m.toggleNode(node, !node.IsOpen())
		case key.Matches(msg, m.KeyMap.Open):
			node := findNode(m.root, m.yOffset)
			if node == nil {
				break
			}
			m.toggleNode(node, true)
		case key.Matches(msg, m.KeyMap.Close):
			node := findNode(m.root, m.yOffset)
			if node == nil {
				break
			}
			m.toggleNode(node, false)

		case key.Matches(msg, m.KeyMap.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.KeyMap.ShowFullHelp):
			fallthrough
		case key.Matches(msg, m.KeyMap.CloseFullHelp):
			m.Help.ShowAll = !m.Help.ShowAll
		}
	}

	// not sure why, but I think m.yOffset is captured in the closure, so we need to update the styles
	return m, tea.Batch(cmds...)
}

// View renders the component.
func (m Model) View() string {
	var leftDebugView string
	// TODO: remove
	if os.Getenv("DEBUG") == "true" {
		leftDebugView = printDebugInfo(m.root) + " "
	}

	treeView := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftDebugView,
		m.viewport.View(),
	)

	var help string
	if m.showHelp {
		help = m.helpView()
	}

	return lipgloss.JoinVertical(lipgloss.Left, treeView, help)
}

func (m *Model) toggleNode(node *Node, open bool) {
	node.open = open

	// reset the offset to 0,0 first
	node.tree.Offset(0, 0)
	if !open {
		node.tree.Offset(node.tree.Children().Length(), 0)
	}
	m.setAttributes()
	m.updateViewport(m.yOffset - node.yOffset)
}

func (m *Model) updateViewport(movement int) {
	m.yOffset = max(min(m.root.size-1, m.yOffset+movement), 0)
	m.updateStyles()
	m.viewport.Style = m.styles.TreeStyle
	m.viewport.SetContent(m.styles.TreeStyle.Render(m.root.String()))
	if movement == 0 {
		return
	}

	// make sure there are enough lines above and below the selected node
	height := m.viewport.VisibleLineCount()
	scrolloff := min(m.ScrollOff, height/2)
	minTop := max(m.yOffset-scrolloff, 0)
	minBottom := min(m.viewport.TotalLineCount()-1, m.yOffset+scrolloff)

	if m.viewport.YOffset > minTop { // reveal more lines above
		m.viewport.SetYOffset(minTop)
	} else if m.viewport.YOffset+height < minBottom+1 { // reveal more lines below
		m.viewport.SetYOffset(minBottom - height + 1)
	}
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
	m.updateViewport(0)
}

// SetShowHelp shows or hides the help view.
func (m *Model) SetShowHelp(v bool) {
	m.showHelp = v
	m.SetSize(m.width, m.height)
}

// Width returns the current width setting.
func (m Model) Width() int {
	return m.width
}

// Height returns the current height setting.
func (m Model) Height() int {
	return m.height
}

// SetWidth sets the width of this component.
func (m *Model) SetWidth(v int) {
	m.SetSize(v, m.height)
}

// SetHeight sets the height of this component.
func (m *Model) SetHeight(v int) {
	m.SetSize(m.width, v)
}

// SetSize sets the width and height of this component.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height

	m.viewport.Width = width
	hv := 0
	if m.showHelp {
		hv = lipgloss.Height(m.helpView())
	}
	m.viewport.Height = height - hv
	m.Help.Width = width
}

// ShortHelp returns bindings to show in the abbreviated help view. It's part
// of the help.KeyMap interface.
func (m Model) ShortHelp() []key.Binding {
	kb := []key.Binding{
		m.KeyMap.Down,
		m.KeyMap.Up,
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
			m.KeyMap.Down,
			m.KeyMap.Up,
		},
		{
			m.KeyMap.Toggle,
			m.KeyMap.Open,
			m.KeyMap.Close,
		},
		{
			m.KeyMap.PageDown,
			m.KeyMap.PageUp,
			m.KeyMap.HalfPageDown,
			m.KeyMap.HalfPageUp,
		},
		{
			m.KeyMap.GoToTop,
			m.KeyMap.GoToBottom,
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
