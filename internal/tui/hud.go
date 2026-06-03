package tui

import (
	"strings"
)

type hudBinding struct {
	key   string
	label string
}

func renderHUD(width int, bindings ...hudBinding) string {
	if width < 10 {
		width = 80
	}
	var parts []string
	for _, b := range bindings {
		parts = append(parts, hudKeyStyle.Render(b.key)+" "+hudLabelStyle.Render(b.label))
	}
	line := strings.Join(parts, hudSepStyle.Render(" │ "))
	return footerStyle.Width(width).Render(line)
}

func chatHUD(m chatModel) string {
	if m.modal.visible {
		bindings := []hudBinding{
			{"enter", "send request"},
			{"esc", "close"},
		}
		if m.modal.loading {
			bindings = []hudBinding{{"…", "sending request"}, {"esc", "close"}}
		}
		return renderHUD(m.width, bindings...)
	}

	common := []hudBinding{
		{"1", "friends"},
		{"2", "incoming"},
		{"3", "outgoing"},
		{"↑↓", "select"},
		{"a", "add friend"},
	}

	switch m.sidebarTab {
	case tabIncoming:
		common = append(common, hudBinding{"enter", "accept"})
	case tabOutgoing:
		common = append(common, hudBinding{"enter", "—"})
	default:
		common = append(common, hudBinding{"enter", "send"})
	}

	common = append(common, hudBinding{"q", "quit"}, hudBinding{"ctrl+c", "quit"})
	return renderHUD(m.width, common...)
}

func welcomeHUD(width int) string {
	return renderHUD(width,
		hudBinding{"l", "login"},
		hudBinding{"r", "register"},
		hudBinding{"q", "quit"},
		hudBinding{"ctrl+c", "quit"},
	)
}

func authHUD(width int) string {
	return renderHUD(width,
		hudBinding{"enter", "submit"},
		hudBinding{"tab", "next field"},
		hudBinding{"esc", "back"},
		hudBinding{"ctrl+c", "quit"},
	)
}
