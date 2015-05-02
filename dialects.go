// Package dialects provides a library for parsing Domain-Specific Languages (DSL's)
package dialects

import (
	"errors"
	"regexp"
)

// PartDefinition provides the struct that's used to define the various parts of a grammar
type PartDefinition struct {
	Description   string
	Ignore        bool
	Constituents  [][]string
	Handler       func(*Part, interface{}) (ok bool)
	Regex         string
	ValidateMatch func([]string) bool
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

// Parser provides a simple container for the primary parsing variables
type Parser struct {
	status            string
	currentPosPointer *int
	input             string
	output            string
	dialect           *Dialect
	model             interface{}
	compiledRegexes   map[string]*regexp.Regexp
}

// Parse provides the entry point for using the dialect library
func Parse(dialectable Dialectable, input string) (string, error) {
	parser := Parser{model: dialectable.NewModel(), dialect: dialectable.NewDialect(), compiledRegexes: make(map[string]*regexp.Regexp)}
	currentPos := 0
	parser.currentPosPointer = &currentPos
	parser.input = input
	// WHEEEEEEEE!!!! (enjoy the ride as you descend into the rabbit hole)
	parts := findOne(parser.dialect.RootName, parser, nil)
	if len(parts) < 1 {
		return "", errors.New("dialects error: Parse() function of dialect unable to find root part (" + parser.dialect.RootName + ") of " + parser.dialect.Title)
	}
	return dialectable.GenerateOutput(parser.model)
}

// findOne returns an array of Parts, returning empty array if none found
func findOne(partName string, parser Parser, path []string) (parts []*Part) {
	partDefinition := parser.dialect.PartDefinitions[partName]
	part := &Part{
		Name:   partName,
		Ignore: partDefinition.Ignore,
	}
	// save current position and pointer reference
	currentPosPointer := parser.currentPosPointer
	tempPos := *currentPosPointer
	// set part start to current position
	part.StartPos = tempPos
	// handle Consituents
	if len(partDefinition.Constituents) > 0 {
		// find Constituents
		part.Constituents = findConstituents(partDefinition.Constituents, parser, path)
		// handle no Constituents
		if len(part.Constituents) < 1 {
			// ensure current pos reset to what it was at the start of the call
			(*currentPosPointer) = tempPos
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
		// return nil if invalid matche(s)
		if partDefinition.ValidateMatch != nil && !partDefinition.ValidateMatch(matches) {
			return nil
		}
		// optionally format match
		if partDefinition.FormatMatch != nil {
			part.Value = partDefinition.FormatMatch(matches)
		} else {
			part.Value = matches[0]
		}
		// update current position to account for length of entire match
		(*currentPosPointer) = (*currentPosPointer) + len(matches[0])
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
	for _, Constituentseq := range Constituents {
		// test each possible set of Constituents
		parts := findConstituentseq(Constituentseq, parser, path)
		// if parts found, return result
		if len(parts) > 0 {
			return parts
		}
		// otherwise, try next sequence
	}
	// no constituent set found, so return empty slice
	return nil
}

func findConstituentseq(Constituentseq []string, parser Parser, path []string) (parts []*Part) {
	var Constituents []*Part
	for _, constituentID := range Constituentseq {
		lastChar := constituentID[len(constituentID)-1:]
		// find modifiers
		switch lastChar {
		case "+":
			parts = findMany(constituentID[:len(constituentID)-1], parser, path)
			// if one or more required part not found, we're done
			if len(parts) < 1 {
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
				return parts
			}
		}
		// add parts that aren't Ignored
		if len(parts) > 0 && !parts[0].Ignore {
			Constituents = append(Constituents, parts...)
		}
	}
	return Constituents
}
