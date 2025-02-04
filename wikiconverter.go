package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// TemplateHandler processes a template and returns HTML
type TemplateHandler func([]string) string

var (
	templateHandlers = make(map[string]TemplateHandler)

	// Matches header patterns like === Header === with symmetrical equals signs
	headerPattern = regexp.MustCompile(`(?m)^(={1,6})\s*(.+?)\s*={1,6}$`)

	// Matches [[link]] or [[link|text]]
	linkPattern = regexp.MustCompile(`\[\[([^\[\]]+?)(?:\|([^\[\]]+?))?\]\]`)

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
			return ""
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
			return ""
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
			return ""
		}

		var links []string
		for _, arg := range args {
			links = append(links, `<a href="`+arg+`">`+arg+`</a>`)
		}

		return `<div class="note">Main article: ` + strings.Join(links, ", ") + `</div>`
	})

	RegisterTemplateHandler("further", func(args []string) string {
		if len(args) == 0 {
			return ""
		}

		var links []string
		for _, arg := range args {
			links = append(links, `<a href="`+arg+`">`+arg+`</a>`)
		}

		return `<div class="note">Futher information: ` + strings.Join(links, ", ") + `</div>`
	})

	// Generic infobox handler function
	infoboxHandler := func(caption string) TemplateHandler {
		return func(args []string) string {
			if len(args) == 0 {
				return ""
			}

			// Build table rows from key=value pairs
			var rows []string
			for _, arg := range args {
				parts := strings.SplitN(arg, "=", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])
					rows = append(rows, "<tr><th>"+key+"</th><td>"+value+"</td></tr>")
				}
			}

			if len(rows) == 0 {
				return ""
			}

			return `<table class="infobox">` +
				`<caption>` + caption + `</caption>` +
				strings.Join(rows, "") +
				`</table>`
		}
	}

	// Register handlers for different infobox types
	RegisterTemplateHandler("infobox television", infoboxHandler("Television Show Information"))
	RegisterTemplateHandler("infobox person", infoboxHandler("Personal Information"))
	RegisterTemplateHandler("infobox award", infoboxHandler("Award Information"))
	RegisterTemplateHandler("infobox organization", infoboxHandler("Organization Information"))

	// For template handler
	RegisterTemplateHandler("for", func(args []string) string {
		if len(args) < 2 {
			return ""
		}
		return `<div class="note">For ` + args[0] + `, see <a href="` + args[1] + `">` + args[1] + `</a></div>`
	})

	// Templates to completely ignore/skip
	skip := func(args []string) string { return "" }
	RegisterTemplateHandler("redirect", skip)
	RegisterTemplateHandler("good page", skip)
	RegisterTemplateHandler("pp-blp", skip)
	RegisterTemplateHandler("pp-move", skip)
	RegisterTemplateHandler("pp-move-indef", skip)
	RegisterTemplateHandler("use mdy dates", skip)
	RegisterTemplateHandler("use dmy dates", skip)
	RegisterTemplateHandler("use american english", skip)
	RegisterTemplateHandler("multiple issues", skip)
	RegisterTemplateHandler("cleanup rewrite", skip)
	RegisterTemplateHandler("citation needed", skip)
	RegisterTemplateHandler("more footnotes", skip)
	RegisterTemplateHandler("reflist", skip)
	RegisterTemplateHandler("update", skip)

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
	RegisterTemplateHandler("citation", citationHandler("general"))
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

	// Language template handler
	RegisterTemplateHandler("lang", func(args []string) string {
		if len(args) < 2 {
			return ""
		}
		lang := args[0]
		text := args[1]
		return `<span title="` + lang + ` language text"><em>` + text + `</em></span>`
	})

	// Non-breaking space template handler
	RegisterTemplateHandler("nbsp", func(args []string) string {
		return "&nbsp;"
	})

	// Start date template handler
	RegisterTemplateHandler("start date", func(args []string) string {
		if len(args) < 4 {
			return ""
		}
		year := args[0]
		month := fmt.Sprintf("%02d", atoi(args[1]))
		day := fmt.Sprintf("%02d", atoi(args[2]))
		return fmt.Sprintf("%s-%s-%s", year, month, day)
	})

	// Marriage template handler
	RegisterTemplateHandler("marriage", func(args []string) string {
		if len(args) < 3 {
			return ""
		}
		name := args[0]
		startYear := args[1]
		endDate := args[2]
		return fmt.Sprintf("%s (m. %s - %s)", name, startYear, endDate)
	})

	// Date and age template handler for both birth dates and start dates
	dateAndAgeHandler := func(suffix string) TemplateHandler {
		return func(args []string) string {
			if len(args) < 3 {
				return ""
			}
			year := atoi(args[0])
			month := atoi(args[1])
			day := atoi(args[2])

			date := fmt.Sprintf("%d-%02d-%02d", year, month, day)

			// Calculate years since date
			now := time.Now()
			years := now.Year() - year
			// Adjust if date hasn't occurred this year
			if now.Month() < time.Month(month) || (now.Month() == time.Month(month) && now.Day() < day) {
				years--
			}

			if suffix == "age" {
				return fmt.Sprintf("%s (age %d)", date, years)
			}
			return fmt.Sprintf("%s (%d %s)", date, years, suffix)
		}
	}

	RegisterTemplateHandler("birth date and age", dateAndAgeHandler("age"))
	RegisterTemplateHandler("start date and age", dateAndAgeHandler("years ago"))

}

func atoi(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

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

		linkText := parts[1]
		if len(parts) > 2 && parts[2] != "" {
			linkText = strings.TrimSpace(parts[2])
		}

		return `<a href="/wiki/` + linkTarget + `">` + linkText + `</a>`
	})

	// Process templates with brace counting to handle nesting
	var templates []struct {
		fullMatch    string
		innerContent string
	}

	// Find all templates by counting braces
	var buf strings.Builder
	var braceLevel int
	inTemplate := false
	for _, r := range content {
		switch {
		case r == '{' && !inTemplate:
			buf.WriteRune(r)
			braceLevel++
			if braceLevel == 2 {
				inTemplate = true
				buf.Reset()
				braceLevel = 0
			}
		case inTemplate:
			switch r {
			case '{':
				braceLevel++
			case '}':
				if braceLevel == 0 {
					// Found closing brace
					inTemplate = false
					templates = append(templates, struct {
						fullMatch    string
						innerContent string
					}{
						fullMatch:    "{{" + buf.String() + "}}",
						innerContent: buf.String(),
					})
					buf.Reset()
					continue
				}
				braceLevel--
			}
			buf.WriteRune(r)
		}
	}

	// Process templates from innermost first (reverse order)
	for i := len(templates) - 1; i >= 0; i-- {
		match := templates[i]
		fullMatch := match.fullMatch
		innerContent := match.innerContent

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
			handler = func(_ []string) string {
				return `<div style="color:#AAA">No template for "` + templateName + `": ` + fullMatch + `</div>`
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
