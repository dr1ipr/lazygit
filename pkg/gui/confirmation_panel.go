// lots of this has been directly ported from one of the example files, will brush up later

// Copyright 2014 The gocui Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gui

import (
	"strings"

	"github.com/fatih/color"
	"github.com/jesseduffield/gocui"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

func (gui *Gui) wrappedConfirmationFunction(function func(*gocui.Gui, *gocui.View) error) func(*gocui.Gui, *gocui.View) error {
	return func(g *gocui.Gui, v *gocui.View) error {
		if function != nil {
			if err := function(g, v); err != nil {
				panic(err)
			}
		}
		return gui.closeConfirmationPrompt(g)
	}
}

func (gui *Gui) closeConfirmationPrompt(g *gocui.Gui) error {
	view, err := g.View("confirmation")
	if err != nil {
		panic(err)
	}
	if err := gui.returnFocus(g, view); err != nil {
		panic(err)
	}
	g.DeleteKeybindings("confirmation")
	return g.DeleteView("confirmation")
}

func (gui *Gui) getMessageHeight(message string, width int) int {
	lines := strings.Split(message, "\n")
	lineCount := 0
	for _, line := range lines {
		lineCount += len(line)/width + 1
	}
	return lineCount
}

func (gui *Gui) getConfirmationPanelDimensions(g *gocui.Gui, prompt string) (int, int, int, int) {
	width, height := g.Size()
	panelWidth := width / 2
	panelHeight := gui.getMessageHeight(prompt, panelWidth)
	return width/2 - panelWidth/2,
		height/2 - panelHeight/2 - panelHeight%2 - 1,
		width/2 + panelWidth/2,
		height/2 + panelHeight/2
}

func (gui *Gui) createPromptPanel(g *gocui.Gui, currentView *gocui.View, title string, handleConfirm func(*gocui.Gui, *gocui.View) error) error {
	gui.onNewPopupPanel()
	// only need to fit one line
	x0, y0, x1, y1 := gui.getConfirmationPanelDimensions(g, "")
	if confirmationView, err := g.SetView("confirmation", x0, y0, x1, y1, 0); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		confirmationView.Editable = true
		confirmationView.Title = title
		gui.switchFocus(g, currentView, confirmationView)
		return gui.setKeyBindings(g, handleConfirm, nil)
	}
	return nil
}

func (gui *Gui) onNewPopupPanel() {
	gui.g.SetViewOnBottom("commitMessage")
}

func (gui *Gui) createConfirmationPanel(g *gocui.Gui, currentView *gocui.View, title, prompt string, handleConfirm, handleClose func(*gocui.Gui, *gocui.View) error) error {
	gui.onNewPopupPanel()
	g.Update(func(g *gocui.Gui) error {
		// delete the existing confirmation panel if it exists
		if view, _ := g.View("confirmation"); view != nil {
			if err := gui.closeConfirmationPrompt(g); err != nil {
				gui.Log.Error("Could not close confirmation prompt: ", err.Error())
			}
		}
		x0, y0, x1, y1 := gui.getConfirmationPanelDimensions(g, prompt)
		if confirmationView, err := g.SetView("confirmation", x0, y0, x1, y1, 0); err != nil {
			if err != gocui.ErrUnknownView {
				return err
			}
			confirmationView.Title = title
			confirmationView.FgColor = gocui.ColorWhite
			gui.renderString(g, "confirmation", prompt)
			gui.switchFocus(g, currentView, confirmationView)
			return gui.setKeyBindings(g, handleConfirm, handleClose)
		}
		return nil
	})
	return nil
}

func (gui *Gui) handleNewline(g *gocui.Gui, v *gocui.View) error {
	// resising ahead of time so that the top line doesn't get hidden to make
	// room for the cursor on the second line
	x0, y0, x1, y1 := gui.getConfirmationPanelDimensions(g, v.Buffer())
	if _, err := g.SetView("confirmation", x0, y0, x1, y1+1, 0); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
	}

	v.EditNewLine()
	return nil
}

func (gui *Gui) setKeyBindings(g *gocui.Gui, handleConfirm, handleClose func(*gocui.Gui, *gocui.View) error) error {
	gui.renderString(g, "options", "esc: close, enter: confirm")
	if err := g.SetKeybinding("confirmation", gocui.KeyEnter, gocui.ModNone, gui.wrappedConfirmationFunction(handleConfirm)); err != nil {
		return err
	}
	if err := g.SetKeybinding("confirmation", gocui.KeyTab, gocui.ModNone, gui.handleNewline); err != nil {
		return err
	}
	return g.SetKeybinding("confirmation", gocui.KeyEsc, gocui.ModNone, gui.wrappedConfirmationFunction(handleClose))
}

func (gui *Gui) createMessagePanel(g *gocui.Gui, currentView *gocui.View, title, prompt string) error {
	return gui.createConfirmationPanel(g, currentView, title, prompt, nil, nil)
}

func (gui *Gui) createErrorPanel(g *gocui.Gui, message string) error {
	currentView := g.CurrentView()
	colorFunction := color.New(color.FgRed).SprintFunc()
	coloredMessage := colorFunction(strings.TrimSpace(message))
	return gui.createConfirmationPanel(g, currentView, "Error", coloredMessage, nil, nil)
}

func (gui *Gui) resizePopupPanel(g *gocui.Gui, v *gocui.View) error {
	// If the confirmation panel is already displayed, just resize the width,
	// otherwise continue
	content := utils.TrimTrailingNewline(v.Buffer())
	x0, y0, x1, y1 := gui.getConfirmationPanelDimensions(g, content)
	vx0, vy0, vx1, vy1 := v.Dimensions()
	if vx0 == x0 && vy0 == y0 && vx1 == x1 && vy1 == y1 {
		return nil
	}
	gui.Log.Info("resizing popup panel")
	_, err := g.SetView(v.Name(), x0, y0, x1, y1, 0)
	return err
}
