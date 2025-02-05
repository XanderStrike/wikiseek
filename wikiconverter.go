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

type handlerRegistration struct {
	pattern *regexp.Regexp
	handler TemplateHandler
}

// processLists converts wikitext lists to HTML lists
func processLists(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	var listStack []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "*") || strings.HasPrefix(trimmed, "#") {
			// Determine list type and depth
			listType := "ul"
			if strings.HasPrefix(trimmed, "#") {
				listType = "ol"
			}
			depth := len(trimmed) - len(strings.TrimLeft(trimmed, "*#"))

			// Close lists if needed
			for len(listStack) > depth {
				result = append(result, "</"+listStack[len(listStack)-1]+">")
				listStack = listStack[:len(listStack)-1]
			}

			// Open new lists if needed
			for len(listStack) < depth {
				result = append(result, "<"+listType+">")
				listStack = append(listStack, listType)
			}

			// Add list item
			itemContent := strings.TrimSpace(trimmed[depth:])
			result = append(result, "<li>"+itemContent+"</li>")
		} else {
			// Close all open lists
			for len(listStack) > 0 {
				result = append(result, "</"+listStack[len(listStack)-1]+">")
				listStack = listStack[:len(listStack)-1]
			}
			result = append(result, line)
		}
	}

	// Close any remaining open lists
	for len(listStack) > 0 {
		result = append(result, "</"+listStack[len(listStack)-1]+">")
		listStack = listStack[:len(listStack)-1]
	}

	return strings.Join(result, "\n")
}

var (
	templateHandlers []handlerRegistration

	// Matches header patterns like === Header === with symmetrical equals signs
	headerPattern = regexp.MustCompile(`(?m)^(={1,6})\s*(.+?)\s*={1,6}$`)

	// Matches [[link]] or [[link|text]]
	linkPattern = regexp.MustCompile(`\[\[([^\[\]]+?)(?:\|([^\[\]]+?))?\]\]`)

	// Matches template name and arguments
	templateArgsPattern = regexp.MustCompile(`(?s)^([^{}|]+)(?:\|(.*))?$`)
)

// RegisterTemplateHandler registers a handler for a template name pattern (supports regex)
func RegisterTemplateHandler(pattern string, handler TemplateHandler) {
	// Compile pattern to regex, adding start/end anchors and case insensitivity
	regexPattern := regexp.MustCompile(`(?i)^` + pattern + `$`)
	templateHandlers = append(templateHandlers, handlerRegistration{
		pattern: regexPattern,
		handler: handler,
	})
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

	// Generic infobox handler for any infobox type
	RegisterTemplateHandler(`infobox\b.*`, func(args []string) string {
		// Extract infobox type from template name
		caption := "Information"
		if len(args) > 0 {
			// Get the infobox type from the template name
			typeParts := strings.SplitN(args[0], " ", 2)
			if len(typeParts) > 1 {
				caption = strings.Title(typeParts[1]) + " Information"
			}
		}

		if len(args) < 1 {
			return ""
		}

		// Build table rows from key=value pairs (skip the first arg which is the infobox type)
		var rows []string
		for _, arg := range args[1:] {
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
	})

	// For template handler
	RegisterTemplateHandler("for", func(args []string) string {
		if len(args) < 2 {
			return ""
		}
		return `<div class="note">For ` + args[0] + `, see <a href="` + args[1] + `">` + args[1] + `</a></div>`
	})

	// Templates to completely ignore/skip
	skip := func(args []string) string { return "" }
	RegisterTemplateHandler("redirect.*", skip)
	RegisterTemplateHandler("good page", skip)
	RegisterTemplateHandler(`pp\b.*`, skip)
	RegisterTemplateHandler("use mdy dates", skip)
	RegisterTemplateHandler("use dmy dates", skip)
	RegisterTemplateHandler("use american english", skip)
	RegisterTemplateHandler("multiple issues", skip)
	RegisterTemplateHandler("cleanup rewrite", skip)
	RegisterTemplateHandler("citation needed", skip)
	RegisterTemplateHandler("more footnotes", skip)
	RegisterTemplateHandler("reflist", skip)
	RegisterTemplateHandler("update", skip)
	RegisterTemplateHandler("!", skip)

	// Generic citation handler for any citation type
	citationHandler := func(args []string) string {
		if len(args) == 0 {
			return "*"
		}

		// Extract citation type from template name
		citationType := "general"
		if len(args) > 0 {
			typeParts := strings.SplitN(args[0], " ", 2)
			if len(typeParts) > 1 {
				citationType = strings.Title(typeParts[1])
			}
		}

		// Build table rows from key=value pairs
		var rows []string
		for _, arg := range args[1:] { // Skip first arg which is template name
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

		return `<span class="citation-marker">*<div class="citation-table"><table>` +
			`<caption>Citation: ` + citationType + `</caption>` +
			strings.Join(rows, "") +
			`</table></div></span>`
	}

	// Register handler for all citation types using regex pattern
	RegisterTemplateHandler(`cite\b.*`, citationHandler)
	RegisterTemplateHandler(`citation`, citationHandler)

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
	RegisterTemplateHandler(`lang(?:x)?`, func(args []string) string {
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

	// Convert template handler - simplified to just show original value and unit
	RegisterTemplateHandler("convert", func(args []string) string {
		if len(args) < 2 {
			return ""
		}
		value := args[0]
		unit := args[1]
		return value + unit
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

	// Then process bold and italic text
	content = regexp.MustCompile(`'''''(.*?)'''''`).ReplaceAllString(content, "<strong><em>$1</em></strong>")
	content = regexp.MustCompile(`'''(.*?)'''`).ReplaceAllString(content, "<strong>$1</strong>")
	content = regexp.MustCompile(`''(.*?)''`).ReplaceAllString(content, "<em>$1</em>")

	// Process bulleted and numbered lists
	content = processLists(content)

	// Process all links
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
		// Find first matching pattern
		var handler TemplateHandler
		for _, registration := range templateHandlers {
			if registration.pattern.MatchString(templateName) {
				handler = registration.handler
				break
			}
		}

		if handler == nil {
			// Default handler for unknown templates
			handler = func(_ []string) string {
				return `<span style="color:#AAA">No template for "` + templateName + `": ` + fullMatch + `</span>`
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
			if braceLevel > 0 {
				braceLevel--
			}
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
