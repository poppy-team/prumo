package layout

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/raillen/prumo-tui/internal/tui/theme"
)

type Container interface {
	tea.Model
	Sizeable
	Bindings
	// SetFocused says whether this surface holds the keyboard. A container that
	// is not focused draws its border differently rather than not at all: the
	// layout cannot shift because focus moved.
	SetFocused(bool)
}

// FocusMsg moves the keyboard to or from the surface that carries a border.
//
// It travels as a message rather than as a call because who holds the keyboard
// is a decision of the shell — a dialog opening takes it away — and the surface
// itself has no way to know.
type FocusMsg struct{ Focused bool }

type container struct {
	width  int
	height int

	content tea.Model

	// Style options
	paddingTop    int
	paddingRight  int
	paddingBottom int
	paddingLeft   int

	borderTop    bool
	borderRight  bool
	borderBottom bool
	borderLeft   bool
	borderStyle  lipgloss.Border

	focused bool
}

// SetFocused records whether this container holds the keyboard.
func (c *container) SetFocused(focused bool) { c.focused = focused }

// focusedBorder thickens every line of a border.
//
// Focus changes the glyph as well as the colour on purpose: the contract
// requires an indicator that survives a terminal without colour, and a border
// that differs only in hue cannot carry it.
func focusedBorder(b lipgloss.Border) lipgloss.Border {
	heavy := map[string]string{
		"─": "━", "│": "┃", "┌": "┏", "┐": "┓", "└": "┗", "┘": "┛",
		"├": "┣", "┤": "┫", "┬": "┳", "┴": "┻", "┼": "╋",
	}
	swap := func(s string) string {
		if thick, ok := heavy[s]; ok {
			return thick
		}
		return s
	}
	return lipgloss.Border{
		Top: swap(b.Top), Bottom: swap(b.Bottom), Left: swap(b.Left), Right: swap(b.Right),
		TopLeft: swap(b.TopLeft), TopRight: swap(b.TopRight),
		BottomLeft: swap(b.BottomLeft), BottomRight: swap(b.BottomRight),
		MiddleLeft: swap(b.MiddleLeft), MiddleRight: swap(b.MiddleRight),
		Middle: swap(b.Middle), MiddleTop: swap(b.MiddleTop), MiddleBottom: swap(b.MiddleBottom),
	}
}

func (c *container) Init() tea.Cmd {
	return c.content.Init()
}

func (c *container) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	u, cmd := c.content.Update(msg)
	c.content = u
	return c, cmd
}

// View renders the component for the terminal.
func (c *container) View() tea.View { return tea.NewView(c.viewString()) }
func (c *container) viewString() string {
	t := theme.CurrentTheme()
	style := lipgloss.NewStyle()
	width := c.width
	height := c.height

	style = style.Background(t.Background())

	// Apply border if any side is enabled
	if c.borderTop || c.borderRight || c.borderBottom || c.borderLeft {
		// Adjust width and height for borders
		if c.borderTop {
			height--
		}
		if c.borderBottom {
			height--
		}
		if c.borderLeft {
			width--
		}
		if c.borderRight {
			width--
		}
		border := c.borderStyle
		foreground := t.BorderNormal()
		if c.focused {
			border = focusedBorder(border)
			foreground = t.BorderFocused()
		}
		style = style.Border(border, c.borderTop, c.borderRight, c.borderBottom, c.borderLeft)
		style = style.BorderBackground(t.Background()).BorderForeground(foreground)
	}
	style = style.
		Width(width).
		Height(height).
		PaddingTop(c.paddingTop).
		PaddingRight(c.paddingRight).
		PaddingBottom(c.paddingBottom).
		PaddingLeft(c.paddingLeft)

	return style.Render(c.content.View().Content)
}

func (c *container) SetSize(width, height int) tea.Cmd {
	c.width = width
	c.height = height

	// If the content implements Sizeable, adjust its size to account for padding and borders
	if sizeable, ok := c.content.(Sizeable); ok {
		// Calculate horizontal space taken by padding and borders
		horizontalSpace := c.paddingLeft + c.paddingRight
		if c.borderLeft {
			horizontalSpace++
		}
		if c.borderRight {
			horizontalSpace++
		}

		// Calculate vertical space taken by padding and borders
		verticalSpace := c.paddingTop + c.paddingBottom
		if c.borderTop {
			verticalSpace++
		}
		if c.borderBottom {
			verticalSpace++
		}

		// Set content size with adjusted dimensions
		contentWidth := max(0, width-horizontalSpace)
		contentHeight := max(0, height-verticalSpace)
		return sizeable.SetSize(contentWidth, contentHeight)
	}
	return nil
}

func (c *container) GetSize() (int, int) {
	return c.width, c.height
}

func (c *container) BindingKeys() []key.Binding {
	if b, ok := c.content.(Bindings); ok {
		return b.BindingKeys()
	}
	return []key.Binding{}
}

type ContainerOption func(*container)

func NewContainer(content tea.Model, options ...ContainerOption) Container {

	c := &container{
		content:     content,
		borderStyle: lipgloss.NormalBorder(),
	}

	for _, option := range options {
		option(c)
	}

	return c
}

// Padding options
func WithPadding(top, right, bottom, left int) ContainerOption {
	return func(c *container) {
		c.paddingTop = top
		c.paddingRight = right
		c.paddingBottom = bottom
		c.paddingLeft = left
	}
}

func WithPaddingAll(padding int) ContainerOption {
	return WithPadding(padding, padding, padding, padding)
}

func WithPaddingHorizontal(padding int) ContainerOption {
	return func(c *container) {
		c.paddingLeft = padding
		c.paddingRight = padding
	}
}

func WithPaddingVertical(padding int) ContainerOption {
	return func(c *container) {
		c.paddingTop = padding
		c.paddingBottom = padding
	}
}

func WithBorder(top, right, bottom, left bool) ContainerOption {
	return func(c *container) {
		c.borderTop = top
		c.borderRight = right
		c.borderBottom = bottom
		c.borderLeft = left
	}
}

func WithBorderAll() ContainerOption {
	return WithBorder(true, true, true, true)
}

func WithBorderHorizontal() ContainerOption {
	return WithBorder(true, false, true, false)
}

func WithBorderVertical() ContainerOption {
	return WithBorder(false, true, false, true)
}

func WithBorderStyle(style lipgloss.Border) ContainerOption {
	return func(c *container) {
		c.borderStyle = style
	}
}

func WithRoundedBorder() ContainerOption {
	return WithBorderStyle(lipgloss.RoundedBorder())
}

func WithThickBorder() ContainerOption {
	return WithBorderStyle(lipgloss.ThickBorder())
}

func WithDoubleBorder() ContainerOption {
	return WithBorderStyle(lipgloss.DoubleBorder())
}
