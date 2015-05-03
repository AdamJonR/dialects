Dialects
========

A recursive-descent parser for Domain Specific Languages (DSLs) that is implemented using Go and facilitates parsing through use of Parsing Expression Grammars (PEGs).

Motivation
----------

DSLs allow you to uniquely emphasize the relevant information required to solve a problem. Syntax is no longer a vestige of the chosen programming language, but rather a carefully selected set of choices that best communicate the potential solutions for a specific problem domain.

The Dialects parser provides a simple library to facilitate the development of DSLs using the Go programming language.

How Does It Work?
-------------------------

You use the dialects library by creating a package that implements the Dialectable interface, which consists of the following methods:

    NewDialect() *Dialect
    NewModel() interface{}
    GenerateOutput(model interface{}) (string, error)

The flow of parsing works through the following steps:

1. Create the Dialect struct pointer and model using the NewDialect() and NewModel() methods, respectively.
2. Parse the source using the grammar defined in the *Dialect struct returned by NewDialect().
3. Store the parts and their corresponding constituents identified by the grammar in a tree structure.
4. When a part that has been found has a handler set, the handler is called, passing in the part tree structure and the model (passed in as an empty interface{}).
5. The handler builds up the model with the information contained in the part.
6. Once the file has been completely parsed, the GenerateOutput() method is called, and this returns the generated output after parsing or an error.

Example
-------

For an example implementation of a DSL using the Dialects library, you can view the [qform DSL I've authored creating html5 forms](https://github.com/AdamJonR/qform).
