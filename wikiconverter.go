package main

import (
	"regexp"
	"strings"
)

// TemplateHandler processes a template and returns HTML
type TemplateHandler func([]string) string

var (
	templateHandlers = make(map[string]TemplateHandler)
	
	// Matches {{template}} or {{template|arg1|arg2}}
	templatePattern = regexp.MustCompile(`(?s)\{\{(.*?)\}\}`)
	
	// Matches template name and arguments
	templateArgsPattern = regexp.MustCompile(`(?s)^([^{}|]+)(?:\|(.*))?$`)
)

// RegisterTemplateHandler registers a handler for a specific template name
func RegisterTemplateHandler(name string, handler TemplateHandler) {
	templateHandlers[name] = handler
}

// ConvertWikiTextToHTML converts wikitext content to HTML
func ConvertWikiTextToHTML(content string) string {
	// Find all template matches
	matches := templatePattern.FindAllStringSubmatch(content, -1)
	
	for _, match := range matches {
		fullMatch := match[0]
		innerContent := match[1]
		
		// Parse template name and arguments
		argsMatch := templateArgsPattern.FindStringSubmatch(innerContent)
		if len(argsMatch) < 2 {
			continue
		}
		
		templateName := strings.TrimSpace(argsMatch[1])
		var args []string
		if len(argsMatch) > 2 && argsMatch[2] != "" {
			// Split arguments by | but handle multi-line arguments
			args = parseTemplateArguments(argsMatch[2])
		}
		
		// Get handler or use default
		handler, exists := templateHandlers[templateName]
		if !exists {
			// Default handler for unknown templates
			handler = func(args []string) string {
				return `<span style="color:red">` + fullMatch + `</span>`
			}
		}
		
		// Replace template with processed content
		content = strings.Replace(content, fullMatch, handler(args), 1)
	}

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

// parseTemplateArguments splits template arguments while handling multi-line arguments
func parseTemplateArguments(argString string) []string {
	var args []string
	var currentArg strings.Builder
	braceLevel := 0
	
	for _, r := range argString {
		switch r {
		case '{':
			braceLevel++
		case '}':
			braceLevel--
		case '|':
			if braceLevel == 0 {
				args = append(args, strings.TrimSpace(currentArg.String()))
				currentArg.Reset()
				continue
			}
		}
		currentArg.WriteRune(r)
	}
	
	// Add the last argument
	if currentArg.Len() > 0 {
		args = append(args, strings.TrimSpace(currentArg.String()))
	}
	
	return args
}
