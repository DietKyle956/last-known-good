package tui

func renderInput(prompt, text, cursor string, width int) string {
	promptStyled := InputPromptStyle.Render(prompt)
	textStyled := InputTextStyle.Render(text)
	inputLine := promptStyled + textStyled + cursor
	return InputBarStyle.Width(width).Render(inputLine)
}
