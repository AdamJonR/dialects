// Package dialects provides a library for parsing Domain-Specific Languages (DSL's)
package dialects

import (
	"bytes"
	"errors"
	"regexp"
	"strconv"
	"strings"
)

// PartDefinition provides the struct that's used to define the various parts of a grammar
type PartDefinition struct {
	Description   string
	Ignore        bool
	Constituents  [][]string
	Handler       func(*Part, interface{}) (ok bool)
	Regex         string
	ValidateMatch func([]string) (bool, string)
	FormatMatch   func([]string) string
}

// Dialect defines the DSL Title, Description, Examples, grammar, and Model
type Dialect struct {
	Title           string
	Description     string
	Examples        map[string]string
	RootName        string
	PartDefinitions map[string]PartDefinition
	Model           interface{}
	Version         float64
}

// Dialectable defines the interface all DSL grammars must fulfill
type Dialectable interface {
	NewDialect() *Dialect
	NewModel() interface{}
	GenerateOutput(model interface{}) (string, error)
}

// Part provides a convenient storage container for the corresponding properties of parsed parts of an input string
type Part struct {
	Name         string
	StartPos     int
	EndPos       int
	Path         []string
	Ignore       bool
	Parent       *Part
	Value        string
	Constituents []*Part
}

type Log struct {
	buffer      *bytes.Buffer
	indent      string
	indentLevel int
	currentLine int
}

// Parser provides a simple container for the primary parsing variables
type Parser struct {
	status            string
	currentPosPointer *int
	input             string
	output            string
	dialect           *Dialect
	model             interface{}
	compiledRegexes   map[string]*regexp.Regexp
	log               *Log
}

// Parse provides the entry point for using the dialect library
func Parse(dialectable Dialectable, input string) (string, error, string) {
	parser := Parser{model: dialectable.NewModel(), dialect: dialectable.NewDialect(), compiledRegexes: make(map[string]*regexp.Regexp)}
	currentPos := 0
	parser.currentPosPointer = &currentPos
	parser.input = input
	parser.log = &Log{buffer: new(bytes.Buffer), indent: "| | | | ", indentLevel: 0, currentLine: 1}
	// WHEEEEEEEE!!!! (enjoy the ride as you descend into the rabbit hole)
	parts := findOne(parser.dialect.RootName, parser, nil)
	if len(parts) < 1 {
		return "", errors.New("dialects error: Parse() function of dialect unable to find root part (" + parser.dialect.RootName + ") of " + parser.dialect.Title), ""
	}
	output, err := dialectable.GenerateOutput(parser.model)
	return output, err, parser.log.buffer.String() + "\n"
}

// findOne returns an array of Parts, returning empty array if none found
func findOne(partName string, parser Parser, path []string) (parts []*Part) {
	// exit early if position pointer is already at the end of the string
	if len(parser.input) == (*parser.currentPosPointer + 1) {
		return nil
	}
	partDefinition := parser.dialect.PartDefinitions[partName]
	part := &Part{
		Name:   partName,
		Ignore: partDefinition.Ignore,
	}
	// save current position and pointer reference
	currentPosPointer := parser.currentPosPointer
	// set part start to current position
	part.StartPos = *currentPosPointer
	// handle Consituents
	if len(partDefinition.Constituents) > 0 {
		// find Constituents
		part.Constituents = findConstituents(partDefinition.Constituents, parser, path)
		// handle no Constituents
		if len(part.Constituents) < 1 {
			// return early with nil
			return nil
		}
		// otherwise call Handler if present
		if partDefinition.Handler != nil {
			if ok := partDefinition.Handler(part, parser.model); !ok {
				// if something went wrong, call handler
				return nil
			}
		}
		// set end position of part to current position
		part.EndPos = *currentPosPointer
		// return part slice
		return []*Part{part}
	}
	// otherwise handle regex
	if partDefinition.Regex != "" {
		compiledRegex, saved := parser.compiledRegexes[partName]
		// compile Regex if not already done
		if !saved {
			compiledRegex = regexp.MustCompile(partDefinition.Regex)
			// save for future use
			parser.compiledRegexes[partName] = compiledRegex
		}
		// find part by Regex
		matches := compiledRegex.FindStringSubmatch(parser.input[(*currentPosPointer):])
		// return nil if no matches
		if len(matches) < 1 {
			return nil
		}
		// check for validator
		if partDefinition.ValidateMatch != nil {
			// call validator if present
			isValid, errMsg := partDefinition.ValidateMatch(matches)
			// check if invalid
			if !isValid {
				// log error
				if errMsg != "" {
					// log custom error message
					parser.log.buffer.WriteString(parser.log.indent[:parser.log.indentLevel] + "invalid " + partName + " starting on line " + strconv.Itoa(parser.log.currentLine) + ": " + errMsg + "\n")
				} else {
					// log generic err message
					parser.log.buffer.WriteString(parser.log.indent[:parser.log.indentLevel] + "invalid " + partName + " starting on line " + strconv.Itoa(parser.log.currentLine) + "\n")
				}
				// return nil
				return nil
			}
		}
		// optionally format match
		if partDefinition.FormatMatch != nil {
			part.Value = partDefinition.FormatMatch(matches)
		} else {
			part.Value = matches[0]
		}
		// update current position to account for length of entire match
		(*currentPosPointer) = (*currentPosPointer) + len(matches[0])
		// update currentLine to account for \n's in the match
		parser.log.currentLine = parser.log.currentLine + strings.Count(matches[0], "\n")
		// update EndPos
		part.EndPos = (*currentPosPointer)
		// return part
		return []*Part{part}
	}
	// handle invalid case where definition has neither parts nor Regex
	return nil
}

func findMany(partName string, parser Parser, path []string) (manyParts []*Part) {
	findMore := true

	for findMore {
		parts := findOne(partName, parser, path)
		if len(parts) > 0 {
			manyParts = append(manyParts, parts...)
			continue
		}

		findMore = false
	}

	return manyParts
}

func findConstituents(Constituents [][]string, parser Parser, path []string) (parts []*Part) {
	// store temporary position in case sequence isn't found
	tempPos := *parser.currentPosPointer
	// store tempory current line
	tempCurrentLine := parser.log.currentLine
	// cycle through constituent sequences
	for _, Constituentseq := range Constituents {
		// test each possible set of Constituents
		parts := findConstituentseq(Constituentseq, parser, path)
		// if parts found, return result
		if len(parts) > 0 {
			return parts
		}
		// otherwise, reset position and try next sequence
		*parser.currentPosPointer = tempPos
		parser.log.currentLine = tempCurrentLine
	}
	// no constituent set found, so return empty slice
	return nil
}

func findConstituentseq(Constituentseq []string, parser Parser, path []string) (parts []*Part) {
	// ensure log indent is large enough
	if parser.log.indentLevel > len(parser.log.indent) {
		parser.log.indent = parser.log.indent + parser.log.indent
	}
	// log sequence parsing
	parser.log.buffer.WriteString(parser.log.indent[:parser.log.indentLevel] + strings.Join(Constituentseq, ", ") + "\n")
	// update indentLevel
	parser.log.indentLevel = parser.log.indentLevel + 2
	var Constituents []*Part
	for _, constituentID := range Constituentseq {
		lastChar := constituentID[len(constituentID)-1:]
		// find modifiers
		switch lastChar {
		case "+":
			parts = findMany(constituentID[:len(constituentID)-1], parser, path)
			// if one or more required part not found, we're done
			if len(parts) < 1 {
				// adjust indent back to current level
				parser.log.indentLevel = parser.log.indentLevel - 2
				// log missing part of sequence
				parser.log.buffer.WriteString(parser.log.indent[:parser.log.indentLevel] + "missing " + constituentID[:len(constituentID)] + " on line " + strconv.Itoa(parser.log.currentLine) + "\n")
				// return empty slice pointer
				return parts
			}
		case "*":
			parts = findMany(constituentID[:len(constituentID)-1], parser, path)
		case "?":
			parts = findOne(constituentID[:len(constituentID)-1], parser, path)
		default:
			parts = findOne(constituentID, parser, path)
			// if required part not found, we're done
			if len(parts) < 1 {
				// adjust indent back to current level
				parser.log.indentLevel = parser.log.indentLevel - 2
				// log missing part of sequence
				parser.log.buffer.WriteString(parser.log.indent[:parser.log.indentLevel] + "missing " + constituentID[:len(constituentID)] + " on line " + strconv.Itoa(parser.log.currentLine) + "\n")
				// return empty slice pointer
				return parts
			}
		}
		// add parts that aren't Ignored
		if len(parts) > 0 && !parts[0].Ignore {
			Constituents = append(Constituents, parts...)
		}
	}
	// adjust indent back to current level
	parser.log.indentLevel = parser.log.indentLevel - 2
	// write to log buffer
	parser.log.buffer.WriteString(parser.log.indent[:parser.log.indentLevel] + "found\n")
	// return slice pointer
	return Constituents
}
