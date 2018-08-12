package mml

import (
	"bytes"
	"errors"
	"strconv"
	"strings"

	"github.com/aryszka/mml/parser"
)

var (
	errUnexpectedParserResult = errors.New("unexpected parser result")
	errMissingStatement       = errors.New("missing statement")
	errMultipleStatements     = errors.New("multiple statements")
)

func parseInt(ast *parser.Node) int {
	t := ast.Text()

	var base int
	switch {
	case strings.HasPrefix(t, "0x"):
		base = 16
		t = t[2:]
	case strings.HasPrefix(t, "0"):
		base = 8
		t = t[1:]
	default:
		base = 10
	}

	i, err := strconv.ParseInt(t, base, 64)
	if err != nil {
		panic(err)
	}

	return int(i)
}

func parseFloat(ast *parser.Node) float64 {
	v, err := strconv.ParseFloat(ast.Text(), 64)
	if err != nil {
		panic(err)
	}

	return v
}

func unescape(s string) string {
	var (
		r   []rune
		esc bool
	)

	for _, c := range s {
		if esc {
			switch c {
			case 'b':
				c = '\b'
			case 'f':
				c = '\f'
			case 'n':
				c = '\n'
			case 'r':
				c = '\r'
			case 't':
				c = '\t'
			case 'v':
				c = '\v'
			}

			r = append(r, c)
			esc = false
			continue
		}

		if c == '\\' {
			esc = true
			continue
		}

		r = append(r, c)
	}

	return string(r)
}

func parseString(ast *parser.Node) string {
	t := ast.Text()
	return unescape(t[1 : len(t)-1])
}

func parseSymbol(ast *parser.Node) symbol {
	return symbol{name: ast.Text()}
}

func parseSpread(ast *parser.Node) spread {
	v := parse(ast.Nodes[0])
	return spread{value: v}
}

func parseExpressionList(n []*parser.Node) []interface{} {
	e := make([]interface{}, len(n))
	for i, ni := range n {
		e[i] = parse(ni)
	}

	return e
}

func parseList(ast *parser.Node) list {
	return list{values: parseExpressionList(ast.Nodes)}
}

func parseMutableList(ast *parser.Node) list {
	l := parseList(ast)
	l.mutable = true
	return l
}

func parseExpressionKey(ast *parser.Node) expressionKey {
	v := parse(ast.Nodes[0])
	return expressionKey{value: v}
}

func parseEntry(ast *parser.Node) entry {
	key := parse(ast.Nodes[0])
	value := parse(ast.Nodes[1])
	return entry{key: key, value: value}
}

func parseStruct(ast *parser.Node) structure {
	s := structure{entries: make([]interface{}, len(ast.Nodes))}
	for i, n := range ast.Nodes {
		e := parse(n)
		s.entries[i] = e
	}

	return s
}

func parseMutableStruct(ast *parser.Node) structure {
	s := parseStruct(ast)
	s.mutable = true
	return s
}

func parseChannel(ast *parser.Node) chan interface{} {
	if len(ast.Nodes) == 0 {
		return make(chan interface{})
	}

	return make(chan interface{}, parse(ast.Nodes[0]).(int))
}

func parseReturn(ast *parser.Node) ret {
	return ret{value: parse(ast.Nodes[0])}
}

func parseStatementList(ast *parser.Node) statementList {
	s := make([]interface{}, len(ast.Nodes))
	for i := range ast.Nodes {
		s[i] = parse(ast.Nodes[i])
	}

	return statementList{statements: s}
}

func parseFunction(ast *parser.Node) (f function) {
	last := len(ast.Nodes) - 1
	params := ast.Nodes[:last]
	value := ast.Nodes[last]

	if len(params) > 0 {
		lastArg := len(params) - 1
		if params[lastArg].Name == "collect-argument" {
			f.collectParam = params[lastArg].Nodes[0].Text()
			params = params[:lastArg]
		}
	}

	f.params = make([]string, len(params))
	for i := range params {
		f.params[i] = params[i].Text()
	}

	f.statement = parse(value)
	return
}

func parseEffect(ast *parser.Node) function {
	f := parseFunction(ast)
	f.effect = true
	return f
}

func parseRange(ast *parser.Node) rangeExpression {
	v := parse(ast.Nodes[0])
	switch ast.Name {
	case "range-from":
		return rangeExpression{from: v}
	default:
		return rangeExpression{to: v}
	}
}

func parseExpressionIndexer(ast *parser.Node) indexer {
	e := parse(ast.Nodes[0])

	if len(ast.Nodes) == 1 {
		return indexer{expression: e, index: rangeExpression{}}
	}

	i := parse(ast.Nodes[1])

	switch it := i.(type) {
	case rangeExpression:
		if len(ast.Nodes) > 2 {
			it.to = parse(ast.Nodes[2]).(rangeExpression).to
		}

		return indexer{expression: e, index: it}
	default:
		return indexer{expression: e, index: i}
	}
}

func parseSymbolIndexer(ast *parser.Node) indexer {
	e := parse(ast.Nodes[0])
	k := ast.Nodes[1].Text()
	return indexer{expression: e, index: k}
}

func parseFunctionApplication(ast *parser.Node) functionApplication {
	f := parse(ast.Nodes[0])
	a := parseExpressionList(ast.Nodes[1:])
	return functionApplication{function: f, args: a}
}

func parseUnaryExpression(ast *parser.Node) unary {
	var op unaryOperator
	switch ast.Nodes[0].Name {
	case "binary-not":
		op = binaryNot
	case "plus":
		op = plus
	case "minus":
		op = minus
	case "logical-not":
		op = logicalNot
	default:
		panic(errUnexpectedParserResult)
	}

	a := parse(ast.Nodes[1])
	return unary{op: op, arg: a}
}

func parseBinaryExpression(ast *parser.Node) binary {
	var op binaryOperator
	switch ast.Nodes[len(ast.Nodes)-2].Name {
	case "binary-and":
		op = binaryAnd
	case "xor":
		op = xor
	case "and-not":
		op = andNot
	case "lshift":
		op = lshift
	case "rshift":
		op = rshift
	case "mul":
		op = mul
	case "div":
		op = div
	case "mod":
		op = mod
	case "add":
		op = add
	case "sub":
		op = sub
	case "eq":
		op = eq
	case "not-eq":
		op = notEq
	case "less":
		op = less
	case "less-or-eq":
		op = lessOrEq
	case "greater":
		op = greater
	case "greater-or-eq":
		op = greaterOrEq
	case "logical-and":
		op = logicalAnd
	case "logical-or":
		op = logicalOr
	default:
		panic(errUnexpectedParserResult)
	}

	var left interface{}
	if len(ast.Nodes) > 3 {
		astc := *ast
		astc.Nodes = astc.Nodes[:len(astc.Nodes)-2]
		left = parse(&astc)
	} else {
		left = parse(ast.Nodes[0])
	}

	right := parse(ast.Nodes[len(ast.Nodes)-1])

	return binary{op: op, left: left, right: right}
}

func parseChaining(ast *parser.Node) interface{} {
	a, n := parse(ast.Nodes[0]), ast.Nodes[1:]
	for {
		if len(n) == 0 {
			return a
		}

		f := parse(n[0])
		a = functionApplication{
			function: f,
			args: []interface{}{a},
		}

		n = n[1:]
	}
}

func parserTernary(ast *parser.Node) cond {
	return cond{
		condition: parse(ast.Nodes[0]),
		consequent: parse(ast.Nodes[1]),
		alternative: parse(ast.Nodes[2]),
	}
}

func parseIf(ast *parser.Node) cond {
	n := ast.Nodes
	c := cond{
		condition: parse(n[0]),
		consequent: parse(n[1]),
	}
	n = n[2:]

	cc := &c
	for len(n) > 0 {
		if len(n) == 1 {
			cc.alternative = parse(n[0])
			return c
		}

		ccc := cond{
			condition: parse(n[0]),
			consequent: parse(n[1]),
		}
		cc.alternative = ccc
		n = n[2:]
		cc = &ccc
	}

	return c
}

func parse(ast *parser.Node) interface{} {
	switch ast.Name {
	case "int":
		return parseInt(ast)
	case "float":
		return parseFloat(ast)
	case "string":
		return parseString(ast)
	case "true":
		return true
	case "false":
		return false
	case "symbol":
		return parseSymbol(ast)
	case "spread-expression":
		return parseSpread(ast)
	case "list":
		return parseList(ast)
	case "mutable-list":
		return parseMutableList(ast)
	case "expression-key":
		return parseExpressionKey(ast)
	case "entry":
		return parseEntry(ast)
	case "struct":
		return parseStruct(ast)
	case "mutable-struct":
		return parseMutableStruct(ast)
	case "channel":
		return parseChannel(ast)
	case "return":
		return parseReturn(ast)
	case "block":
		return parseStatementList(ast)
	case "function":
		return parseFunction(ast)
	case "effect":
		return parseEffect(ast)
	case "range-from", "range-to":
		return parseRange(ast)
	case "expression-indexer":
		return parseExpressionIndexer(ast)
	case "symbol-indexer":
		return parseSymbolIndexer(ast)
	case "function-application":
		return parseFunctionApplication(ast)
	case "unary-expression":
		return parseUnaryExpression(ast)
	case "binary0", "binary1", "binary2", "binary3", "binary4":
		return parseBinaryExpression(ast)
	case "chaining":
		return parseChaining(ast)
	case "ternary-expression":
		return parserTernary(ast)
	case "if":
		return parseIf(ast)
	default:
		panic(errUnexpectedParserResult)
	}
}

func parseModule(doc string) (m module, err error) {
	var ast *parser.Node
	ast, err = parser.Parse(bytes.NewBufferString(doc))
	if err != nil {
		return
	}

	for _, sn := range ast.Nodes {
		var s interface{}
		s = parse(sn)
		m.statements = append(m.statements, s)
	}

	return m, nil
}

func parseStatement(doc string) (interface{}, error) {
	m, err := parseModule(doc)
	if err != nil {
		return nil, err
	}

	if len(m.statements) == 0 {
		return nil, errMissingStatement
	}

	if len(m.statements) > 1 {
		return nil, errMultipleStatements
	}

	return m.statements[0], nil
}
