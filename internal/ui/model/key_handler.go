package model

import (
	"strings"
	"unicode"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/liamb/aide/internal/message"
	"github.com/liamb/aide/internal/ui/completions"
	"github.com/liamb/aide/internal/ui/dialog"
	"github.com/liamb/aide/internal/ui/util"
)

func (m *UI) handleKeyPressMsg(msg tea.KeyPressMsg) tea.Cmd {
	var cmds []tea.Cmd

	handleGlobalKeys := func(msg tea.KeyPressMsg) bool {
		switch {
		case key.Matches(msg, m.keyMap.Help):
			m.status.ToggleHelp()
			m.updateLayoutAndSize()
			return true
		case key.Matches(msg, m.keyMap.Commands):
			if cmd := m.openCommandsDialog(); cmd != nil {
				cmds = append(cmds, cmd)
			}
			return true
		case key.Matches(msg, m.keyMap.Models):
			if cmd := m.openModelsDialog(); cmd != nil {
				cmds = append(cmds, cmd)
			}
			return true
		case key.Matches(msg, m.keyMap.Sessions):
			if cmd := m.openSessionsDialog(); cmd != nil {
				cmds = append(cmds, cmd)
			}
			return true
		case key.Matches(msg, m.keyMap.Chat.Details) && m.isCompact:
			m.detailsOpen = !m.detailsOpen
			m.updateLayoutAndSize()
			return true
		case key.Matches(msg, m.keyMap.Chat.TogglePills):
			if m.state == uiChat && m.hasSession() {
				if cmd := m.togglePillsExpanded(); cmd != nil {
					cmds = append(cmds, cmd)
				}
				return true
			}
		case key.Matches(msg, m.keyMap.Chat.PillLeft):
			if m.state == uiChat && m.hasSession() && m.pillsExpanded && m.focus != uiFocusEditor {
				if cmd := m.switchPillSection(-1); cmd != nil {
					cmds = append(cmds, cmd)
				}
				return true
			}
		case key.Matches(msg, m.keyMap.Chat.PillRight):
			if m.state == uiChat && m.hasSession() && m.pillsExpanded && m.focus != uiFocusEditor {
				if cmd := m.switchPillSection(1); cmd != nil {
					cmds = append(cmds, cmd)
				}
				return true
			}
		case key.Matches(msg, m.keyMap.Suspend):
			if m.isAgentBusy() {
				cmds = append(cmds, util.ReportWarn("Agent is busy, please wait..."))
				return true
			}
			cmds = append(cmds, tea.Suspend)
			return true
		case key.Matches(msg, m.keyMap.ToggleYolo):
			yolo := !m.com.Workspace.PermissionSkipRequests()
			m.com.Workspace.PermissionSetSkipRequests(yolo)
			m.setEditorPrompt(yolo)
			status := "disabled"
			if yolo {
				status = "enabled"
			}
			cmds = append(cmds, util.ReportInfo("Yolo mode "+status))
			return true
		}
		return false
	}

	if key.Matches(msg, m.keyMap.Quit) && !m.dialog.ContainsDialog(dialog.QuitID) {
		// Always handle quit keys first
		if cmd := m.openQuitDialog(); cmd != nil {
			cmds = append(cmds, cmd)
		}

		return tea.Batch(cmds...)
	}

	// Route all messages to dialog if one is open.
	if m.dialog.HasDialogs() {
		return m.handleDialogMsg(msg)
	}

	// Handle Switch Agent key globally (when no dialog is open).
	if key.Matches(msg, m.keyMap.SwitchAgent) {
		agents := m.availableAgents()
		if len(agents) > 1 {
			nextAgent := agents[0]
			for i, a := range agents {
				if a == m.agentMode && i+1 < len(agents) {
					nextAgent = agents[i+1]
					break
				}
			}
			cmds = append(cmds, m.switchAgent(nextAgent))
		}
		return tea.Batch(cmds...)
	}

	// Handle cancel key when agent is busy.
	if key.Matches(msg, m.keyMap.Chat.Cancel) {
		if m.isAgentBusy() {
			if cmd := m.cancelAgent(); cmd != nil {
				cmds = append(cmds, cmd)
			}
			return tea.Batch(cmds...)
		}
	}

	switch m.state {
	case uiOnboarding:
		return tea.Batch(cmds...)
	case uiInitialize:
		cmds = append(cmds, m.updateInitializeView(msg)...)
		return tea.Batch(cmds...)
	case uiChat, uiLanding:
		switch m.focus {
		case uiFocusEditor:
			// Handle completions if open.
			if m.completionsOpen {
				if msg, ok := m.completions.Update(msg); ok {
					switch msg := msg.(type) {
					case completions.SelectionMsg[completions.FileCompletionValue]:
						cmds = append(cmds, m.insertFileCompletion(msg.Value.Path))
						if !msg.KeepOpen {
							m.closeCompletions()
						}
					case completions.SelectionMsg[completions.ResourceCompletionValue]:
						cmds = append(cmds, m.insertMCPResourceCompletion(msg.Value))
						if !msg.KeepOpen {
							m.closeCompletions()
						}
					case completions.ClosedMsg:
						m.completionsOpen = false
					}
					return tea.Batch(cmds...)
				}
			}

			if ok := m.attachments.Update(msg); ok {
				return tea.Batch(cmds...)
			}

			switch {
			case key.Matches(msg, m.keyMap.Editor.AddImage):
				if !m.currentModelSupportsImages() {
					break
				}
				if cmd := m.openFilesDialog(); cmd != nil {
					cmds = append(cmds, cmd)
				}

			case key.Matches(msg, m.keyMap.Editor.PasteImage):
				if !m.currentModelSupportsImages() {
					break
				}
				cmds = append(cmds, m.pasteImageFromClipboard)

			case key.Matches(msg, m.keyMap.Editor.SendMessage):
				prevHeight := m.textarea.Height()
				value := m.textarea.Value()
				if before, ok := strings.CutSuffix(value, "\\"); ok {
					// If the last character is a backslash, remove it and add a newline.
					m.textarea.SetValue(before)
					if cmd := m.handleTextareaHeightChange(prevHeight); cmd != nil {
						cmds = append(cmds, cmd)
					}
					break
				}

				// Otherwise, send the message
				m.textarea.Reset()
				if cmd := m.handleTextareaHeightChange(prevHeight); cmd != nil {
					cmds = append(cmds, cmd)
				}

				value = strings.TrimSpace(value)
				if value == "exit" || value == "quit" {
					return m.openQuitDialog()
				}

				if m.bangMode && value != "" {
					m.bangMode = false
					yolo := m.com.Workspace.PermissionSkipRequests()
					m.setEditorPrompt(yolo)
					m.randomizePlaceholders()
					m.historyReset()
					return tea.Batch(m.runShellCommand(value))
				}

				attachments := m.attachments.List()
				m.attachments.Reset()
				if len(value) == 0 && !message.ContainsTextAttachment(attachments) {
					return nil
				}

				m.randomizePlaceholders()
				m.historyReset()

				return tea.Batch(m.sendMessage(value, attachments...), m.loadPromptHistory())
			case key.Matches(msg, m.keyMap.Chat.NewSession):
				if !m.hasSession() {
					break
				}
				if m.isAgentBusy() {
					cmds = append(cmds, util.ReportWarn("Agent is busy, please wait before starting a new session..."))
					break
				}
				if cmd := m.newSession(); cmd != nil {
					cmds = append(cmds, cmd)
				}
			case key.Matches(msg, m.keyMap.Tab):
				if m.state != uiLanding {
					m.setState(m.state, uiFocusMain)
					m.textarea.Blur()
					m.chat.Focus()
					m.chat.SetSelected(m.chat.Len() - 1)
				}
			case key.Matches(msg, m.keyMap.SwitchAgent):
				agents := m.availableAgents()
				if len(agents) > 1 {
					nextAgent := agents[0]
					for i, a := range agents {
						if a == m.agentMode && i+1 < len(agents) {
							nextAgent = agents[i+1]
							break
						}
					}
					cmds = append(cmds, m.switchAgent(nextAgent))
				}
			case key.Matches(msg, m.keyMap.Editor.OpenEditor):
				if m.isAgentBusy() {
					cmds = append(cmds, util.ReportWarn("Agent is working, please wait..."))
					break
				}
				editorValue := m.textarea.Value()
				if m.bangMode {
					editorValue = "!" + editorValue
				}
				cmds = append(cmds, m.openEditor(editorValue))
			case key.Matches(msg, m.keyMap.Editor.Newline):
				prevHeight := m.textarea.Height()
				m.textarea.InsertRune('\n')
				m.closeCompletions()
				cmds = append(cmds, m.updateTextareaWithPrevHeight(msg, prevHeight))
			case key.Matches(msg, m.keyMap.Editor.HistoryPrev):
				cmd := m.handleHistoryUp(msg)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			case key.Matches(msg, m.keyMap.Editor.HistoryNext):
				cmd := m.handleHistoryDown(msg)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			case key.Matches(msg, m.keyMap.Editor.Escape):
				cmd := m.handleHistoryEscape(msg)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			case key.Matches(msg, m.keyMap.Editor.Commands) && m.textarea.Value() == "":
				if cmd := m.openCommandsDialog(); cmd != nil {
					cmds = append(cmds, cmd)
				}
			default:
				if handleGlobalKeys(msg) {
					// Handle global keys first before passing to textarea.
					break
				}

				// Bang mode: backspace on already-empty prompt exits.
				if m.bangMode && m.bangWasEmpty && msg.Code == tea.KeyBackspace {
					m.bangMode = false
					m.bangWasEmpty = false
					yolo := m.com.Workspace.PermissionSkipRequests()
					m.setEditorPrompt(yolo)
					break
				}

				// Check for @ trigger before passing to textarea.
				curValue := m.textarea.Value()
				curIdx := len(curValue)

				// Trigger completions on @.
				if msg.String() == "@" && !m.completionsOpen {
					// Only show if beginning of prompt or after whitespace.
					if curIdx == 0 || (curIdx > 0 && isWhitespace(curValue[curIdx-1])) {
						m.completionsOpen = true
						m.completionsQuery = ""
						m.completionsStartIndex = curIdx
						m.completionsPositionStart = m.completionsPosition()
						depth, limit := m.com.Config().Options.TUI.Completions.Limits()
						cmds = append(cmds, m.completions.Open(depth, limit))
					}
				}

				// remove the details if they are open when user starts typing
				if m.detailsOpen {
					m.detailsOpen = false
					m.updateLayoutAndSize()
				}

				prevHeight := m.textarea.Height()
				cmds = append(cmds, m.updateTextareaWithPrevHeight(msg, prevHeight))

				// Bang mode: enter when "!" is typed at the start of the
				// prompt, optionally preceded by whitespace (either on an
				// empty/whitespace-only prompt or prepended to existing text).
				// Exit on backspace clearing the last character.
				newVal := m.textarea.Value()
				trimmedNew := strings.TrimLeftFunc(newVal, unicode.IsSpace)
				trimmedCur := strings.TrimLeftFunc(curValue, unicode.IsSpace)
				if !m.bangMode && strings.HasPrefix(trimmedNew, "!") && !strings.HasPrefix(trimmedCur, "!") {
					m.bangMode = true
					m.bangWasEmpty = len(strings.TrimSpace(curValue)) == 0
					// Strip leading whitespace and the "!" from the textarea
					// while preserving the cursor position relative to the
					// command text.
					col := m.textarea.Column()
					line := m.textarea.Line()
					stripped := trimmedNew[1:]
					m.textarea.SetValue(stripped)
					m.textarea.SetCursorColumn(max(0, col-(len(newVal)-len(stripped))))
					_ = line // cursor line doesn't change; prefix removed
					yolo := m.com.Workspace.PermissionSkipRequests()
					m.setEditorPrompt(yolo)
				} else if m.bangMode && newVal == "" && curValue != "" {
					// Just cleared last character; mark empty, stay in bang mode.
					m.bangWasEmpty = true
				} else if m.bangMode && newVal != "" {
					m.bangWasEmpty = false
				}

				// Any text modification becomes the current draft.
				m.updateHistoryDraft(curValue)

				// After updating textarea, check if we need to filter completions.
				// Skip filtering on the initial @ keystroke since items are loading async.
				if m.completionsOpen && msg.String() != "@" {
					newValue := m.textarea.Value()
					newIdx := len(newValue)

					// Close completions if cursor moved before start.
					if newIdx <= m.completionsStartIndex {
						m.closeCompletions()
					} else if msg.String() == "space" {
						// Close on space.
						m.closeCompletions()
					} else {
						// Extract current word and filter.
						word := m.textareaWord()
						if strings.HasPrefix(word, "@") {
							m.completionsQuery = word[1:]
							m.completions.Filter(m.completionsQuery)
						} else if m.completionsOpen {
							m.closeCompletions()
						}
					}
				}
			}
		case uiFocusMain:
			switch {
			case key.Matches(msg, m.keyMap.Tab):
				m.focus = uiFocusEditor
				cmds = append(cmds, m.textarea.Focus())
				m.chat.Blur()
			case key.Matches(msg, m.keyMap.SwitchAgent):
				agents := m.availableAgents()
				if len(agents) > 1 {
					nextAgent := agents[0]
					for i, a := range agents {
						if a == m.agentMode && i+1 < len(agents) {
							nextAgent = agents[i+1]
							break
						}
					}
					cmds = append(cmds, m.switchAgent(nextAgent))
				}
			case key.Matches(msg, m.keyMap.Chat.NewSession):
				if !m.hasSession() {
					break
				}
				if m.isAgentBusy() {
					cmds = append(cmds, util.ReportWarn("Agent is busy, please wait before starting a new session..."))
					break
				}
				m.focus = uiFocusEditor
				if cmd := m.newSession(); cmd != nil {
					cmds = append(cmds, cmd)
				}
			case key.Matches(msg, m.keyMap.Chat.Expand):
				m.chat.ToggleExpandedSelectedItem()
			case key.Matches(msg, m.keyMap.Chat.Up):
				if cmd := m.chat.ScrollByAndAnimate(-1); cmd != nil {
					cmds = append(cmds, cmd)
				}
				if !m.chat.SelectedItemInView() {
					m.chat.SelectPrev()
					if cmd := m.chat.ScrollToSelectedAndAnimate(); cmd != nil {
						cmds = append(cmds, cmd)
					}
				}
			case key.Matches(msg, m.keyMap.Chat.Down):
				if cmd := m.chat.ScrollByAndAnimate(1); cmd != nil {
					cmds = append(cmds, cmd)
				}
				if !m.chat.SelectedItemInView() {
					m.chat.SelectNext()
					if cmd := m.chat.ScrollToSelectedAndAnimate(); cmd != nil {
						cmds = append(cmds, cmd)
					}
				}
			case key.Matches(msg, m.keyMap.Chat.UpOneItem):
				m.chat.SelectPrev()
				if cmd := m.chat.ScrollToSelectedAndAnimate(); cmd != nil {
					cmds = append(cmds, cmd)
				}
			case key.Matches(msg, m.keyMap.Chat.DownOneItem):
				m.chat.SelectNext()
				if cmd := m.chat.ScrollToSelectedAndAnimate(); cmd != nil {
					cmds = append(cmds, cmd)
				}
			case key.Matches(msg, m.keyMap.Chat.HalfPageUp):
				if cmd := m.chat.ScrollByAndAnimate(-m.chat.Height() / 2); cmd != nil {
					cmds = append(cmds, cmd)
				}
				m.chat.SelectFirstInView()
			case key.Matches(msg, m.keyMap.Chat.HalfPageDown):
				if cmd := m.chat.ScrollByAndAnimate(m.chat.Height() / 2); cmd != nil {
					cmds = append(cmds, cmd)
				}
				m.chat.SelectLastInView()
			case key.Matches(msg, m.keyMap.Chat.PageUp):
				if cmd := m.chat.ScrollByAndAnimate(-m.chat.Height()); cmd != nil {
					cmds = append(cmds, cmd)
				}
				m.chat.SelectFirstInView()
			case key.Matches(msg, m.keyMap.Chat.PageDown):
				if cmd := m.chat.ScrollByAndAnimate(m.chat.Height()); cmd != nil {
					cmds = append(cmds, cmd)
				}
				m.chat.SelectLastInView()
			case key.Matches(msg, m.keyMap.Chat.Home):
				if cmd := m.chat.ScrollToTopAndAnimate(); cmd != nil {
					cmds = append(cmds, cmd)
				}
				m.chat.SelectFirst()
			case key.Matches(msg, m.keyMap.Chat.End):
				if cmd := m.chat.ScrollToBottomAndAnimate(); cmd != nil {
					cmds = append(cmds, cmd)
				}
				m.chat.SelectLast()
			default:
				if ok, cmd := m.chat.HandleKeyMsg(msg); ok {
					cmds = append(cmds, cmd)
				} else {
					handleGlobalKeys(msg)
				}
			}
		default:
			handleGlobalKeys(msg)
		}
	default:
		handleGlobalKeys(msg)
	}

	return tea.Sequence(cmds...)
}
