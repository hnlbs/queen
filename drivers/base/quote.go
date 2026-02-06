// Package base provides common functionality for database drivers.
package base

import "strings"

// QuoteChar represents a SQL identifier quote character.
type QuoteChar string

const (
	DoubleQuote QuoteChar = `"`
	Backtick    QuoteChar = "`"
	Bracket     QuoteChar = `[]`
)

// QuoteIdentifier escapes and wraps a SQL identifier using the provided quote character.
func QuoteIdentifier(name string, quoteChar QuoteChar) string {
	if quoteChar == Bracket {
		escaped := strings.ReplaceAll(name, "]", "]]")
		return "[" + escaped + "]"
	}

	quote := string(quoteChar)
	escaped := strings.ReplaceAll(name, quote, quote+quote)
	return quote + escaped + quote
}

// QuoteDoubleQuotes is a convenience wrapper for QuoteIdentifier with DoubleQuote.
func QuoteDoubleQuotes(name string) string {
	return QuoteIdentifier(name, DoubleQuote)
}

// QuoteBackticks is a convenience wrapper for QuoteIdentifier with Backtick.
func QuoteBackticks(name string) string {
	return QuoteIdentifier(name, Backtick)
}

// QuoteBrackets is a convenience wrapper for QuoteIdentifier with Bracket.
func QuoteBrackets(name string) string {
	return QuoteIdentifier(name, Bracket)
}
