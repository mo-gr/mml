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

type parseResult struct {
	accepting bool
	valid     bool
	parsed int
	fromCache int
	unparsed  []token
	node      node
}

type parser interface {
	parse(token) parseResult
	path() []string
	name() string
	out(...interface{})
}

type generator interface {
	canCreate(node, []string) bool
	create([]string, node, []string) parser
	name() string
	out(...interface{})
	member(string) bool
}

type baseParser struct {
	p []string
	skip int
}

type baseGenerator struct {
	node string
}

type primitiveParser struct {
	baseParser
	accepting bool
	token     tokenType
	node      node
	valid bool
	parsed int
	fromCache int
}

type primitiveGenerator struct {
	baseGenerator
	token tokenType
}

type optionalParser struct {
	baseParser
	optional         generator
	optionalAccepted bool
	init             node
	excluded         []string
	parser           parser
	lastResult parseResult
}

type sequenceParser struct {
	baseParser
	node          node
	init          node
	itemGenerator generator
	currentParser parser
	queue         []token
	excluded      []string
	accepting bool
	finalResult parseResult
	parsed int
}

type groupParser struct {
	baseParser
	node          node
	init          node
	generators    []generator
	currentParser parser
	queue         []token
	excluded      []string
	accepted      []token
	itemAccepted  []token
	accepting bool
	finalResult parseResult
	parsed int
}

type unionParser struct {
	baseParser
	currentParser    parser
	generators       []generator
	activeGenerators []generator
	node             node
	valid            bool
	queue            []token
	excluded         []string
	init             node
	hasAccepted      bool
	accepting bool
	finalResult parseResult
	parsed int
}

type optionalGenerator struct {
	baseGenerator
	optional string
}

type sequenceGenerator struct {
	baseGenerator
	item string
}

type groupGenerator struct {
	baseGenerator
	items []string
}

type unionGenerator struct {
	baseGenerator
	union []string
}

type cacheItem struct {
	node node
	length int
}

type tokenCacheItem struct {
	match map[string]cacheItem
	noMatch map[string]bool
}

type tokenCache struct {
	tokens map[token]tokenCacheItem
}

var (
	isSep      func(node) bool
	postParse  = make(map[string]func(node) node)
	generators = make(map[string]generator)
	zeroNode   = node{}
	cache = tokenCache{tokens: make(map[token]tokenCacheItem)}
)

func (c tokenCache) getMatch(t token, name string) (ci cacheItem, ok bool) {
	var tci tokenCacheItem
	tci, ok = c.tokens[t]
	if !ok {
		return
	}

	ci, ok = tci.match[name]
	return
}

func (c tokenCache) hasNoMatch(t token, name string) bool {
	tci, ok := c.tokens[t]
	if !ok {
		return false
	}

	return tci.noMatch[name]
}

func (c tokenCache) setMatch(t token, name string, n node, length int) {
	tci, ok := c.tokens[t]
	if !ok {
		c.tokens[t] = tci
	}

	if tci.match == nil {
		tci.match = make(map[string]cacheItem)
	}

	tci.match[name] = cacheItem{
		node: n,
		length: length,
	}
}

func (c tokenCache) setNoMatch(t token, name string) {
	tci, ok := c.tokens[t]
	if !ok {
		c.tokens[t] = tci
	}

	if tci.noMatch == nil {
		tci.noMatch = make(map[string]bool)
	}

	tci.noMatch[name] = true
}

func stringsContain(strs []string, str string) bool {
	for _, s := range strs {
		if s == str {
			return true
		}
	}

	return false
}

func uniq(strs []string) []string {
	var strsu []string
	m := make(map[string]struct{})
	for _, s := range strs {
		if _, ok := m[s]; !ok {
			strsu = append(strsu, s)
			m[s] = struct{}{}
		}
	}

	return strsu
}

func setPostParse(p map[string]func(node) node) {
	for pi, pp := range p {
		postParse[pi] = pp
	}
}

func (n node) zero() bool { return n.typ == "" }

func (p *baseParser) path() []string { return p.p }

func (p *baseParser) name() string {
	path := p.path()
	if len(path) == 0 {
		return "empty-parser"
	}

	return path[len(path)-1]
}

func (p *baseParser) out(args ...interface{}) {
	log.Println(
		append(
			[]interface{}{strings.Join(p.path(), "/")},
			args...,
		)...,
	)
}

func (g *baseGenerator) name() string { return g.node }

func (g *baseGenerator) out(args ...interface{}) {
	log.Println(
		append(
			[]interface{}{g.name()},
			args...,
		)...,
	)
}

func newPrimitiveParser(path []string, name string, token tokenType, init node) *primitiveParser {
	p := &primitiveParser{}

	p.p = append(path, name)
	if init.zero() {
		p.accepting = true
		p.token = token
	} else {
		p.out("initialized with node")
		p.node = init
	}

	return p
}

func (p *primitiveParser) parse(t token) parseResult {
	p.out("parse", t)

	if !p.accepting {
		return parseResult{
			valid: p.valid,
			unparsed: []token{t},
			node: p.node,
			parsed: p.parsed,
			fromCache: p.fromCache,
		}
	}

	p.accepting = false

	if cache.hasNoMatch(t, p.name()) {
		return parseResult{
			unparsed: []token{t},
		}
	}

	if ci, ok := cache.getMatch(t, p.name()); ok {
		p.valid = true
		p.node = ci.node
		p.fromCache = ci.length
		return parseResult{
			valid: true,
			unparsed: []token{t},
			node: ci.node,
			fromCache: ci.length,
		}
	}

	if t.typ != p.token {
		// cache.setNoMatch(t, p.name())
		return parseResult{
			unparsed: []token{t},
		}
	}

	p.valid = true
	p.node = node{typ: p.name(), token: t}
	p.parsed = 1
	// cache.setMatch(t, p.name(), p.node, 1)
	return parseResult{
		valid: true,
		node: p.node,
		parsed: 1,
	}
}

func primitive(name string, token tokenType) {
	g := &primitiveGenerator{}
	g.node = name
	g.token = token
	generators[name] = g
}

func (g *primitiveGenerator) name() string { return g.node }

func (g *primitiveGenerator) canCreate(init node, excluded []string) bool {
	if stringsContain(excluded, g.name()) {
		return false
	}

	if !init.zero() && init.typ != g.name() {
		return false
	}

	return true
}

func (g *primitiveGenerator) create(path []string, init node, _ []string) parser {
	return newPrimitiveParser(path, g.name(), g.token, init)
}

func (g *primitiveGenerator) member(node string) bool {
	return node == g.name()
}

func newOptionalParser(path []string, name string, optional generator, init node, excluded []string) *optionalParser {
	p := &optionalParser{}
	p.optional = optional
	p.init = init
	p.excluded = append(excluded, name)
	p.p = append(path, name)
	p.lastResult = parseResult{accepting: true}
	return p
}

func (p *optionalParser) parse(t token) parseResult {
	p.out("parsing", t)

	if !p.lastResult.accepting {
		r := p.lastResult
		r.unparsed = append(r.unparsed, t)
		return r
	}

	if p.parser == nil {
		p.parser = p.optional.create(p.path(), p.init, p.excluded)
	}

	p.lastResult = p.parser.parse(t)
	if !p.lastResult.accepting {
		p.lastResult.valid = true
	}

	return p.lastResult
}

func optional(name, optional string) {
	g := &optionalGenerator{}
	g.node = name
	g.optional = optional
	generators[name] = g
}

func (g *optionalGenerator) name() string { return g.node }

func (g *optionalGenerator) canCreate(init node, excluded []string) bool {
	optional, ok := generators[g.optional]
	if !ok {
		panic("generator not found:" + g.optional)
	}

	if stringsContain(excluded, g.name()) {
		return false
	}

	return optional.canCreate(init, excluded)
}

func (g *optionalGenerator) create(path []string, init node, excluded []string) parser {
	return newOptionalParser(path, g.name(), generators[g.optional], init, excluded)
}

func (g *optionalGenerator) member(node string) bool {
	return node == g.name() || node == g.optional
}

func newSequenceParser(path []string, name string, itemGenerator generator, init node, excluded []string) *sequenceParser {
	p := &sequenceParser{}
	p.node = node{typ: name}
	p.init = init
	p.itemGenerator = itemGenerator
	p.p = append(path, name)
	p.excluded = excluded
	p.accepting = true
	return p
}

func (p *sequenceParser) parse(t token) parseResult {
	p.out("parsing", t)

	if !p.accepting {
		p.out("cached result")
		r := p.finalResult
		r.unparsed = append(r.unparsed, t)
		return r
	}

	p.out("finding parser")

	if p.currentParser == nil {
		if len(p.node.nodes) == 0 {
			p.currentParser = p.itemGenerator.create(p.path(), p.init, p.excluded)
		} else {
			p.currentParser = p.itemGenerator.create(p.path(), zeroNode, []string{p.name()})
		}
	}

	itemResult := p.currentParser.parse(t)
	if itemResult.accepting {
		if len(p.queue) > 0 {
			t, p.queue = p.queue[0], p.queue[1:]
			p.out("accepting from queue")
			return p.parse(t)
		}

		p.out("accepting")
		return parseResult{accepting: true}
	}

	p.out("item parse done")

	if !itemResult.valid {
		p.out("invalid")

		p.finalResult = parseResult{
			valid: true,
			node:  p.node,
			unparsed: append(
				itemResult.unparsed,
				p.queue...,
			),
			parsed: p.parsed,
		}
		p.accepting = false
		return p.finalResult
	}

	p.out("valid")

	if !itemResult.node.zero() {
		p.out("has node")
		if len(p.node.nodes) == 0 {
			p.node.token = itemResult.node.token
		}

		p.out("appending node", itemResult.node.typ)
		p.node.nodes = append(p.node.nodes, itemResult.node)
	}

	p.currentParser = nil
	p.parsed += itemResult.parsed
	p.queue = append(itemResult.unparsed, p.queue...)
	if len(p.queue) == 0 {
		return parseResult{
			accepting: true,
		}
	}

	t, p.queue = p.queue[0], p.queue[1:]

	p.out("next from queue")
	return p.parse(t)
}

func sequence(name string, item string) {
	g := &sequenceGenerator{}
	g.node = name
	g.item = item
	generators[name] = g
}

func (g *sequenceGenerator) name() string { return g.node }

func (g *sequenceGenerator) canCreate(init node, excluded []string) bool {
	gen, ok := generators[g.item]
	if !ok {
		panic("generator not found: " + g.item)
	}

	if stringsContain(excluded, g.name()) {
		return false
	}

	return gen.canCreate(init, append(excluded, g.name()))
}

func (g *sequenceGenerator) create(path []string, init node, excluded []string) parser {
	return newSequenceParser(path, g.node, generators[g.item], init, append(excluded, g.name()))
}

func (g *sequenceGenerator) member(node string) bool {
	return node == g.name()
}

func newGroupParser(path []string, name string, generators []generator, init node, excluded []string) *groupParser {
	p := &groupParser{}
	p.node = node{typ: name}
	p.init = init
	p.generators = generators
	p.p = append(path, name)
	p.excluded = excluded
	p.accepting = true
	if !p.init.zero() {
		p.out("initialized with node")
	}

	return p
}

func (p *groupParser) parse(t token) parseResult {
	p.out("parsing", t, p.queue)

	if !p.accepting {
		r := p.finalResult
		r.unparsed = append(r.unparsed, t)
		return r
	}

	if p.currentParser == nil {
		if len(p.generators) == 0 {
			p.out("done")
			p.out("returning", append([]token{t}, p.queue...))
			p.finalResult = parseResult{
				valid:    true,
				node:     p.node,
				unparsed: append([]token{t}, p.queue...),
				parsed: p.parsed,
			}

			p.accepting = false
			return p.finalResult
		}

		if len(p.node.nodes) == 0 {
			p.currentParser = p.generators[0].create(
				p.path(),
				p.init,
				p.excluded,
			)
		} else {
			p.currentParser = p.generators[0].create(p.path(), zeroNode, nil)
		}

		p.generators = p.generators[1:]
	}

	itemResult := p.currentParser.parse(t)
	p.itemAccepted = append(p.itemAccepted, t) // rename to item fed

	if itemResult.accepting {
		if len(p.queue) > 0 {
			t, p.queue = p.queue[0], p.queue[1:]
			p.out("accepting from queue")
			p.out("same item, accepted", len(p.itemAccepted), len(p.accepted))
			return p.parse(t)
		}

		p.out("accepting")
		return parseResult{accepting: true}
	}

	if !itemResult.valid {
		p.itemAccepted = nil
		if len(p.node.nodes) == 0 && !p.init.zero() &&
			generators[p.currentParser.name()].member(p.init.typ) {

			p.out("init item as node")
			p.node.token = p.init.token
			p.node.nodes = append(p.node.nodes, p.init)
			p.currentParser = nil
			p.queue = append(itemResult.unparsed, p.queue...)
			t, p.queue = p.queue[0], p.queue[1:]
			p.out("invalid, accepted", len(p.itemAccepted), len(p.accepted))
			return p.parse(t)
		}

		p.out("invalid")
		p.out(
			"returning rather",
			p.accepted,
			itemResult.unparsed,
			p.queue,
		)
		p.finalResult = parseResult{
			unparsed: append(
				p.accepted,
				append(
					itemResult.unparsed,
					p.queue...,
				)...,
			),
		}
		p.accepting = false
		return p.finalResult
	}

	if !itemResult.node.zero() {
		if len(p.node.nodes) == 0 {
			p.node.token = itemResult.node.token
		}

		p.node.nodes = append(p.node.nodes, itemResult.node)
	}

	// p.itemAccepted = p.itemAccepted[0 : len(p.itemAccepted)-len(itemResult.unparsed)]
	// println(len(p.itemAccepted), itemResult.parsed, itemResult.node.token.String(), p.name())
	p.itemAccepted = p.itemAccepted[:itemResult.parsed]
	p.parsed += len(p.itemAccepted)

	p.currentParser = nil
	p.out(
		"adding to accepted",
		p.accepted,
		p.itemAccepted,
		itemResult.valid,
		itemResult.node.zero(),
		itemResult.unparsed,
	)
	p.accepted = append(p.accepted, p.itemAccepted...)
	p.itemAccepted = nil
	// println(len(p.queue), len(itemResult.unparsed), itemResult.valid, itemResult.parsed)
	p.queue = append(itemResult.unparsed, p.queue...)
	if len(p.queue) == 0 {
		return parseResult{accepting: true}
	}

	t, p.queue = p.queue[0], p.queue[1:]

	p.out("next from queue")
	p.out("valid, accepted", len(p.itemAccepted), len(p.accepted))
	return p.parse(t)
}

func group(name string, items ...string) {
	g := &groupGenerator{}
	g.node = name
	g.items = items
	generators[name] = g
}

func (g *groupGenerator) name() string { return g.node }

func (g *groupGenerator) canCreate(init node, excluded []string) bool {
	if stringsContain(excluded, g.name()) {
		return false
	}

	for _, gi := range g.items {
		if _, ok := generators[gi]; !ok {
			panic("generator not found: " + gi)
		}
	}

	if len(g.items) == 0 {
		return false
	}

	first := g.items[0]
	if generators[first].canCreate(init, append(excluded, g.name())) {
		return true
	}

	if !init.zero() && generators[first].member(init.typ) {
		return true
	}

	return false
}

func (g *groupGenerator) create(path []string, init node, excluded []string) parser {
	gens := make([]generator, len(g.items))
	for i, item := range g.items {
		gens[i] = generators[item]
	}

	return newGroupParser(path, g.node, gens, init, append(excluded, g.name()))
}

func (g *groupGenerator) member(node string) bool {
	return node == g.name()
}

func newUnionParser(path []string, name string, init node, generators []generator, excluded []string) *unionParser {
	p := &unionParser{}
	p.p = append(path, name)
	p.node = init
	p.generators = generators
	p.activeGenerators = generators
	p.excluded = append(excluded, name)
	p.accepting = true

	gs := make([]string, len(p.generators))
	for i, gi := range p.generators {
		gs[i] = gi.name()
	}

	p.out("created", name, gs, p.excluded)
	return p
}

func (p *unionParser) parse(t token) parseResult {
	p.out("parsing", t)

	if !p.accepting {
		r := p.finalResult
		r.unparsed = append(r.unparsed, t)
		return r
	}

	if p.currentParser == nil {
		p.out("excluded", p.excluded)
		for {
			if len(p.activeGenerators) == 0 {
				p.out("finished union, valid:", p.valid)
				p.finalResult = parseResult{
					node:  p.node,
					valid: p.valid,
					unparsed: append(
						[]token{t},
						p.queue...,
					),
					parsed: p.parsed,
				}

				return p.finalResult
			}

			var g generator
			g, p.activeGenerators = p.activeGenerators[0], p.activeGenerators[1:]
			p.out("looking for generator", g.name())
			if g.canCreate(p.node, p.excluded) {
				p.currentParser = g.create(p.path(), p.node, p.excluded)
				break
			}
		}
	}

	p.out("call to parse")
	elementResult := p.currentParser.parse(t)

	if elementResult.accepting {
		p.out("accepting")
		if len(p.queue) > 0 {
			p.out("from queue", p.queue)
			t, p.queue = p.queue[0], p.queue[1:]
			p.out("queue set after accept", p.queue)
			return p.parse(t)
		}

		return parseResult{accepting: true}
	}

	p.out("element parse done")

	p.currentParser = nil

	if !elementResult.valid {
		p.out("invalid union parse", p.valid, elementResult.unparsed, p.queue)
		p.queue = append(elementResult.unparsed, p.queue...)
		p.out("queue set after invalid", p.queue)
		if len(p.queue) > 0 {
			t, p.queue = p.queue[0], p.queue[1:]
			p.out("queue set after taken on invalid", p.queue)
			return p.parse(t)
		}

		return parseResult{accepting: true}
	}

	p.out("valid")
	p.hasAccepted = elementResult.parsed > 0

	if !p.valid || p.hasAccepted {
		p.out("setting valid")
		p.valid = true
		p.node = elementResult.node
		p.parsed = elementResult.parsed
		p.activeGenerators = p.generators
		p.hasAccepted = false
	}

	p.out("a valid union parse", p.valid, elementResult.unparsed, p.queue)
	p.queue = append(elementResult.unparsed, p.queue...)
	p.out("queue set after valid", p.queue)
	if len(p.queue) == 0 {
		p.out("next from outside")
		return parseResult{accepting: true}
	}

	t, p.queue = p.queue[0], p.queue[1:]
	p.out("queue set after taken on valid", p.queue)
	p.out("next from queue")
	return p.parse(t)
}

func union(node string, union ...string) {
	g := &unionGenerator{}
	g.node = node
	g.union = union
	generators[node] = g
}

func (g *unionGenerator) name() string { return g.node }

func (g *unionGenerator) expand(path []string) []string {
	if stringsContain(path, g.name()) {
		panic("union expansion loop")
	}

	var expanded []string
	for _, name := range g.union {
		gi, ok := generators[name]
		if !ok {
			panic("generator not found")
		}

		if u, ok := gi.(*unionGenerator); ok {
			expanded = append(expanded, u.expand(append(path, g.name()))...)
		} else {
			expanded = append(expanded, name)
		}
	}

	return uniq(expanded)
}

func (g *unionGenerator) canCreate(init node, excluded []string) bool {
	if len(g.union) == 0 {
		return false
	}

	expanded := g.expand(nil)
	for _, element := range expanded {
		if generators[element].canCreate(init, excluded) {
			return true
		}
	}

	return false
}

func (g *unionGenerator) create(path []string, init node, excluded []string) parser {
	expanded := g.expand(nil)

	var gens []generator
	for _, element := range expanded {
		gen := generators[element]
		if gen.canCreate(init, excluded) {
			gens = append(gens, gen)
		}
	}

	return newUnionParser(path, g.node, init, gens, excluded)
}

func (g *unionGenerator) member(node string) bool {
	expanded := g.expand(nil)
	for _, gi := range expanded {
		if generators[gi].member(node) {
			return true
		}
	}

	return false
}

func dropSeps(n []node) []node {
	if isSep == nil {
		return n
	}

	nn := make([]node, 0, len(n))
	for _, ni := range n {
		if !isSep(ni) {
			nn = append(nn, ni)
		}
	}

	return nn
}

func postParseNode(n node) node {
	n.nodes = postParseNodes(n.nodes)
	if pp, ok := postParse[n.typ]; ok {
		n = pp(n)
	}

	return n
}

func postParseNodes(n []node) []node {
	n = dropSeps(n)
	for i, ni := range n {
		n[i] = postParseNode(ni)
	}

	return n
}

func parse(p generator, r *tokenReader) (node, error) {
	gi := p.create(nil, zeroNode, nil)
	for {
		t, err := r.next()
		if err != nil && err != io.EOF {
			return node{}, err
		}

		if err == io.EOF {
			gi.out("accepting after eof", token{})
			result := gi.parse(token{})
			if len(result.unparsed) != 1 && result.unparsed[0].typ != noToken {
				// println("unparsed length", len(result.unparsed))
				if len(result.unparsed) > 0 {
					// println(result.unparsed[0].value)
				}
				return node{}, errors.New("unexpected EOF")
			}

			// println("root post-parsing", result.node.typ, len(result.node.nodes))
			// for _, ni := range result.node.nodes {
			// 	println(ni.typ)
			// }

			n := postParseNode(result.node)
			// println("root returning", n.typ, len(n.nodes))
			return n, nil
		}

		// println("root accepting", t.value, t.value == "", len(t.value))
		gi.out("accepting in root", t, t.value == "")
		result := gi.parse(t)
		if len(result.unparsed) > 0 {
			// println("unparsed", len(result.unparsed), t.value)
			// for _, up := range result.unparsed {
			// 	println(up.value)
			// }
			return node{}, fmt.Errorf("unexpected token:%d:%d: %v", t.line, t.column, t)
		}
	}
}
