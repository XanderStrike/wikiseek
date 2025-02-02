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

// Register default template handlers
func init() {
	// Short description template handler
	RegisterTemplateHandler("short description", func(args []string) string {
		if len(args) > 0 {
			return `<em>` + args[0] + `</em>`
		}
		return ""
	})

	// See also template handler
	RegisterTemplateHandler("see also", func(args []string) string {
		if len(args) == 0 {
			return `<div class="note">See Also</div>`
		}

		var links []string
		for _, arg := range args {
			links = append(links, `<a href="`+arg+`">`+arg+`</a>`)
		}

		return `<div class="note">See Also: ` + strings.Join(links, ", ") + `</div>`
	})

	// Other uses template handler
	RegisterTemplateHandler("other uses", func(args []string) string {
		if len(args) == 0 {
			return `<div class="note">Other Uses</div>`
		}

		var links []string
		for _, arg := range args {
			links = append(links, `<a href="`+arg+`">`+arg+`</a>`)
		}

		return `<div class="note">Other Uses: ` + strings.Join(links, ", ") + `</div>`
	})
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

			// Process any nested templates in the arguments
			for i, arg := range args {
				args[i] = ConvertWikiTextToHTML(arg)
			}
		}

		// Get handler or use default
		handler, exists := templateHandlers[strings.ToLower(templateName)]
		if !exists {
			// Default handler for unknown templates
			handler = func(args []string) string {
				return `<div style="color:red">No match for "` + templateName + `": ` + fullMatch + `</div>`
			}
		}

		// Replace template with processed content
		content = strings.Replace(content, fullMatch, handler(args), 1)
	}

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
