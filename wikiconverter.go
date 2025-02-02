package main

// TemplateHandler processes a template and returns HTML
type TemplateHandler func([]string) string

// ConvertWikiTextToHTML converts wikitext content to HTML
func ConvertWikiTextToHTML(content string) string {

	// wikitext has templates like {{foo}} that are automatically replaced with
	// specific content, where foo is the name of the template

	//templates can take arguments separated by bars | so {{see also|bonjour}}
	//would render the "see also" template with the first argument as "bonjour"

	// template calls cam be multiple lines long, for instance this template:
	// {{Infobox
	// | name     = {{{name|{{PAGENAME}}}}}
	// | image    = {{{image|}}}
	// | caption1 = {{{caption|}}}

	// | label1   = Former names
	// |  data1   = {{{former_names|}}}

	// | header2  = General information

	// | label3   = Status
	// |  data3   = {{{status|}}}
	// ... <!-- etc. -->
	// }}
	// is one template and should be handled all at once

	// this needs to call a method that identifies all the templates in the
	// string and calls a different method for each

	// also needs to handle an unrecognized template, for now just color it red

	return content
}
