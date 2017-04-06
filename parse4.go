/*
[generator]
- created by the syntax definition
- the syntax definition accepts the node type and, in case of complex parsers, a list of other node types
- creates a parser instance for a concrete node
- before creating, it can tell whether it can create a parser with the provided init node and excluded node
  types
- create accepts an optional init node and a list of excluded nodes
- can tell whether a node type is a member of the type of the generator
- can create and create return an error when a referenced generator is not defined by the syntax

[parser]
- tries parsing a concrete node
- returns whether it accepts more tokens or it is done
- when done it can have a valid or invalid result
- when done it returns the unparsed tokens
- when done with a valid result, returns the parsed node
- on init, it accepts an init node
- on init, it accepts a set of excluded node types
- if it has a valid result for the given token position in the cache, returns that
- if it knows that it cannot be valid for the given token position, returns invalid
- when the parse was successful for the given token position, caches the result
- when the parse was unsuccessful for the given token position, caches the result, but only if there was no
  successful parse before, because this can happen when parsing further
- parse returns an error when it detects that the syntax is invalid
- calling parse after inidicating done, causes a panic
- returns how many tokens were taken from the cache in addition to the provided ones minus the unparsed
- in complex parsers, the can create check of the items must happen before checking the cache
- every complex parser sets the token of the returned node to the token of the first item node
- token position for the cache considers empty sequences

[init item]
- a node that is already parsed at the given token position and can be used as the initial segment of a more
  complex node

[excluded node types]
- nodes that are being tried at the given token position on a higher level in the parser tree

[parse result]
- tells whether the parser is accepting more tokens
- if not, tells whether the node is valid, and contains the parsed node and the unparsed tokens
- if an item was taken from the cache, it tells how many tokens more were taken from the cache than accepted

[primitive generator]
- cannot create a parser when its node type is excluded
- cannot create a parser when it is supplied with an init node, and the init node is of a different type
- creates a primitive parser with its name, expected token type and init node

[primitive parser]
- on init, expects the node type, and either the token type or an init node
- when it has an init node, it is automatically valid, and doesn't accept more tokens
- when it doesn't have an init node, it accepts a single token of the provided token type
- possible valid results: the node of the token
- it always returns its own node type
- no need to cache it

[optional generator]
- returns an error when the optional generator is not defined in the syntax
- returns false if it is excluded
- returns the result of the optional generator otherwise
- extends the excluded with itself
- cannot contain itself

[optional parser]
- on init, expects the node type, the generator of the optional node, an optional init node and the list of
  excluded nodes
- on init, it adds itself to the excluded list
- always returns a valid result
- when the parse of the optional node failed, returns a valid result with a zero node and all the tokens passed
  in
- when the parse of the optional node succeeded, returns the result of the optional parser
- it never returns its own node type
- if the result is empty, the first unparsed token is used as the node token
- possible valid results: the optional node or a zero node

[sequence generator]
- returns an error when the item generator is not defined by the syntax
- can create returns false if the sequence is excluded
- extends the excluded with itself
- returns true if the init item is a member type and is not excluded
- returns the result of the item generator
- cannot contain itself

[sequence parser]
- on init, expects the node type, an item generator, an optional init node and the list of excluded nodes
- the init node is considered an item
- always returns a valid result
- when the parse of an item failed, returns the existing items
- when the parse of an item succeeded, stores it, queues the unparsed tokens, and tries to parse the next item
- the init item is only used with the first item
- when an item from the cache has more read ahead than tokens in the queue, it ignores the right amount of
  tokens before continuing with the next item
- when there is an init item, it's token is used to check whether there is a cached result
- in case of the first item, it uses the excluded types and init node to initialize the item, for the rest it
  uses only itself as the excluded type and the zero node
- the unparsed tokens are stored in a queue, returned as unparsed when done
- possible valid results: an empty node, or a node with item nodes
- it always returns its own node type
- parses only the first node with the init item
- if there is an init item, and the parse of the first node fails, and the init item is a member of the item,
  then it is added as the first node
- if the result is empty, the first unparsed token is used as the node token
- it returns also if zero
- TODO: what if the init item can be an element? try to do the same as in the group

[group generator]
- returns an error if any of the items is not defined by the syntax
- returns false if it is excluded
- returns an error if it doesn't have items
- extends the excluded with itself
- returns true if the first item returns true
- returns true if it has an init item and it can be the first item
- can contain itself

[group parser]
- on init, it expects the node type, the generators of the group items, an optional init node and a list of the
  excluded node types
- the init node is considered the first item or the init node of the first item
- it always uses the next generator for the next item. When there are no more generators, the parse is
  successful
- the unparsed tokens are stored in a queue, returned as unparsed when done
- when creating the parser of the first item, it passes in the init item and the excluded types. For the rest of
  the items, no init item and no excluded types are passed in.
- if the parse of the first item failed, it checks if the init item can be used as the first item, and if yes,
  continues with next item, otherwise it fails
- if the parse of an item fails, it fails
- on failure, it returns the tokens of the parsed items, the unparsed tokens and the tokens in the queue
- if the parse of an item succeeds, it appends the node to its nodes, and continues with the next item
- possible valid results: the group node with the non-zero items
- it always returns its own node type

[union generator]
- it expands the unions in the union for the actual items
- returns an error if any of the items is not defined in the syntax
- returns an error if it doesn't have items
- can contain itself, but it's ignored
- returns true if any of the generators return true
- returns the generators that return true

[union parser]
- on init, it expects the node type, an optional init node, the element generators and a list of the excluded
  node types
- the init node is considered an element or an init node to the elements
- possible valid results: the node of the matching element, can be zero from optional
- when the element parsing hasn't started, it tries to find a generator that accepts the current init item and
  the set of excluded types and parses the item with that
- when the element parser failed, tries the next generator
- when an element parser succeeded tries all the generators again, for a result that consumes more tokens than
  the last successful element
- it never returns its own node type
- TODO: should be able to use the init node as an element

[errors]
- TODO: errors coming from invalid syntax specification
- TODO: errors coming from invalid syntax

[tracing]
*/
package mml

import (
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
)

type node struct {
	token
	typ   string
	nodes []node
}

type generator interface {
	canCreate(init node, excludedTypes []string) (bool, error)
	create(t trace, init node, excludedTypes []string) (parser, error)
	member(nodeType string) (bool, error)
}

type parseResult struct {
	accepting bool
	valid     bool
	unparsed  []token
	fromCache int
	node      node
}

type parser interface {
	parse(t token) (parseResult, error)
}

type cacheItem struct {
	match   map[string]node
	noMatch map[string]bool
}

type tokenCache map[token]cacheItem

type traceLevel int

const (
	traceOff traceLevel = iota
	traceOn
	traceDebug
)

type trace struct {
	level      traceLevel
	path       []string
	pathString string
}

type baseParser struct {
	trace         trace
	nodeType      string
	init          node
	excludedTypes []string
	done          bool
}

type collectionParser struct {
	baseParser
	node           node
	started        bool
	queue          []token
	skip           int
	parser         parser
	firstGenerator generator
}

type primitiveGenerator struct {
	nodeType  string
	tokenType tokenType
}

type primitiveParser struct {
	baseParser
	token tokenType
}

type optionalGenerator struct {
	nodeType string
	optional string
}

type optionalParser struct {
	baseParser
	optional       generator
	optionalParser parser
}

type sequenceGenerator struct {
	nodeType string
	itemType string
}

type sequenceParser struct {
	collectionParser
	generator generator
}

type groupGenerator struct {
	nodeType  string
	itemTypes []string
}

type groupParser struct {
	collectionParser
	items     []generator
	firstItem generator
}

var (
	generators              = make(map[string]generator)
	isSep                   func(node) bool
	postParse               = make(map[string]func(node) node)
	cache                   = make(tokenCache)
	eofToken                = token{}
	zeroNode                = node{}
	errInvalidRootGenerator = errors.New("invalid root generator")
	errUnexpectedEOF        = errors.New("unexpected EOF")
)

func stringsContain(strc []string, stri ...string) bool {
	for _, sc := range strc {
		for _, si := range stri {
			if si == sc {
				return true
			}
		}
	}

	return false
}

func (c tokenCache) getMatch(t token, name string) (node, bool) {
	ci, ok := c[t].match[name]
	return ci, ok
}

func (c tokenCache) hasNoMatch(t token, name string) bool {
	return c[t].noMatch[name]
}

func (c tokenCache) setMatch(t token, name string, n node) {
	tci := c[t]
	if tci.match == nil {
		tci.match = make(map[string]node)
	}

	tci.match[name] = n
	c[t] = tci
}

func (c tokenCache) setNoMatch(t token, name string) {
	// reasonable to leak this check over to here:
	// a shorter variant may already have been parsed
	if _, ok := c.getMatch(t, name); ok {
		return
	}

	tci := c[t]
	if tci.noMatch == nil {
		tci.noMatch = make(map[string]bool)
	}

	tci.noMatch[name] = true
	c[t] = tci
}

func newTrace(l traceLevel) trace {
	return trace{level: l}
}

func (t trace) extend(nodeType string) trace {
	p := append(t.path, nodeType)
	return trace{
		level:      t.level,
		path:       p,
		pathString: strings.Join(p, "/"),
	}
}

func (t trace) outLevel(l traceLevel, a ...interface{}) {
	if l > t.level {
		return
	}

	if t.pathString == "" {
		log.Println(a...)
		return
	}

	log.Println(append([]interface{}{t.pathString}, a...)...)
}

func (t trace) out(a ...interface{}) {
	t.outLevel(traceOn, a...)
}

func (t trace) debug(a ...interface{}) {
	t.outLevel(traceDebug, a...)
}

func (n node) zero() bool {
	return n.typ == ""
}

func (n node) tokens() []token {
	if n.zero() {
		return nil
	}

	if len(n.nodes) == 0 {
		return []token{n.token}
	}

	var t []token
	for _, ni := range n.nodes {
		t = append(t, ni.tokens()...)
	}

	return t
}

func (n node) length() int {
	return len(n.tokens())
}

func register(nodeType string, g generator) generator {
	generators[nodeType] = g
	return g
}

func unexpectedToken(nodeType string, t token) error {
	return fmt.Errorf("unexpected token: %v, %v", nodeType, t)
}

func unspecifiedParser(nodeType string) error {
	return fmt.Errorf("unspecified parser: %s", nodeType)
}

func optionalContainingSelf(nodeType string) error {
	return fmt.Errorf("optional containing self: %s", nodeType)
}

func sequenceContainingSelf(nodeType string) error {
	return fmt.Errorf("sequence containing self: %s", nodeType)
}

func unexpectedResult(nodeType string) error {
	return fmt.Errorf("unexpected parse result: %s", nodeType)
}

func groupWithoutItems(nodeType string) error {
	return fmt.Errorf("group without items: %s", nodeType)
}

func primitive(nodeType string, token tokenType) generator {
	return register(nodeType, &primitiveGenerator{
		nodeType:  nodeType,
		tokenType: token,
	})
}

func (p *baseParser) checkDone(currentToken token) {
	if p.done {
		panic(unexpectedToken(p.nodeType, currentToken))
	}
}

func (p *baseParser) cacheToken(t token) token {
	if p.init.zero() {
		return t
	}

	return p.init.token
}

func (p *collectionParser) checkSkip() (parseResult, bool) {
	if p.skip == 0 {
		return parseResult{}, false
	}

	p.skip--
	return parseResult{accepting: true}, true
}

func (p *collectionParser) appendNode(n node) {
	p.node.nodes = append(p.node.nodes, n)
	if len(p.node.nodes) == 1 {
		p.node.token = n.token
	}
}

func (p *collectionParser) appendInitIfMember() (bool, error) {
	if p.init.zero() {
		return false, nil
	}

	if m, err := p.firstGenerator.member(p.init.typ); !m || err != nil {
		return m, err
	}

	p.appendNode(p.init)
	return true, nil
}

func (p *collectionParser) appendParsedItem(n node, fromCache int) {
	p.appendNode(n)
	if fromCache < len(p.queue) {
		p.queue = p.queue[fromCache:]
	} else {
		p.queue, p.skip = nil, fromCache-len(p.queue)
	}
}

func (p *collectionParser) parseNextToken(parser parser) (parseResult, error) {
	if len(p.queue) > 0 {
		var t token
		t, p.queue = p.queue[0], p.queue[1:]
		return parser.parse(t)
	}

	return parseResult{accepting: true}, nil
}

func (p *collectionParser) error(currentToken token, err error) (parseResult, error) {
	return parseResult{unparsed: append([]token{currentToken}, p.queue...)}, err
}

func (g *primitiveGenerator) canCreate(init node, excludedTypes []string) (bool, error) {
	if stringsContain(excludedTypes, g.nodeType) {
		return false, nil
	}

	if !init.zero() && init.typ != g.nodeType {
		return false, nil
	}

	return true, nil
}

func (g *primitiveGenerator) create(t trace, init node, excludedTypes []string) (parser, error) {
	return newPrimitiveParser(t.extend(g.nodeType), g.nodeType, g.tokenType, init), nil
}

func (g *primitiveGenerator) member(nodeType string) (bool, error) {
	return nodeType == g.nodeType, nil
}

func newPrimitiveParser(t trace, nodeType string, token tokenType, init node) *primitiveParser {
	return &primitiveParser{
		baseParser: baseParser{
			trace:    t,
			nodeType: nodeType,
			init:     init,
		},
		token: token,
	}
}

func (p *primitiveParser) parse(t token) (parseResult, error) {
	p.trace.out("parsing", t)

	p.checkDone(t)
	p.done = true

	if !p.init.zero() {
		p.trace.out("valid from init")
		return parseResult{
			valid: true,
			node:  p.init,
		}, nil
	}

	if t.typ != p.token {
		p.trace.out("invalid token")
		return parseResult{
			unparsed: []token{t},
		}, nil
	}

	p.trace.out("valid token")
	n := node{typ: p.nodeType, token: t}
	return parseResult{
		valid: true,
		node:  n,
	}, nil
}

func optional(nodeType, optionalType string) generator {
	return register(nodeType, &optionalGenerator{
		nodeType: nodeType,
		optional: optionalType,
	})
}

func (g *optionalGenerator) canCreate(init node, excludedTypes []string) (bool, error) {
	optional, ok := generators[g.optional]
	if !ok {
		return false, unspecifiedParser(g.optional)
	}

	if m, err := optional.member(g.nodeType); err != nil {
		return false, err
	} else if m {
		return false, optionalContainingSelf(g.nodeType)
	}

	if stringsContain(excludedTypes, g.nodeType) {
		return false, nil
	}

	return optional.canCreate(init, append(excludedTypes, g.nodeType))
}

func (g *optionalGenerator) create(t trace, init node, excludedTypes []string) (parser, error) {
	optional, ok := generators[g.optional]
	if !ok {
		return nil, unspecifiedParser(g.optional)
	}

	return newOptionalParser(
		t.extend(g.nodeType),
		g.nodeType,
		optional,
		init,
		append(excludedTypes, g.nodeType),
	), nil
}

func (g *optionalGenerator) member(nodeType string) (bool, error) {
	optional, ok := generators[g.optional]
	if !ok {
		return false, unspecifiedParser(g.optional)
	}

	if m, err := optional.member(nodeType); m || err != nil {
		return m, err
	}

	return nodeType == g.nodeType, nil
}

func newOptionalParser(t trace, nodeType string, optional generator, init node, excludedTypes []string) parser {
	return &optionalParser{
		baseParser: baseParser{
			trace:         t,
			nodeType:      nodeType,
			init:          init,
			excludedTypes: append(excludedTypes, nodeType),
		},
		optional: optional,
	}
}

func (p *optionalParser) parse(t token) (parseResult, error) {
	p.trace.out("parsing", t)
	p.checkDone(t)

	if p.optionalParser == nil {
		if ok, err := p.optional.canCreate(p.init, p.excludedTypes); !ok || err != nil {
			p.trace.out("cannot create optional")
			p.done = true
			return parseResult{unparsed: []token{t}}, err
		}

		optional, err := p.optional.create(p.trace, p.init, p.excludedTypes)
		if err != nil {
			p.trace.out("failed to create optional")
			p.done = true
			return parseResult{unparsed: []token{t}}, err
		}

		p.optionalParser = optional
	}

	ct := p.cacheToken(t)
	if cache.hasNoMatch(ct, p.nodeType) {
		p.trace.out("cached mismatch")
		p.done = true
		return parseResult{unparsed: []token{t}}, nil
	}

	if cn, ok := cache.getMatch(ct, p.nodeType); ok {
		p.trace.out("cached match")
		p.done = true
		return parseResult{
			valid:     true,
			node:      cn,
			unparsed:  []token{t},
			fromCache: cn.length(),
		}, nil
	}

	r, err := p.optionalParser.parse(t)
	if err != nil {
		p.trace.out("failed to parse optional")
		p.done = true
		return parseResult{unparsed: []token{t}}, err
	}

	if r.accepting {
		return r, nil
	}

	p.trace.out("optional done, parsed:", r.valid)
	p.done = true

	ct = r.node.token
	if r.node.zero() {
		if len(r.unparsed) == 0 {
			panic(unexpectedResult(p.nodeType))
		}

		ct = r.unparsed[0]
	}

	cache.setMatch(ct, p.nodeType, r.node)
	r.valid = true
	return r, nil
}

func sequence(nodeType, itemType string) generator {
	return register(nodeType, &sequenceGenerator{
		nodeType: nodeType,
		itemType: itemType,
	})
}

func (g *sequenceGenerator) canCreate(init node, excludedTypes []string) (bool, error) {
	item, ok := generators[g.itemType]
	if !ok {
		return false, unspecifiedParser(g.itemType)
	}

	if m, err := item.member(g.nodeType); err != nil {
		return false, err
	} else if m {
		return false, sequenceContainingSelf(g.nodeType)
	}

	if stringsContain(excludedTypes, g.nodeType) {
		return false, nil
	}

	excludedTypes = append(excludedTypes, g.nodeType)

	if !init.zero() {
		if m, err := item.member(init.typ); err != nil {
			return false, err
		} else if m && !stringsContain(excludedTypes, init.typ) {
			return true, nil
		}
	}

	return item.canCreate(init, excludedTypes)
}

func (g *sequenceGenerator) create(t trace, init node, excludedTypes []string) (parser, error) {
	item, ok := generators[g.itemType]
	if !ok {
		return nil, unspecifiedParser(g.itemType)
	}

	return newSequenceParser(
		t.extend(g.nodeType),
		g.nodeType,
		item,
		init,
		append(excludedTypes, g.nodeType),
	), nil
}

func (g *sequenceGenerator) member(nodeType string) (bool, error) {
	return nodeType == g.nodeType, nil
}

func newSequenceParser(t trace, nodeType string, item generator, init node, excludedTypes []string) parser {
	return &sequenceParser{
		collectionParser: collectionParser{
			baseParser: baseParser{
				trace:         t,
				nodeType:      nodeType,
				init:          init,
				excludedTypes: excludedTypes,
			},
			node:           node{typ: nodeType},
			firstGenerator: item,
		},
		generator: item,
	}
}

func (p *sequenceParser) nextParser() (parser, bool, error) {
	var (
		init     node
		excluded []string
	)

	if p.started {
		excluded = []string{p.nodeType}
	} else {
		init = p.init
		excluded = p.excludedTypes
	}

	if ok, err := p.generator.canCreate(init, excluded); !ok || err != nil {
		return nil, ok, err
	}

	parser, err := p.generator.create(p.trace, init, excluded)
	return parser, err == nil, err
}

func (p *sequenceParser) parse(t token) (parseResult, error) {
	p.trace.out("parsing", t)

	p.checkDone(t)
	if r, ok := p.checkSkip(); ok {
		return r, nil
	}

	if p.parser == nil {
		parser, ok, err := p.nextParser()
		if !ok || err != nil {
			p.trace.out("failed to create next item parser")
			p.done = true
			return p.error(t, err)
		}

		p.parser = parser
	}

	if !p.started {
		// should not get here when parsing from the queue
		ct := p.cacheToken(t)
		if n, ok := cache.getMatch(ct, p.nodeType); ok {
			p.trace.out("cached match")
			p.done = true
			return parseResult{
				valid:     true,
				node:      n,
				unparsed:  []token{t},
				fromCache: n.length(),
			}, nil
		}
	}

	r, err := p.parser.parse(t)
	if err != nil {
		p.trace.out("failed to parse item")
		p.done = true
		return p.error(t, err)
	}

	if r.accepting {
		return p.parseNextToken(p)
	}

	p.parser = nil
	p.queue = append(r.unparsed, p.queue...)

	if r.valid && !r.node.zero() {
		p.started = true
		p.appendParsedItem(r.node, r.fromCache)
		return p.parseNextToken(p)
	}

	if !p.started {
		p.started = true
		if ok, err := p.appendInitIfMember(); err != nil {
			p.trace.out("failed to check init item membership")
			p.done = true
			return p.error(t, err)
		} else if ok {
			return p.parseNextToken(p)
		}
	}

	p.trace.out("parse done")
	p.done = true
	if len(p.node.nodes) == 0 {
		p.node.token = p.queue[0]
	}

	cache.setMatch(p.node.token, p.nodeType, p.node)
	return parseResult{
		valid:    true,
		unparsed: p.queue,
		node:     p.node,
	}, nil
}

func group(nodeType string, itemTypes ...string) generator {
	return register(nodeType, &groupGenerator{
		nodeType:  nodeType,
		itemTypes: itemTypes,
	})
}

func (g *groupGenerator) itemGenerators() ([]generator, error) {
	ig := make([]generator, len(g.itemTypes))
	for i, it := range g.itemTypes {
		g, ok := generators[it]
		if !ok {
			return nil, unspecifiedParser(it)
		}

		ig[i] = g
	}

	return ig, nil
}

func (g *groupGenerator) canCreate(init node, excludedTypes []string) (bool, error) {
	if len(g.itemTypes) == 0 {
		return false, groupWithoutItems(g.nodeType)
	}

	if stringsContain(excludedTypes, g.nodeType) {
		return false, nil
	}

	first := generators[g.itemTypes[0]]

	if ok, err := first.canCreate(init, append(excludedTypes, g.nodeType)); ok || err != nil {
		return ok, err
	}

	if ok, err := first.member(init.typ); ok || err != nil {
		return ok, err
	}

	return false, nil
}

func (g *groupGenerator) create(t trace, init node, excludedTypes []string) (parser, error) {
	ig, err := g.itemGenerators()
	if err != nil {
		return nil, err
	}

	return newGroupParser(t.extend(g.nodeType), g.nodeType, ig, init, append(excludedTypes, g.nodeType)), nil
}

func (g *groupGenerator) member(nodeType string) (bool, error) {
	return nodeType == g.nodeType, nil
}

func newGroupParser(t trace, nodeType string, items []generator, init node, excludedTypes []string) parser {
	return &groupParser{
		collectionParser: collectionParser{
			baseParser: baseParser{
				trace:         t,
				nodeType:      nodeType,
				init:          init,
				excludedTypes: excludedTypes,
			},
			node:           node{typ: nodeType},
			firstGenerator: items[0],
		},
		items:     items,
		firstItem: items[0],
	}
}

func (p *groupParser) nextParser() (parser, error) {
	var item generator
	item, p.items = p.items[0], p.items[1:]

	var (
		init     node
		excluded []string
	)

	if !p.started {
		init = p.init
		excluded = append(p.excludedTypes, p.nodeType)
	}

	if ok, err := item.canCreate(init, excluded); !ok || err != nil {
		return nil, err
	}

	return item.create(p.trace, init, excluded)
}

func (p *groupParser) parseOrDone() (parseResult, error) {
	if len(p.items) > 0 {
		return p.parseNextToken(p)
	}

	p.trace.out("parse done")
	p.done = true
	cache.setMatch(p.node.token, p.nodeType, p.node)
	return parseResult{
		valid:    true,
		node:     p.node,
		unparsed: p.queue,
	}, nil
}

func (p *groupParser) parse(t token) (parseResult, error) {
	p.trace.out("parsing", t)
	p.checkDone(t)

	if r, ok := p.checkSkip(); ok {
		return r, nil
	}

	if p.parser == nil {
		if parser, err := p.nextParser(); err != nil {
			p.trace.out("failed to create next item parser")
			p.done = true
			return p.error(t, err)
		} else {
			p.parser = parser
		}
	}

	if !p.started {
		// should not get here when parsing from the queue

		ct := p.cacheToken(t)
		if cache.hasNoMatch(ct, p.nodeType) {
			p.trace.out("no match identified in cache")
			p.done = true
			return p.error(t, nil)
		}

		if n, ok := cache.getMatch(ct, p.nodeType); ok {
			p.trace.out("cached match")
			p.done = true
			return parseResult{
				valid:     true,
				node:      n,
				fromCache: n.length(),
				unparsed:  []token{t},
			}, nil
		}
	}

	r, err := p.parser.parse(t)
	if err != nil {
		p.trace.out("failed to parse item")
		p.done = true
		return p.error(t, err)
	}

	if r.accepting {
		return p.parseNextToken(p)
	}

	p.parser = nil
	p.queue = append(r.unparsed, p.queue...)

	if r.valid && !r.node.zero() {
		p.started = true
		p.appendParsedItem(r.node, r.fromCache)
		return p.parseOrDone()
	}

	if !p.started {
		p.started = true
		if ok, err := p.appendInitIfMember(); err != nil {
			p.trace.out("failed to check init item membership")
			p.done = true
			return p.error(t, err)
		} else if ok {
			return p.parseOrDone()
		}
	}

	if r.valid {
		p.started = true
		return p.parseOrDone()
	}

	p.trace.out("invalid item")

	var ct token
	if p.node.zero() {
		ct = p.queue[0]
	} else {
		ct = p.node.token
	}

	cache.setNoMatch(ct, p.nodeType)

	p.done = true
	return parseResult{unparsed: append(p.node.tokens(), p.queue...)}, nil
}

func union(nodeType string, nodeTypes ...string) generator { return nil }

func setPostParse(p map[string]func(node) node) {
	for pi, pp := range p {
		postParse[pi] = pp
	}
}

func parse(l traceLevel, g generator, r *tokenReader) (node, error) {
	if ok, err := g.canCreate(zeroNode, nil); err != nil {
		return zeroNode, err
	} else if !ok {
		return zeroNode, errInvalidRootGenerator
	}

	trace := newTrace(l)
	p, err := g.create(trace, zeroNode, nil)
	if err != nil {
		return zeroNode, err
	}

	last := parseResult{accepting: true}
	for {
		trace.out("checking next token")
		t, err := r.next()
		if err != nil && err != io.EOF {
			return zeroNode, err
		}

		if !last.accepting {
			trace.out("last not accepting")
			if err != io.EOF {
				return zeroNode, unexpectedToken("root", t)
			}

			return last.node, nil
		}

		if err == io.EOF {
			last, err = p.parse(eofToken)
			if err != nil {
				return zeroNode, err
			}

			if !last.valid {
				trace.out("last not valid")
				return zeroNode, errUnexpectedEOF
			}

			if len(last.unparsed) != 1 || last.unparsed[0] != eofToken {
				trace.out("invalid unparsed count")
				return zeroNode, errUnexpectedEOF
			}

			return last.node, nil
		}

		last, err = p.parse(t)
		if err != nil {
			return zeroNode, err
		}

		if !last.accepting {
			trace.out("not accepting")
			if !last.valid {
				return zeroNode, unexpectedToken("root", t)
			}

			if len(last.unparsed) > 0 {
				return zeroNode, unexpectedToken("root", last.unparsed[0])
			}
		}
	}
}