package main

import (
	. "github.com/julvo/htmlgo"
	a "github.com/julvo/htmlgo/attributes"
)

func queuePage(url string) HTML {
	if url == "" {
		return page(
			Div(
				Attr(a.Id("wrapper")),
				H1_(Text("Well done - queue is empty!"))))
	}

	return page(
		Div(
			Attr(a.Id("wrapper")),
			H1_(Text("DCR Diff - Queue")),
			A(Attr(a.Id("link"), a.Href(url)), Text(url)),
			Iframe(Attr(a.Id("frontend"), a.Src("/proxy?url="+url))),

			Div(
				Attr(a.Id("dcr-wrapper")),
				Div(Attr(a.Id("dcr-label")), Text_("Dotcom Rendering")),
				Iframe(Attr(a.Id("dcr"), a.Src("/proxy?url="+url+"?dcr"))),
			),
		),

		Div(
			Attr(a.Id("queueForm")),
			acceptForm(),
			rejectForm(),
		),
	)
}

func page(inner ...HTML) HTML {
	return Html5_(
		Head_(
			Meta(Attr(a.Charset("utf-8"))),
			Link(Attr(a.Rel("icon"), a.Href("https://static.guim.co.uk/images/favicon-32x32-dev-yellow.ico"))),
			Link(Attr(a.Rel("stylesheet"), a.Href("/static/main.css"))),
		),
		Body_(inner...),
	)
}

func acceptForm() HTML {
	return Form(
		Attr(a.Method("post")),
		Button(
			Attr(a.Type_("submit"), a.Value("accept")),
			Text_("✓"),
		),
	)
}

func rejectForm() HTML {
	return Form(
		Attr(a.Method("post")),
		Input(Attr(a.Type("text"), a.Name("comment"), a.Required("true"), a.Placeholder("Comment..."))),
		Button(
			Attr(a.Type_("submit"), a.Value("reject")),
			Text_("✕"),
		),
	)
}
