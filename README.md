# Dialects

Dialects is a recursive-descent parser for Domain Specific Languages (DSLs) that is implemented using Go and facilitates parsing through use of Parsing Expression Grammars (PEGs).

## Motivation

DSLs allow you to uniquely emphasize the relevant information required to solve a problem. Syntax is no longer a vestige of the chosen programming language, but rather a carefully selected set of choices that best communicate the potential solutions for a specific problem domain.

The Dialects parser provides a simple library to facilitate the development of DSLs using the Go programming language.

## How Does It Work?

You use the dialects library by creating a package with a struct that implements the Dialectable interface and then passing a pointer to the Dialectable value to the Parse function.

### Dialectable Interface

```
NewDialect() *Dialect
NewModel() interface{}
GenerateOutput(model interface{}) (string, error)
```

The Dialectable interface essentially serves as a container for callbacks needed during the parsing process.

### Dialect Struct

```
type Dialect struct {
	Title           string
	Description     string
	Examples        map[string]string
	RootName        string
	PartDefinitions map[string]PartDefinition
	Model           interface{}
	Version         float64
}
```

The Dialect struct contains information about this particular DSL Dialect, including the title, description, examples, version, and an empty interface for the model that the DSL builds up during parsing. The root name and part definitions require further explanation.

### Part Definitions

The map of part definitions defines the grammar for a particular DSL Dialect. Because the definitions are contained in a map, which doesn't have a specific order, the root name identifies the part that serves as the starting point/state for parsing.

The part definition struct has the following form listed below.

```
type PartDefinition struct {
	Description   string
	Ignore        bool
	Constituents  [][]string
	Handler       func(*Part, interface{}) (ok bool)
	Regex         string
	ValidateMatch func([]string) bool
	FormatMatch   func([]string) string
}
```

### Parse() Function

```
Parse(dialectable Dialectable, input string) (string, error, string)
```

The flow of Parse() function works through the following steps:

1. Create the Dialect struct pointer and model using the NewDialect() and NewModel() methods, respectively.
2. Parse the source using the grammar defined in the *Dialect struct returned by NewDialect().
3. Store the parts and their corresponding constituents identified by the grammar in a tree structure.
4. When a part that has been found has a handler set, the handler is called, passing in the part tree structure and the model (passed in as an empty interface{}).
5. The handler builds up the model with the information contained in the part.
6. Once the file has been completely parsed, the GenerateOutput() method is called, and this returns the generated output after parsing or an error.

## Example

For an example implementation of a DSL using the Dialects library, you can view the [qform DSL I've authored creating html5 forms](https://github.com/AdamJonR/qform).
