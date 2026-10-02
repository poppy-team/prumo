package theme

// SoftLightTheme keeps the daylight surface quiet while preserving readable text.
type SoftLightTheme struct {
	BaseTheme
}

func NewSoftLightTheme() *SoftLightTheme {
	t := &SoftLightTheme{BaseTheme: NewFlexokiTheme().BaseTheme}
	color := func(dark, light string) AdaptiveColor {
		return AdaptiveColor{Dark: dark, Light: light}
	}

	t.BackgroundColor = color("#202926", "#faf9f5")
	t.BackgroundSecondaryColor = color("#29332f", "#f1f1eb")
	t.BackgroundDarkerColor = color("#17201d", "#e7eae3")
	t.TextColor = color("#e0e6de", "#303d38")
	t.TextMutedColor = color("#a2b0a7", "#65716b")
	t.TextEmphasizedColor = color("#f0eddf", "#273b37")
	t.PrimaryColor = color("#a4cbbd", "#366b5d")
	t.SecondaryColor = color("#b7b5d5", "#655c85")
	t.AccentColor = color("#dec4a5", "#825d39")
	t.ErrorColor = color("#e6a7a2", "#a74342")
	t.WarningColor = color("#e4c393", "#845e29")
	t.SuccessColor = color("#a8c995", "#4c742f")
	t.InfoColor = color("#9bc5d1", "#356e80")
	t.BorderNormalColor = color("#52625a", "#929e96")
	t.BorderFocusedColor = t.PrimaryColor
	t.BorderDimColor = color("#3b4941", "#cbd3ca")

	t.DiffAddedColor = t.SuccessColor
	t.DiffRemovedColor = t.ErrorColor
	t.DiffContextColor = t.TextMutedColor
	t.DiffHunkHeaderColor = t.PrimaryColor
	t.DiffHighlightAddedColor = t.SuccessColor
	t.DiffHighlightRemovedColor = t.ErrorColor
	t.DiffAddedBgColor = color("#263a2e", "#ebf3e6")
	t.DiffRemovedBgColor = color("#3c2c2b", "#f7eae8")
	t.DiffContextBgColor = t.BackgroundColor
	t.DiffLineNumberColor = t.TextMutedColor
	t.DiffAddedLineNumberBgColor = t.DiffAddedBgColor
	t.DiffRemovedLineNumberBgColor = t.DiffRemovedBgColor

	t.MarkdownTextColor = t.TextColor
	t.MarkdownHeadingColor = t.PrimaryColor
	t.MarkdownLinkColor = t.InfoColor
	t.MarkdownLinkTextColor = t.PrimaryColor
	t.MarkdownCodeColor = t.SuccessColor
	t.MarkdownBlockQuoteColor = t.TextMutedColor
	t.MarkdownEmphColor = t.AccentColor
	t.MarkdownStrongColor = t.TextEmphasizedColor
	t.MarkdownHorizontalRuleColor = t.BorderNormalColor
	t.MarkdownListItemColor = t.PrimaryColor
	t.MarkdownListEnumerationColor = t.PrimaryColor
	t.MarkdownImageColor = t.SecondaryColor
	t.MarkdownImageTextColor = t.InfoColor
	t.MarkdownCodeBlockColor = t.TextColor

	t.SyntaxCommentColor = t.TextMutedColor
	t.SyntaxKeywordColor = t.SecondaryColor
	t.SyntaxFunctionColor = t.PrimaryColor
	t.SyntaxVariableColor = t.TextColor
	t.SyntaxStringColor = t.SuccessColor
	t.SyntaxNumberColor = t.AccentColor
	t.SyntaxTypeColor = t.InfoColor
	t.SyntaxOperatorColor = t.TextMutedColor
	t.SyntaxPunctuationColor = t.TextColor
	return t
}

func init() {
	RegisterTheme("softlight", NewSoftLightTheme())
}
