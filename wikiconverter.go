package main

import (
	"fmt"
	"regexp"
	"strings"
)

// TemplateHandler processes a template and returns HTML
type TemplateHandler func([]string) string

var templateHandlers = map[string]TemplateHandler{
    "infobox": parseInfobox,
    "usd": handleUSDTemplate,
    "other uses": handleOtherUsesTemplate, 
    "short description": handleShortDescriptionTemplate,
    "main": handleMainTemplate,
    "see also": handleSeeAlsoTemplate,
    "lang": handleLangTemplate,
    "langx": handleLangTemplate, // Handle both lang and langx the same way
}

// ConvertWikiTextToHTML converts wikitext content to HTML
func ConvertWikiTextToHTML(content string) string {
	var outputLines []string
	var currentTable []string
	inTable := false

	lines := strings.Split(content, "\n")
	for i := 0; i < len(lines); i++ {
		line := lines[i]

		if strings.HasPrefix(line, "{|") {
			inTable = true
			currentTable = []string{line}
			continue
		}

		if inTable {
			if strings.HasPrefix(line, "|}") {
				currentTable = append(currentTable, line)
				outputLines = append(outputLines, parseWikitable(currentTable))
				inTable = false
				currentTable = nil
				continue
			}
			currentTable = append(currentTable, line)
			continue
		}

		if strings.HasPrefix(line, "{{") {
			var templateLines []string
			templateLines = append(templateLines, line)
			templateOpenCount := strings.Count(line, "{{") - strings.Count(line, "}}")

			for templateOpenCount > 0 && i+1 < len(lines) {
				i++
				line = lines[i]
				templateLines = append(templateLines, line)
				templateOpenCount += strings.Count(line, "{{") - strings.Count(line, "}}")
			}

			templateName := getTemplateName(templateLines[0])
			if handler, exists := templateHandlers[templateName]; exists {
				outputLines = append(outputLines, handler(templateLines))
			} else {
				// Handle unknown templates
				for _, tline := range templateLines {
					parsedLine := removeTemplates(tline)
					parsedLine = parseHeader(parsedLine)
					parsedLine = parseLinks(parsedLine)
					parsedLine = wrapInParagraph(parsedLine)
					outputLines = append(outputLines, parsedLine)
				}
			}
		} else {
			if strings.HasPrefix(strings.TrimSpace(line), "*") || strings.HasPrefix(strings.TrimSpace(line), "#") {
				listContent, processedLines := parseList(lines[i:])
				outputLines = append(outputLines, listContent)
				i += processedLines - 1
				continue
			}
			
			parsedLine := removeTemplates(line)
			parsedLine = parseHeader(parsedLine)
			parsedLine = parseLinks(parsedLine)
			parsedLine = parseStyle(parsedLine)
			parsedLine = wrapInParagraph(parsedLine)
			outputLines = append(outputLines, parsedLine)
		}
	}

	return strings.Join(outputLines, "\n")
}

// parseInfobox converts infobox template to HTML table
func parseInfobox(lines []string) string {
	if len(lines) == 0 || !strings.HasPrefix(strings.ToLower(lines[0]), "{{infobox") {
		return strings.Join(lines, "\n")
	}

	var tableRows []string
	tableRows = append(tableRows, "<table class=\"infobox\">")

	// Extract title from first line
	title := strings.TrimPrefix(strings.ToLower(lines[0]), "{{infobox")
	title = strings.TrimSpace(title)
	if title != "" {
		tableRows = append(tableRows, fmt.Sprintf("<tr><th colspan=\"2\">%s</th></tr>", title))
	}

	// Process remaining lines
	for _, line := range lines[1 : len(lines)-1] { // Skip first and last lines
		line = strings.TrimSpace(line)
		if line == "" || line == "}}" {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			key = strings.TrimPrefix(key, "|")
			// Parse templates and style in both key and value
			key = removeTemplates(key)
			key = parseHeader(key)
			key = parseLinks(key)
			value = removeTemplates(value)
			value = parseHeader(value)
			value = parseLinks(value)
			tableRows = append(tableRows, fmt.Sprintf("<tr><th>%s</th><td>%s</td></tr>", key, value))
		}
	}

	tableRows = append(tableRows, "</table>")
	return strings.Join(tableRows, "\n")
}

// removeTemplates removes unwanted template tags
func removeTemplates(line string) string {
    // Remove <ref></ref> tags and their contents, including empty ones
    refTags := regexp.MustCompile(`<ref[^>]*>(.*?)</ref>|<ref[^>]*></ref>`)
    line = refTags.ReplaceAllString(line, "")

    // Also remove single <ref /> tags
    singleRefTags := regexp.MustCompile(`<ref[^>]*/>`)
    line = singleRefTags.ReplaceAllString(line, "")

    // Remove cite templates
    citeTemplates := regexp.MustCompile(`(?i)\{\{cite[^}]*\}\}`)
    line = citeTemplates.ReplaceAllString(line, "")

    // Remove specific templates that should be ignored
    ignoreTemplates := regexp.MustCompile(`\{\{(pp-move|--\)|!|Pp-semi-indef|Good article|Sfn[^}]*|Sfnm[^}]*|refn.*?)\}\}`)
    return ignoreTemplates.ReplaceAllString(line, "")
}

// parseLinks converts wikitext links to HTML links
func parseLinks(line string) string {
	// Regex to match both simple and named links
	linkRegex := regexp.MustCompile(`\[\[([^|\]]+)(?:\|([^\]]+))?\]\]`)
	return linkRegex.ReplaceAllStringFunc(line, func(match string) string {
		parts := linkRegex.FindStringSubmatch(match)
		if len(parts) == 3 && parts[2] != "" {
			// Named link [[target|name]]
			return fmt.Sprintf(`<a href="%s">%s</a>`, parts[1], parts[2])
		}
		// Simple link [[target]]
		target := parts[1]
		return fmt.Sprintf(`<a href="%s">%s</a>`, target, target)
	})
}

// isTemplateBlock checks if a line is part of a template block
func isTemplateBlock(line string) bool {
	// Check if line starts and ends with {{ }}
	if strings.HasPrefix(line, "{{") && strings.HasSuffix(line, "}}") {
		return true
	}
	// Check if line contains an unclosed {{ or }}
	openCount := strings.Count(line, "{{")
	closeCount := strings.Count(line, "}}")
	return openCount != closeCount
}

// wrapInParagraph wraps text in paragraph tags if needed
func wrapInParagraph(line string) string {
	if line == "" || isTemplateBlock(line) || strings.HasPrefix(line, "<h") {
		return line
	}
	return "<p>" + line + "</p>"
}

// parseWikitable converts wikitable syntax to HTML table
func parseWikitable(lines []string) string {
	if len(lines) == 0 || !strings.HasPrefix(lines[0], "{|") {
		return strings.Join(lines, "\n")
	}

	var tableRows []string
	var currentRow []string
	var caption string

	// Extract class from first line
	classMatch := regexp.MustCompile(`class="([^"]*)"`)
	classes := classMatch.FindStringSubmatch(lines[0])
	tableClass := ""
	if len(classes) > 1 {
		tableClass = fmt.Sprintf(` class="%s"`, classes[1])
	}

	tableRows = append(tableRows, fmt.Sprintf("<table%s>", tableClass))

	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)

		switch {
		case strings.HasPrefix(line, "|+"):
			// Caption
			caption = strings.TrimPrefix(line, "|+")
			caption = strings.TrimSpace(caption)
			caption = removeTemplates(caption)
			caption = parseHeader(caption)
			caption = parseLinks(caption)
			tableRows = append(tableRows, fmt.Sprintf("<caption>%s</caption>", caption))

		case strings.HasPrefix(line, "|-"):
			// Row delimiter - output current row if exists
			if len(currentRow) > 0 {
				tableRows = append(tableRows, "<tr>"+strings.Join(currentRow, "")+"</tr>")
				currentRow = nil
			}

		case strings.HasPrefix(line, "!"):
			// Header cell
			content := strings.TrimPrefix(line, "!")
			content = strings.TrimSpace(content)
			content = removeTemplates(content)
			content = parseHeader(content)
			content = parseLinks(content)
			content = parseStyle(content)
			currentRow = append(currentRow, fmt.Sprintf("<th>%s</th>", content))

		case strings.HasPrefix(line, "|"):
			if strings.HasPrefix(line, "|}") {
				// Table end
				if len(currentRow) > 0 {
					tableRows = append(tableRows, "<tr>"+strings.Join(currentRow, "")+"</tr>")
				}
				break
			}
			// Regular cell
			content := strings.TrimPrefix(line, "|")
			content = strings.TrimSpace(content)
			content = removeTemplates(content)
			content = parseHeader(content)
			content = parseLinks(content)
			currentRow = append(currentRow, fmt.Sprintf("<td>%s</td>", content))
		}
	}

	tableRows = append(tableRows, "</table>")
	return strings.Join(tableRows, "\n")
}

// parseList converts wikitext bullet/numbered lists to HTML lists
func parseList(lines []string) (string, int) {
	if len(lines) == 0 {
		return "", 0
	}

	line := lines[0]
	if !strings.HasPrefix(strings.TrimSpace(line), "*") && !strings.HasPrefix(strings.TrimSpace(line), "#") {
		return strings.Join(lines, "\n"), 0
	}

	var listItems []string
	var processedLines int
	currentLevel := 0
	isBullet := strings.HasPrefix(strings.TrimSpace(line), "*")
	tag := "ul"
	if !isBullet {
		tag = "ol"
	}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "*") && !strings.HasPrefix(trimmed, "#") {
			break
		}

		// Count the number of * or # at start
		level := 0
		for _, char := range trimmed {
			if (isBullet && char == '*') || (!isBullet && char == '#') {
				level++
			} else {
				break
			}
		}

		if level != currentLevel {
			if level > currentLevel {
				// Start new nested list
				listItems = append(listItems, fmt.Sprintf("<%s>", tag))
			} else {
				// Close nested lists
				for j := 0; j < (currentLevel - level); j++ {
					listItems = append(listItems, fmt.Sprintf("</%s>", tag))
				}
			}
			currentLevel = level
		}

		// Process the list item content
		content := strings.TrimSpace(strings.TrimLeft(trimmed, "*#"))
		content = removeTemplates(content)
		content = parseHeader(content)
		content = parseLinks(content)
		listItems = append(listItems, fmt.Sprintf("<li>%s</li>", content))
		processedLines = i + 1
	}

	// Close any remaining nested lists
	for i := 0; i < currentLevel; i++ {
		listItems = append(listItems, fmt.Sprintf("</%s>", tag))
	}

	return strings.Join(listItems, "\n"), processedLines
}

// parseStyle converts wikitext style markers to HTML tags
func parseStyle(line string) string {
	// Handle bold and italics together first
	boldItalicRegex := regexp.MustCompile(`'''''([^']+)'''''`)
	line = boldItalicRegex.ReplaceAllString(line, "<b><i>$1</i></b>")

	// Handle bold
	boldRegex := regexp.MustCompile(`'''([^']+)'''`)
	line = boldRegex.ReplaceAllString(line, "<b>$1</b>")

	// Handle italics
	italicRegex := regexp.MustCompile(`''([^']+)''`)
	line = italicRegex.ReplaceAllString(line, "<i>$1</i>")

	return line
}

func handleUSDTemplate(lines []string) string {
    if len(lines) == 0 {
        return ""
    }
    // Extract amount from {{USD|amount|...}}
    usdTemplate := regexp.MustCompile(`\{\{USD\|(\d+)(?:\|[^}]*)?}}`)
    matches := usdTemplate.FindStringSubmatch(lines[0])
    if len(matches) > 1 {
        return "$" + matches[1]
    }
    return lines[0]
}

func handleOtherUsesTemplate(lines []string) string {
    if len(lines) == 0 {
        return ""
    }
    otherUsesRegex := regexp.MustCompile(`\{\{Other uses\|([^|}]+)(?:\|([^}]+))?}}`)
    matches := otherUsesRegex.FindStringSubmatch(lines[0])
    if len(matches) < 2 {
        return lines[0]
    }
    
    mainLink := fmt.Sprintf(`<a href="%s">%s</a>`, matches[1], matches[1])
    otherLinks := ""
    if len(matches) > 2 && matches[2] != "" {
        linkParts := strings.Split(matches[2], "|")
        for _, link := range linkParts {
            otherLinks += fmt.Sprintf(`, <a href="%s">%s</a>`, link, link)
        }
    }
    return fmt.Sprintf(`<div class="note">For other uses, see %s%s</div>`, mainLink, otherLinks)
}

func handleShortDescriptionTemplate(lines []string) string {
    if len(lines) == 0 {
        return ""
    }
    shortDescRegex := regexp.MustCompile(`\{\{Short description\|([^}]+)}}`)
    matches := shortDescRegex.FindStringSubmatch(lines[0])
    if len(matches) > 1 {
        return fmt.Sprintf(`<em class="short-description">%s</em>`, matches[1])
    }
    return lines[0]
}

func handleMainTemplate(lines []string) string {
    if len(lines) == 0 {
        return ""
    }
    mainRegex := regexp.MustCompile(`\{\{Main\|([^}]+)}}`)
    matches := mainRegex.FindStringSubmatch(lines[0])
    if len(matches) > 1 {
        return fmt.Sprintf(`<div class="note">Main article: <a href="%s">%s</a></div>`, matches[1], matches[1])
    }
    return lines[0]
}

func handleSeeAlsoTemplate(lines []string) string {
    if len(lines) == 0 {
        return ""
    }
    seeAlsoRegex := regexp.MustCompile(`\{\{See also\|([^}]+)}}`)
    matches := seeAlsoRegex.FindStringSubmatch(lines[0])
    if len(matches) > 1 {
        return fmt.Sprintf(`<div class="note">See also: <a href="%s">%s</a></div>`, matches[1], matches[1])
    }
    return lines[0]
}

func handleLangTemplate(lines []string) string {
    if len(lines) == 0 {
        return ""
    }
    langRegex := regexp.MustCompile(`\{\{Lang[x]?\|([^|]+)\|(?:link=no\|)?([^}]+)}}`)
    matches := langRegex.FindStringSubmatch(lines[0])
    if len(matches) > 2 {
        return fmt.Sprintf(`%s: <em>%s</em>`, matches[1], matches[2])
    }
    return lines[0]
}

func getTemplateName(line string) string {
    if !strings.HasPrefix(line, "{{") {
        return ""
    }
    
    // Extract template name between {{ and first | or }}
    content := strings.TrimPrefix(line, "{{")
    end := strings.IndexAny(content, "|}")
    if end == -1 {
        return ""
    }
    
    return strings.ToLower(strings.TrimSpace(content[:end]))
}

// parseHeader converts wikitext headers to HTML headers
func parseHeader(line string) string {
	if len(line) < 3 {
		return line
	}

	// Check if line starts and ends with equal signs
	if line[0] == '=' && line[len(line)-1] == '=' {
		// Count the number of = signs at start
		level := 0
		for i := 0; i < len(line) && line[i] == '='; i++ {
			level++
		}

		// Limit to h6 maximum
		if level > 6 {
			level = 6
		}

		// Trim the = signs and whitespace
		content := strings.Trim(line, "= ")
		return fmt.Sprintf("<h%d>%s</h%d>", level, content, level)
	}

	return line
}
