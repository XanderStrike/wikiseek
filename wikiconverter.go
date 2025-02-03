package main

import (
	"regexp"
	"strings"
)

// TemplateHandler processes a template and returns HTML
type TemplateHandler func([]string) string

var (
	templateHandlers = make(map[string]TemplateHandler)

	// Matches header patterns like === Header === with symmetrical equals signs
	headerPattern = regexp.MustCompile(`(?m)^(={1,6})\s*(.+?)\s*={1,6}$`)

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

	// Other uses template handler
	RegisterTemplateHandler("main", func(args []string) string {
		if len(args) == 0 {
			return `<div class="note">Main article</div>`
		}

		var links []string
		for _, arg := range args {
			links = append(links, `<a href="`+arg+`">`+arg+`</a>`)
		}

		return `<div class="note">Main article: ` + strings.Join(links, ", ") + `</div>`
	})

	// For template handler
	RegisterTemplateHandler("for", func(args []string) string {
		if len(args) < 2 {
			return ""
		}
		return `<div class="note">For ` + args[0] + `, see <a href="` + args[1] + `">` + args[1] + `</a></div>`
	})

	// Templates to completely ignore/skip
	RegisterTemplateHandler("redirect", func(args []string) string {
		return ""
	})
	RegisterTemplateHandler("good page", func(args []string) string {
		return ""
	})
	RegisterTemplateHandler("pp-blp", func(args []string) string {
		return ""
	})
	RegisterTemplateHandler("use mdy dates", func(args []string) string {
		return ""
	})
	RegisterTemplateHandler("use american english", func(args []string) string {
		return ""
	})

	// Cite web template handler
	// Generic citation handler function
	citationHandler := func(citeType string) TemplateHandler {
		return func(args []string) string {
			if len(args) == 0 {
				return "*"
			}

			// Build table rows from key=value pairs
			var rows []string
			for _, arg := range args {
				parts := strings.SplitN(arg, "=", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])
					// Truncate long values
					displayValue := value
					if len(value) > 40 {
						displayValue = value[:37] + "..."
					}
					rows = append(rows, "<tr><td>"+key+"</td><td title=\""+value+"\">"+displayValue+"</td></tr>")
				}
			}

			if len(rows) == 0 {
				return "*"
			}

			table := `<span class="citation-marker">*<div class="citation-table"><table>` +
				`<caption>Citation: ` + citeType + `</caption>` +
				strings.Join(rows, "") +
				`</table></div></span>`

			return table
		}
	}

	// Register handlers for all citation types
	RegisterTemplateHandler("cite web", citationHandler("web"))
	RegisterTemplateHandler("cite book", citationHandler("book"))
	RegisterTemplateHandler("cite news", citationHandler("news"))
	RegisterTemplateHandler("cite journal", citationHandler("journal"))

	// Nowrap template handler
	RegisterTemplateHandler("nowrap", func(args []string) string {
		if len(args) > 0 {
			return `<span style="white-space:nowrap">` + args[0] + `</span>`
		}
		return ""
	})

	// Plainlist template handler
	RegisterTemplateHandler("plainlist", func(args []string) string {
		if len(args) == 0 {
			return `<ul class="plainlist"></ul>`
		}

		// Split content into lines and wrap each line in <li> tags
		lines := strings.Split(args[0], "\n")
		var items []string
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "* ") {
				line = strings.TrimPrefix(line, "* ")
				items = append(items, "<li>"+line+"</li>")
			}
		}

		return `<ul class="plainlist">` + strings.Join(items, "\n") + `</ul>`
	})

}

// Matches [[link]] or [[link|text]]
var linkPattern = regexp.MustCompile(`\[\[([^\[\]]+?)(?:\|([^\[\]]+?))?\]\]`)

// ConvertWikiTextToHTML converts wikitext content to HTML
func ConvertWikiTextToHTML(content string) string {
	// First process all headers
	content = headerPattern.ReplaceAllStringFunc(content, func(match string) string {
		parts := headerPattern.FindStringSubmatch(match)
		if len(parts) < 3 {
			return match
		}
		level := len(parts[1]) // Number of = signs
		text := parts[2]
		return "<h" + string(rune('0'+level)) + ">" + text + "</h" + string(rune('0'+level)) + ">"
	})

	// Then process all links
	content = linkPattern.ReplaceAllStringFunc(content, func(match string) string {
		parts := linkPattern.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}

		linkTarget := strings.TrimSpace(parts[1])
		// Convert spaces to underscores in the link target
		linkTarget = strings.ReplaceAll(linkTarget, " ", "_")
		// URL encode the link target
		linkTarget = strings.ReplaceAll(linkTarget, "/wiki/", "")

		linkText := linkTarget
		if len(parts) > 2 && parts[2] != "" {
			linkText = strings.TrimSpace(parts[2])
		}

		return `<a href="/wiki/` + linkTarget + `">` + linkText + `</a>`
	})

	// Then process all template matches
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
